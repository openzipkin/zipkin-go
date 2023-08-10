// Copyright 2022 The OpenZipkin Authors
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
	"errors"
	"net"
	"testing"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/openzipkin/zipkin-go"
	zipkingrpc "github.com/openzipkin/zipkin-go/middleware/grpc"
	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/propagation/b3"
	service "github.com/openzipkin/zipkin-go/proto/testing"
	"github.com/openzipkin/zipkin-go/reporter/recorder"
)

var (
	serverIDGenerator *sequentialIDGenerator
	serverReporter    *recorder.ReporterRecorder

	server     *grpc.Server
	serverAddr string

	customServer     *grpc.Server
	customServerAddr string
)

func TestGrpc(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Grpc Suite")
}

var _ = ginkgo.BeforeSuite(func() {
	var (
		err       error
		tracer    *zipkin.Tracer
		lis       net.Listener
		customLis net.Listener
	)

	serverReporter = recorder.NewReporter()
	ep, _ := zipkin.NewEndpoint("grpcServer", "")
	serverIDGenerator = newSequentialIDGenerator(0x1000000)

	tracer, err = zipkin.NewTracer(
		serverReporter, zipkin.WithLocalEndpoint(ep),
		zipkin.WithIDGenerator(serverIDGenerator),
		zipkin.WithSharedSpans(false),
	)
	gomega.Expect(tracer, err).ToNot(gomega.BeNil(), "failed to create Zipkin tracer")

	lis, err = net.Listen("tcp", ":0")
	gomega.Expect(lis, err).ToNot(gomega.BeNil(), "failed to listen to tcp port")

	server = grpc.NewServer(grpc.StatsHandler(zipkingrpc.NewServerHandler(tracer)))
	service.RegisterHelloServiceServer(server, &TestHelloService{})
	go func() {
		_ = server.Serve(lis)
	}()
	serverAddr = lis.Addr().String()

	customLis, err = net.Listen("tcp", ":0")
	gomega.Expect(customLis, err).ToNot(gomega.BeNil(), "failed to listen to tcp port")

	tracer, err = zipkin.NewTracer(
		serverReporter,
		zipkin.WithLocalEndpoint(ep),
		zipkin.WithIDGenerator(serverIDGenerator),
		zipkin.WithSharedSpans(true),
	)
	gomega.Expect(tracer, err).ToNot(gomega.BeNil(), "failed to create Zipkin tracer")

	customServer = grpc.NewServer(
		grpc.StatsHandler(
			zipkingrpc.NewServerHandler(
				tracer, zipkingrpc.ServerTags(map[string]string{"default": "tag"}),
			),
		),
	)
	service.RegisterHelloServiceServer(customServer, &TestHelloService{})
	go func() {
		_ = customServer.Serve(customLis)
	}()
	customServerAddr = customLis.Addr().String()
})

var _ = ginkgo.AfterSuite(func() {
	server.Stop()
	customServer.Stop()
	_ = serverReporter.Close()
})

type sequentialIDGenerator struct {
	nextTraceID uint64
	nextSpanID  uint64
	start       uint64
}

func newSequentialIDGenerator(start uint64) *sequentialIDGenerator {
	return &sequentialIDGenerator{start, start, start}
}

func (g *sequentialIDGenerator) SpanID(_ model.TraceID) model.ID {
	id := model.ID(g.nextSpanID)
	g.nextSpanID++
	return id
}

func (g *sequentialIDGenerator) TraceID() model.TraceID {
	id := model.TraceID{
		High: 0,
		Low:  g.nextTraceID,
	}
	g.nextTraceID++
	return id
}

func (g *sequentialIDGenerator) reset() {
	g.nextTraceID = g.start
	g.nextSpanID = g.start
}

type TestHelloService struct {
	service.UnimplementedHelloServiceServer
}

func (s *TestHelloService) Hello(ctx context.Context, req *service.HelloRequest) (*service.HelloResponse, error) {
	if req.Payload == "fail" {
		return nil, status.Error(codes.Aborted, "fail")
	}

	resp := &service.HelloResponse{
		Payload:     "World",
		Metadata:    map[string]string{},
		SpanContext: map[string]string{},
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("could not parse incoming metadata")
	}

	for k := range md {
		// Just append the first value for a key for simplicity since we don't use multi-value headers.
		resp.GetMetadata()[k] = md[k][0]
	}

	span := zipkin.SpanFromContext(ctx)
	if span != nil {
		spanCtx := span.Context()
		resp.GetSpanContext()[b3.SpanID] = spanCtx.ID.String()
		resp.GetSpanContext()[b3.TraceID] = spanCtx.TraceID.String()
		if spanCtx.ParentID != nil {
			resp.GetSpanContext()[b3.ParentSpanID] = spanCtx.ParentID.String()
		}
	}

	return resp, nil
}
