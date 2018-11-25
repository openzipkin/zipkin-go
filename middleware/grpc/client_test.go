// +build go1.9

package grpc_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/stats"

	"github.com/openzipkin/zipkin-go"
	. "github.com/openzipkin/zipkin-go/middleware/grpc"
	"github.com/openzipkin/zipkin-go/propagation/b3"
	service "github.com/openzipkin/zipkin-go/proto/testing"
	"github.com/openzipkin/zipkin-go/reporter/recorder"
)

var _ = Describe("gRPC Client", func() {
	var (
		reporter *recorder.ReporterRecorder
		tracer   *zipkin.Tracer
		conn     *grpc.ClientConn
		client   service.HelloServiceClient
	)

	BeforeEach(func() {
		var err error

		reporter = recorder.NewReporter()
		ep, _ := zipkin.NewEndpoint("grpcClient", "")
		tracer, err = zipkin.NewTracer(
			reporter, zipkin.WithLocalEndpoint(ep), zipkin.WithIDGenerator(newSequentialIdGenerator()))
		Expect(tracer, err).ToNot(BeNil())
	})

	AfterEach(func() {
		_ = reporter.Close()
		_ = conn.Close()
	})

	Context("with defaults", func() {
		BeforeEach(func() {
			var err error

			conn, err = grpc.Dial(serverAddr, grpc.WithInsecure(), grpc.WithStatsHandler(NewClientHandler(tracer)))
			Expect(conn, err).ToNot(BeNil())
			client = service.NewHelloServiceClient(conn)
		})

		It("creates a span", func() {
			resp, err := client.Hello(context.Background(), &service.HelloRequest{Payload: "Hello"})
			Expect(resp, err).ToNot(BeNil())

			spans := reporter.Flush()
			Expect(spans).To(HaveLen(1))
			Expect(spans[0].Name).To(Equal("zipkin.testing.HelloService.Hello"))
			Expect(spans[0].Tags).To(BeEmpty())
		})

		It("propagates trace context", func() {
			resp, err := client.Hello(context.Background(), &service.HelloRequest{Payload: "Hello"})
			Expect(resp.GetMetadata(), err).To(HaveKeyWithValue(b3.TraceID, "0000000000000001"))
			Expect(resp.GetMetadata(), err).To(HaveKeyWithValue(b3.SpanID, "0000000000000001"))
			Expect(resp.GetMetadata(), err).ToNot(HaveKey(b3.ParentSpanID))
		})

		It("propagates parent span", func() {
			_, ctx := tracer.StartSpanFromContext(context.Background(), "parent")
			resp, err := client.Hello(ctx, &service.HelloRequest{Payload: "Hello"})
			Expect(resp.GetMetadata(), err).To(HaveKeyWithValue(b3.TraceID, "0000000000000001"))
			Expect(resp.GetMetadata(), err).To(HaveKeyWithValue(b3.SpanID, "0000000000000002"))
			Expect(resp.GetMetadata(), err).To(HaveKeyWithValue(b3.ParentSpanID, "0000000000000001"))
		})

		It("tags with error code", func() {
			_, err := client.Hello(context.Background(), &service.HelloRequest{Payload: "fail"})
			Expect(err).To(HaveOccurred())

			spans := reporter.Flush()
			Expect(spans).To(HaveLen(1))
			Expect(spans[0].Tags).To(HaveLen(2))
			Expect(spans[0].Tags).To(HaveKeyWithValue("grpc.status_code", codes.Aborted.String()))
			Expect(spans[0].Tags).To(HaveKeyWithValue(string(zipkin.TagError), codes.Aborted.String()))
		})
	})

	Context("with custom RPCHandler", func() {
		BeforeEach(func() {
			var err error

			conn, err = grpc.Dial(
				serverAddr,
				grpc.WithInsecure(),
				grpc.WithStatsHandler(NewClientHandler(tracer, WithRPCHandler(func(span zipkin.Span, rpcStats stats.RPCStats) {
					span.Tag("custom", "tag")
				}))))
			Expect(conn, err).ToNot(BeNil())
			client = service.NewHelloServiceClient(conn)
		})

		It("calls custom RPCHandler", func() {
			resp, err := client.Hello(context.Background(), &service.HelloRequest{Payload: "Hello"})
			Expect(resp, err).ToNot(BeNil())

			spans := reporter.Flush()
			Expect(spans).To(HaveLen(1))
			Expect(spans[0].Tags).To(HaveLen(1))
			Expect(spans[0].Tags).To(HaveKeyWithValue("custom", "tag"))
		})
	})
})
