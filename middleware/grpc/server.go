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

package grpc

import (
	"context"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/stats"

	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/propagation/b3"
)

type serverHandler struct {
	tracer      *zipkin.Tracer
	defaultTags map[string]string
	baggage     model.Baggage
}

// A ServerOption can be passed to NewServerHandler to customize the returned handler.
type ServerOption func(*serverHandler)

// ServerTags adds default Tags to inject into server spans.
func ServerTags(tags map[string]string) ServerOption {
	return func(h *serverHandler) {
		h.defaultTags = tags
	}
}

// EnableBaggage can be passed to NewServerHandler to enable propagation of
// whitelisted headers through the SpanContext object.
func EnableBaggage(b model.Baggage) ServerOption {
	return func(h *serverHandler) {
		h.baggage = b
	}
}

// NewServerHandler returns a stats.Handler which can be used with grpc.WithStatsHandler to add
// tracing to a gRPC server. The gRPC method name is used as the span name and by default the only
// tags are the gRPC status code if the call fails. Use ServerTags to add additional tags that
// should be applied to all spans.
func NewServerHandler(tracer *zipkin.Tracer, options ...ServerOption) stats.Handler {
	c := &serverHandler{
		tracer: tracer,
	}
	for _, option := range options {
		option(c)
	}
	return c
}

// HandleConn exists to satisfy gRPC stats.Handler.
func (s *serverHandler) HandleConn(_ context.Context, _ stats.ConnStats) {
	// no-op
}

// TagConn exists to satisfy gRPC stats.Handler.
func (s *serverHandler) TagConn(ctx context.Context, _ *stats.ConnTagInfo) context.Context {
	// no-op
	return ctx
}

// HandleRPC implements per-RPC tracing and stats instrumentation.
func (s *serverHandler) HandleRPC(ctx context.Context, rs stats.RPCStats) {
	handleRPC(ctx, rs)
}

// TagRPC implements per-RPC context management.
func (s *serverHandler) TagRPC(ctx context.Context, rti *stats.RPCTagInfo) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	// In practice, ok never seems to be false but add a defensive check.
	if !ok {
		md = metadata.New(nil)
	}

	name := spanName(rti)

	sc := s.tracer.Extract(b3.ExtractGRPC(&md))

	// store whitelisted headers to be propagated in spanContext
	if s.baggage != nil {
		sc.Baggage = s.baggage.Init()
		sc.Baggage.IterateWhiteList(func(key string) {
			vals := md.Get(key)
			if len(vals) > 0 {
				sc.Baggage.AddHeader(key, vals...)
			}
		})
	}

	span := s.tracer.StartSpan(
		name,
		zipkin.Kind(model.Server),
		zipkin.Parent(sc),
		zipkin.RemoteEndpoint(remoteEndpointFromContext(ctx, "")),
	)

	if !zipkin.IsNoop(span) {
		for k, v := range s.defaultTags {
			span.Tag(k, v)
		}
	}

	return zipkin.NewContext(ctx, span)
}
