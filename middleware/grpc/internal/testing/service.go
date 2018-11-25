// +build go1.9

package testing

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	service "github.com/openzipkin/zipkin-go/proto/testing"
)

type TestHelloService struct{}

func (s *TestHelloService) Hello(ctx context.Context, req *service.HelloRequest) (*service.HelloResponse, error) {
	if req.Payload == "fail" {
		return nil, status.Error(codes.Aborted, "fail")
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("could not parse incoming metadata")
	}

	resp := &service.HelloResponse{
		Payload: "World",
		Metadata: map[string]string{},
	}

	for k, _ := range md {
		// Just append the first value for a key for simplicity since we don't use multi-value headers.
		resp.GetMetadata()[k] = md[k][0]
	}

	return resp, nil
}
