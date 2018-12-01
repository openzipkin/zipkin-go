package grpc_test

import (
	"context"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/propagation/b3"
	service "github.com/openzipkin/zipkin-go/proto/testing"
	"github.com/openzipkin/zipkin-go/reporter"
)

var _ = ginkgo.Describe("gRPC Server", func() {
	var (
		conn   *grpc.ClientConn
		client service.HelloServiceClient
	)

	ginkgo.BeforeEach(func() {
		serverIdGenerator.reset()
		serverReporter.Flush()
	})

	ginkgo.Context("with defaults", func() {
		ginkgo.BeforeEach(func() {
			var err error

			conn, err = grpc.Dial(serverAddr, grpc.WithInsecure())
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			client = service.NewHelloServiceClient(conn)
		})

		ginkgo.It("creates a span and context", func() {
			resp, err := client.Hello(context.Background(), &service.HelloRequest{Payload: "Hello"})
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			spans := serverReporter.Flush()
			gomega.Expect(spans).To(gomega.HaveLen(1))

			span := spans[0]
			gomega.Expect(span.Kind).To(gomega.Equal(model.Server))
			// Set to local host for tests, might be IPv4 or IPv6 not worth checking the actual address.
			gomega.Expect(span.RemoteEndpoint.Empty()).To(gomega.BeFalse())
			gomega.Expect(span.Name).To(gomega.Equal("zipkin.testing.HelloService.Hello"))
			gomega.Expect(span.Tags).To(gomega.BeEmpty())

			spanCtx := resp.GetSpanContext()
			gomega.Expect(spanCtx).To(gomega.HaveLen(2))
			gomega.Expect(spanCtx).To(gomega.HaveKeyWithValue(b3.TraceID, "0000000001000000"))
			gomega.Expect(spanCtx).To(gomega.HaveKeyWithValue(b3.SpanID, "0000000001000000"))
		})

		ginkgo.It("propagates parent", func() {
			// Manually create a client context
			tracer, err := zipkin.NewTracer(
				reporter.NewNoopReporter(),
				zipkin.WithIDGenerator(newSequentialIdGenerator(1)))
			testSpan := tracer.StartSpan("test")
			md := metadata.New(nil)
			_ = b3.InjectGRPC(&md)(testSpan.Context())
			ctx := metadata.NewOutgoingContext(context.Background(), md)

			resp, err := client.Hello(ctx, &service.HelloRequest{Payload: "Hello"})
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			spans := serverReporter.Flush()
			gomega.Expect(spans).To(gomega.HaveLen(1))

			span := spans[0]
			gomega.Expect(span.Kind).To(gomega.Equal(model.Server))
			gomega.Expect(span.RemoteEndpoint.Empty()).To(gomega.BeFalse())
			gomega.Expect(span.Name).To(gomega.Equal("zipkin.testing.HelloService.Hello"))
			gomega.Expect(span.Tags).To(gomega.BeEmpty())

			spanCtx := resp.GetSpanContext()
			gomega.Expect(spanCtx).To(gomega.HaveLen(3))
			gomega.Expect(spanCtx).To(gomega.HaveKeyWithValue(b3.SpanID, "0000000001000000"))
			gomega.Expect(spanCtx).To(gomega.HaveKeyWithValue(b3.TraceID, "0000000000000001"))
			gomega.Expect(spanCtx).To(gomega.HaveKeyWithValue(b3.ParentSpanID, "0000000000000001"))
		})

		ginkgo.It("tags with error code", func() {
			_, err := client.Hello(context.Background(), &service.HelloRequest{Payload: "fail"})
			gomega.Expect(err).To(gomega.HaveOccurred())

			spans := serverReporter.Flush()
			gomega.Expect(spans).To(gomega.HaveLen(1))
			gomega.Expect(spans[0].Tags).To(gomega.HaveLen(2))
			gomega.Expect(spans[0].Tags).To(gomega.HaveKeyWithValue("grpc.status_code", "ABORTED"))
			gomega.Expect(spans[0].Tags).To(gomega.HaveKeyWithValue(string(zipkin.TagError), "ABORTED"))
		})
	})

	ginkgo.Context("with joined spans and custom handler", func() {
		ginkgo.BeforeEach(func() {
			var err error

			conn, err = grpc.Dial(customServerAddr, grpc.WithInsecure())
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			client = service.NewHelloServiceClient(conn)
		})

		ginkgo.It("calls custom handler", func() {
			resp, err := client.Hello(context.Background(), &service.HelloRequest{Payload: "Hello"})
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			spans := serverReporter.Flush()
			gomega.Expect(spans).To(gomega.HaveLen(1))

			span := spans[0]
			gomega.Expect(span.RemoteEndpoint.Empty()).To(gomega.BeFalse())
			gomega.Expect(span.Tags).To(gomega.HaveLen(1))
			gomega.Expect(span.Tags).To(gomega.HaveKeyWithValue("custom", "tag"))

			spanCtx := resp.GetSpanContext()
			gomega.Expect(spanCtx).To(gomega.HaveLen(2))
			gomega.Expect(spanCtx).To(gomega.HaveKeyWithValue(b3.TraceID, "0000000001000000"))
			gomega.Expect(spanCtx).To(gomega.HaveKeyWithValue(b3.SpanID, "0000000001000000"))
		})

		ginkgo.It("joins with caller", func() {
			// Manually create a client context
			tracer, err := zipkin.NewTracer(
				reporter.NewNoopReporter(),
				zipkin.WithIDGenerator(newSequentialIdGenerator(1)))
			testSpan := tracer.StartSpan("test")
			md := metadata.New(nil)
			_ = b3.InjectGRPC(&md)(testSpan.Context())
			ctx := metadata.NewOutgoingContext(context.Background(), md)

			resp, err := client.Hello(ctx, &service.HelloRequest{Payload: "Hello"})
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			spanCtx := resp.GetSpanContext()
			gomega.Expect(spanCtx).To(gomega.HaveLen(2))
			gomega.Expect(spanCtx).To(gomega.HaveKeyWithValue(b3.TraceID, "0000000000000001"))
			gomega.Expect(spanCtx).To(gomega.HaveKeyWithValue(b3.SpanID, "0000000000000001"))
		})
	})
})
