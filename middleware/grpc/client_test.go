// Copyright 2021 The OpenZipkin Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package grpc_test

import (
	"context"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/openzipkin/zipkin-go"
	zipkingrpc "github.com/openzipkin/zipkin-go/middleware/grpc"
	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/propagation/b3"
	service "github.com/openzipkin/zipkin-go/proto/testing"
	"github.com/openzipkin/zipkin-go/reporter/recorder"
)

var _ = ginkgo.Describe("gRPC Client", func() {
	var (
		reporter *recorder.ReporterRecorder
		tracer   *zipkin.Tracer
		conn     *grpc.ClientConn
		client   service.HelloServiceClient
	)

	ginkgo.BeforeEach(func() {
		var err error

		reporter = recorder.NewReporter()
		ep, _ := zipkin.NewEndpoint("grpcClient", "")
		tracer, err = zipkin.NewTracer(
			reporter, zipkin.WithLocalEndpoint(ep), zipkin.WithIDGenerator(newSequentialIdGenerator(1)))
		gomega.Expect(tracer, err).ToNot(gomega.BeNil())
	})

	ginkgo.AfterEach(func() {
		_ = reporter.Close()
		_ = conn.Close()
	})

	ginkgo.Context("with defaults", func() {
		ginkgo.BeforeEach(func() {
			var err error

			conn, err = grpc.Dial(serverAddr, grpc.WithInsecure(), grpc.WithStatsHandler(zipkingrpc.NewClientHandler(tracer)))
			gomega.Expect(conn, err).ToNot(gomega.BeNil())
			client = service.NewHelloServiceClient(conn)
		})

		ginkgo.It("creates a span", func() {
			resp, err := client.Hello(context.Background(), &service.HelloRequest{Payload: "Hello"})
			gomega.Expect(resp, err).ToNot(gomega.BeNil())

			spans := reporter.Flush()
			gomega.Expect(spans).To(gomega.HaveLen(1))

			span := spans[0]
			gomega.Expect(span.Kind).To(gomega.Equal(model.Client))
			gomega.Expect(span.Name).To(gomega.Equal("zipkin.testing.HelloService.Hello"))
			gomega.Expect(span.RemoteEndpoint).To(gomega.BeNil())
			gomega.Expect(span.Tags).To(gomega.BeEmpty())
		})

		ginkgo.It("propagates trace context", func() {
			resp, err := client.Hello(context.Background(), &service.HelloRequest{Payload: "Hello"})
			gomega.Expect(resp.GetMetadata(), err).To(gomega.HaveKeyWithValue(b3.TraceID, "0000000000000001"))
			gomega.Expect(resp.GetMetadata(), err).To(gomega.HaveKeyWithValue(b3.SpanID, "0000000000000001"))
			gomega.Expect(resp.GetMetadata(), err).ToNot(gomega.HaveKey(b3.ParentSpanID))
		})

		ginkgo.It("propagates parent span", func() {
			_, ctx := tracer.StartSpanFromContext(context.Background(), "parent")
			resp, err := client.Hello(ctx, &service.HelloRequest{Payload: "Hello"})
			gomega.Expect(resp.GetMetadata(), err).To(gomega.HaveKeyWithValue(b3.TraceID, "0000000000000001"))
			gomega.Expect(resp.GetMetadata(), err).To(gomega.HaveKeyWithValue(b3.SpanID, "0000000000000002"))
			gomega.Expect(resp.GetMetadata(), err).To(gomega.HaveKeyWithValue(b3.ParentSpanID, "0000000000000001"))
		})

		ginkgo.It("tags with error code", func() {
			_, err := client.Hello(context.Background(), &service.HelloRequest{Payload: "fail"})
			gomega.Expect(err).To(gomega.HaveOccurred())

			spans := reporter.Flush()
			gomega.Expect(spans).To(gomega.HaveLen(1))
			gomega.Expect(spans[0].Tags).To(gomega.HaveLen(2))
			gomega.Expect(spans[0].Tags).To(gomega.HaveKeyWithValue("grpc.status_code", "ABORTED"))
			gomega.Expect(spans[0].Tags).To(gomega.HaveKeyWithValue(string(zipkin.TagError), "ABORTED"))
		})

		ginkgo.It("copies existing metadata", func() {
			ctx := metadata.AppendToOutgoingContext(context.Background(), "existing", "metadata")
			resp, err := client.Hello(ctx, &service.HelloRequest{Payload: "Hello"})

			gomega.Expect(resp.GetMetadata(), err).To(gomega.HaveKeyWithValue("existing", "metadata"))
		})
	})

	ginkgo.Context("with remote service name", func() {
		ginkgo.BeforeEach(func() {
			var err error

			conn, err = grpc.Dial(
				serverAddr,
				grpc.WithInsecure(),
				grpc.WithStatsHandler(zipkingrpc.NewClientHandler(
					tracer,
					zipkingrpc.WithRemoteServiceName("remoteService"))))
			gomega.Expect(conn, err).ToNot(gomega.BeNil())
			client = service.NewHelloServiceClient(conn)
		})

		ginkgo.It("has remote service name", func() {
			resp, err := client.Hello(context.Background(), &service.HelloRequest{Payload: "Hello"})
			gomega.Expect(resp, err).ToNot(gomega.BeNil())

			spans := reporter.Flush()
			gomega.Expect(spans).To(gomega.HaveLen(1))
			gomega.Expect(spans[0].RemoteEndpoint.ServiceName).To(gomega.Equal("remoteService"))
		})
	})
})
