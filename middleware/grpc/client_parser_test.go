// Copyright 2020 The OpenZipkin Authors
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
	"testing"

	"github.com/openzipkin/zipkin-go"
	zipkingrpc "github.com/openzipkin/zipkin-go/middleware/grpc"
	service "github.com/openzipkin/zipkin-go/proto/testing"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/stats"
)

func TestGRPCClientCanAccessToPayloadAndMetadata(t *testing.T) {
	tracer, flusher := createTracer(false)

	s := grpc.NewServer()
	defer s.Stop()

	service.RegisterHelloServiceServer(s, &TestHelloService{
		responseHeader:  metadata.Pairs("test_key", "test_value_1"),
		responseTrailer: metadata.Pairs("test_key", "test_value_2"),
	})

	dialer := initListener(s)

	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx,
		"bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithInsecure(),
		grpc.WithStatsHandler(zipkingrpc.NewClientHandler(
			tracer,
			zipkingrpc.WithClientOutPayloadParser(func(outPayload *stats.OutPayload, span zipkin.SpanCustomizer) {
				m, ok := outPayload.Payload.(*service.HelloRequest)
				if !ok {
					t.Fatal("failed to cast the payload as a service.HelloResponse")
				}
				if want, have := "Hello", m.Payload; want != have {
					t.Errorf("incorrect payload: want %q, have %q", want, have)
				}
			}),
			zipkingrpc.WithClientOutHeaderParser(func(outHeader *stats.OutHeader, span zipkin.SpanCustomizer) {
				if want, have := "test_value", outHeader.Header.Get("test_key")[0]; want != have {
					t.Errorf("incorrect header value, want %q, have %q", want, have)
				}
			}),
			zipkingrpc.WithClientInPayloadParser(func(inPayload *stats.InPayload, span zipkin.SpanCustomizer) {
				m, ok := inPayload.Payload.(*service.HelloResponse)
				if !ok {
					t.Fatal("failed to cast the payload as a service.HelloRequest")
				}
				if want, have := "World", m.Payload; want != have {
					t.Errorf("incorrect payload: want %q, have %q", want, have)
				}
			}),
			zipkingrpc.WithClientInHeaderParser(func(inHeader *stats.InHeader, span zipkin.SpanCustomizer) {
				if want, have := "test_value_1", inHeader.Header.Get("test_key")[0]; want != have {
					t.Errorf("incorrect header value, want %q, have %q", want, have)
				}
			}),
			zipkingrpc.WithClientInTrailerParser(func(inTrailer *stats.InTrailer, span zipkin.SpanCustomizer) {
				if want, have := "test_value_2", inTrailer.Trailer.Get("test_key")[0]; want != have {
					t.Errorf("incorrect header value, want %q, have %q", want, have)
				}
			}),
		)),
	)

	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	client := service.NewHelloServiceClient(conn)

	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("test_key", "test_value"))
	_, err = client.Hello(ctx, &service.HelloRequest{
		Payload: "Hello",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spans := flusher()
	if want, have := 1, len(spans); want != have {
		t.Errorf("unexpected number of spans, want %d, have %d", want, have)
	}
}
