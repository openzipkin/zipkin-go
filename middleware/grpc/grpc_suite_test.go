// +build go1.9

package grpc_test

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"

	"github.com/openzipkin/zipkin-go"
	zipkingrpc "github.com/openzipkin/zipkin-go/middleware/grpc"
	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/propagation/b3"
	service "github.com/openzipkin/zipkin-go/proto/testing"
	"github.com/openzipkin/zipkin-go/reporter/recorder"
)

var (
	serverIdGenerator *sequentialIdGenerator
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
	var err error

	serverReporter = recorder.NewReporter()
	ep, _ := zipkin.NewEndpoint("grpcServer", "")
	serverIdGenerator = newSequentialIdGenerator(0x1000000)
	tracer, err := zipkin.NewTracer(
		serverReporter, zipkin.WithLocalEndpoint(ep), zipkin.WithIDGenerator(serverIdGenerator), zipkin.WithSharedSpans(false))

	lis, err := net.Listen("tcp", ":0")
	gomega.Expect(lis, err).ToNot(gomega.BeNil(), "failed to listen to tcp port")

	server = grpc.NewServer(grpc.StatsHandler(zipkingrpc.NewServerHandler(tracer)))
	service.RegisterHelloServiceServer(server, &TestHelloService{})
	go func() {
		_ = server.Serve(lis)
	}()
	serverAddr = lis.Addr().String()

	lis, err = net.Listen("tcp", ":0")
	gomega.Expect(lis, err).ToNot(gomega.BeNil(), "failed to listen to tcp port")

	tracer, err = zipkin.NewTracer(
		serverReporter, zipkin.WithLocalEndpoint(ep), zipkin.WithIDGenerator(serverIdGenerator), zipkin.WithSharedSpans(true))
	customServer = grpc.NewServer(grpc.StatsHandler(zipkingrpc.NewServerHandler(
		tracer,
		zipkingrpc.WithServerRPCHandler(func(span zipkin.Span, rpcStats stats.RPCStats) {
			span.Tag("custom", "tag")
		}))))
	service.RegisterHelloServiceServer(customServer, &TestHelloService{})
	go func() {
		_ = customServer.Serve(lis)
	}()
	customServerAddr = lis.Addr().String()
})

var _ = ginkgo.AfterSuite(func() {
	server.Stop()
})

type sequentialIdGenerator struct {
	nextTraceId uint64
	nextSpanId  uint64
	start       uint64
}

func newSequentialIdGenerator(start uint64) *sequentialIdGenerator {
	return &sequentialIdGenerator{start, start, start}
}

func (g *sequentialIdGenerator) SpanID(traceID model.TraceID) model.ID {
	id := model.ID(g.nextSpanId)
	g.nextSpanId++
	return id
}

func (g *sequentialIdGenerator) TraceID() model.TraceID {
	id := model.TraceID{
		High: 0,
		Low:  g.nextTraceId,
	}
	g.nextTraceId++
	return id
}

func (g *sequentialIdGenerator) reset() {
	g.nextTraceId = g.start
	g.nextSpanId = g.start
}

type TestHelloService struct{}

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
