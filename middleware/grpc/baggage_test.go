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

	"github.com/openzipkin/zipkin-go/idgenerator"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/openzipkin/zipkin-go"
	zipkingrpc "github.com/openzipkin/zipkin-go/middleware/grpc"
	"github.com/openzipkin/zipkin-go/propagation/baggage"
	service "github.com/openzipkin/zipkin-go/proto/testing"
)

const ReqID = "x-request-id"

func TestGRPCBaggage(t *testing.T) {
	var (
		tracer, _     = zipkin.NewTracer(nil)
		bag           = baggage.New(ReqID)
		serverHandler = zipkingrpc.NewServerHandler(tracer, zipkingrpc.EnableBaggage(bag))
		srv           = grpc.NewServer(grpc.StatsHandler(serverHandler))
		bSrv          = &baggageServer{}
	)

	// create listener
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("unable to create listener for grpc server: %+v", err)
	}
	defer func() {
		_ = ln.Close()
	}()

	// create client
	cc, err := grpc.Dial(
		ln.Addr().String(),
		grpc.WithInsecure(),
		grpc.WithStatsHandler(zipkingrpc.NewClientHandler(tracer)),
	)
	if err != nil {
		t.Fatalf("unable to create connection for grpc client: %+v", err)
	}
	bSrv.client = service.NewBaggageServiceClient(cc)

	// start grpc server
	go func() {
		service.RegisterBaggageServiceServer(srv, bSrv)
		_ = srv.Serve(ln)
	}()

	// generate request to handler1 with x-request-id set
	reqID := idgenerator.NewRandom128().TraceID().String()
	md := metadata.New(nil)
	md.Set(ReqID, reqID)
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	if _, err = bSrv.client.Handler1(ctx, &emptypb.Empty{}); err != nil {
		t.Fatalf("unexpected grpc request error: %+v", err)
	}

	// check server inspection for request id propagation
	if bSrv.h1 != reqID {
		t.Errorf("h1 expected propagated %s: want %s, have: %s", ReqID, reqID, bSrv.h1)
	}
	if bSrv.h2 != reqID {
		t.Errorf("h2 expected propagated %s: want %s, have: %s", ReqID, reqID, bSrv.h2)
	}
}

type baggageServer struct {
	service.UnimplementedBaggageServiceServer
	client service.BaggageServiceClient
	h1, h2 string
}

func (b *baggageServer) Handler1(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get(ReqID); len(vals) > 0 {
			b.h1 = vals[0]
		}
	}
	if _, err := b.client.Handler2(ctx, &emptypb.Empty{}); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (b *baggageServer) Handler2(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get(ReqID); len(vals) > 0 {
			b.h2 = vals[0]
		}
	}
	return &emptypb.Empty{}, nil
}

var _ service.BaggageServiceServer = (*baggageServer)(nil)
