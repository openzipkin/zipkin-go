package grpc

import (
	"context"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/reporter/recorder"
	"net"
	"testing"

	"google.golang.org/grpc"

	service "github.com/openzipkin/zipkin-go/proto/testing"
)

type testHelloService struct{}

func (s *testHelloService) Hello(context.Context, *service.HelloRequest) (*service.HelloResponse, error) {
	return &service.HelloResponse{
		Payload: "World",
	}, nil
}

func TestGRPCClient(t *testing.T) {
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	service.RegisterHelloServiceServer(grpcServer, &testHelloService{})
	go func() {
		grpcServer.Serve(lis)
	}()
	defer grpcServer.Stop()

	reporter := recorder.NewReporter()
	defer reporter.Close()

	ep, _ := zipkin.NewEndpoint("httpClient", "")
	tracer, err := zipkin.NewTracer(reporter, zipkin.WithLocalEndpoint(ep))
	if err != nil {
		t.Fatalf("unable to create tracer: %+v", err)
	}

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure(), grpc.WithStatsHandler(NewClientHandler(tracer)))
	if err != nil {
		t.Fatalf("Could not connect to gRPC server: %v", err)
	}
	defer conn.Close()

	client := service.NewHelloServiceClient(conn)

	_, err = client.Hello(context.Background(), &service.HelloRequest{Payload: "Hello"})
	if err != nil {
		t.Fatalf("Error from gRPC server: %v", err)
	}

	spans := reporter.Flush()
	if len(spans) == 0 {
		t.Errorf("Span Count want 1+, have 0")
	}

	span := tracer.StartSpan("ParentSpan")
	ctx := zipkin.NewContext(context.Background(), span)
	client.Hello(ctx, &service.HelloRequest{Payload: "Hello"})

	spans = reporter.Flush()
	if len(spans) == 0 {
		t.Errorf("Span Count want 1+, have 0")
	}
}
