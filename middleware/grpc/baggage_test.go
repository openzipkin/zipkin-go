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
	"net"
	"testing"

	"github.com/openzipkin/zipkin-go/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/openzipkin/zipkin-go"
	zgrpc "github.com/openzipkin/zipkin-go/middleware/grpc"
	"github.com/openzipkin/zipkin-go/propagation/baggage"
	service "github.com/openzipkin/zipkin-go/proto/testing"
)

const (
	reqID            = "x-request-id"
	reqIDValue       = "5a3553a7-4088-4ae0-8845-8314ebd59ddb"
	customField      = "custom-field"
	customFieldValue = "custom-value"
)

var tracer *zipkin.Tracer

// TestGRPCBaggage tests baggage propagation through actual gRPC client -
// server connections. It creates a client which will inject an x-request-id
// header which will trigger the receiving server to retrieve the incoming value
// on the handler1 endpoint, propagate and use it in an outgoing call to the
// handler2 endpoint, which should also retrieve the incoming value.
// By doing this we test:
// - outgoing baggage on client side (stand-alone client)
// - incoming baggage on server side (handler1 endpoint)
// - in process baggage propagation on server side (handler1 implementation)
// - add additional header in handler1 implementation
// - incoming baggage on server side (handler2 endpoint)
func TestGRPCBaggage(t *testing.T) {
	tracer, _ = zipkin.NewTracer(nil)

	var bagHandler = baggage.New(reqID, customField)

	// create listener for server to use
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("unable to create listener for grpc server: %+v", err)
	}
	defer func() {
		_ = ln.Close()
	}()

	// create gRPC client
	client := newClient(t, ln.Addr().String())

	// start gRPC server using the provided listener, gRPC client and baggage
	// handler
	bSrv := runServer(ln, client, bagHandler)

	// set x-request-id using a UUID as value
	md := metadata.New(nil)
	md.Set(reqID, reqIDValue)

	ctx := metadata.NewOutgoingContext(context.Background(), md)

	// call gRPC server Handler1 method
	if _, err = client.Handler1(ctx, &emptypb.Empty{}); err != nil {
		t.Fatalf("unexpected grpc request error: %+v", err)
	}

	// check server inspection variables for correct baggage field propagation
	if bSrv.resultHandler1 != reqIDValue {
		t.Errorf("resultHandler1 expected propagated %s: want %s, have: %s",
			reqID, reqIDValue, bSrv.resultHandler1)
	}
	if bSrv.result1Handler2 != reqIDValue {
		t.Errorf("result1Handler2 expected propagated %s: want %s, have: %s",
			reqID, reqIDValue, bSrv.result1Handler2)
	}
	if bSrv.result2Handler2 != customFieldValue {
		t.Errorf("result2Handler2 expected propagated %s: want %s, have: %s",
			customField, customFieldValue, bSrv.result2Handler2)
	}
}

func runServer(
	ln net.Listener, // listener to use
	client service.BaggageServiceClient, // the server can call itself
	bagHandler middleware.BaggageHandler, // baggage handler to use
) *baggageServer {
	var (
		zHnd = zgrpc.NewServerHandler(tracer, zgrpc.EnableBaggage(bagHandler))
		gSrv = grpc.NewServer(grpc.StatsHandler(zHnd))
		bSrv = &baggageServer{client: client}
	)
	service.RegisterBaggageServiceServer(gSrv, bSrv)
	go func() {
		_ = gSrv.Serve(ln)
	}()
	return bSrv
}

func newClient(t *testing.T, serverAddr string) service.BaggageServiceClient {
	zHnd := zgrpc.NewClientHandler(tracer)
	cc, err := grpc.Dial(
		serverAddr,
		grpc.WithInsecure(),
		grpc.WithStatsHandler(zHnd),
	)
	if err != nil {
		t.Fatalf("unable to create connection for grpc client: %+v", err)
	}
	return service.NewBaggageServiceClient(cc)
}

type baggageServer struct {
	service.UnimplementedBaggageServiceServer
	client          service.BaggageServiceClient
	resultHandler1  string
	result1Handler2 string
	result2Handler2 string
}

func (b *baggageServer) Handler1(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	// retrieve received value from incoming context
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get(reqID); len(values) > 0 {
			b.resultHandler1 = values[0]
		}
	}
	// add additional baggage field
	if span := zipkin.SpanFromContext(ctx); span != nil {
		span.Context().Baggage.Add(customField, customFieldValue)
	}
	// outgoing call from client uses baggage found in context
	if _, err := b.client.Handler2(ctx, &emptypb.Empty{}); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (b *baggageServer) Handler2(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	// retrieve received value from incoming context
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get(reqID); len(values) > 0 {
			b.result1Handler2 = values[0]
		}
		if values := md.Get(customField); len(values) > 0 {
			b.result2Handler2 = values[0]
		}
	}
	return &emptypb.Empty{}, nil
}

var _ service.BaggageServiceServer = (*baggageServer)(nil)
