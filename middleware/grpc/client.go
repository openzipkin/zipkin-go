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

package grpc

import (
	"context"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/stats"

	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/propagation/b3"
)

type clientHandler struct {
	tracer            *zipkin.Tracer
	remoteServiceName string
}

// A ClientOption can be passed to NewClientHandler to customize the returned handler.
type ClientOption func(*clientHandler)

// WithRemoteServiceName will set the value for the remote endpoint's service name on
// all spans.
func WithRemoteServiceName(name string) ClientOption {
	return func(c *clientHandler) {
		c.remoteServiceName = name
	}
}

// NewClientHandler returns a stats.Handler which can be used with grpc.WithStatsHandler to add
// tracing to a gRPC client. The gRPC method name is used as the span name and by default the only
// tags are the gRPC status code if the call fails.
func NewClientHandler(tracer *zipkin.Tracer, options ...ClientOption) stats.Handler {
	c := &clientHandler{
		tracer: tracer,
	}
	for _, option := range options {
		option(c)
	}
	return c
}

// HandleConn exists to satisfy gRPC stats.Handler.
func (c *clientHandler) HandleConn(_ context.Context, _ stats.ConnStats) {
	// no-op
}

// TagConn exists to satisfy gRPC stats.Handler.
func (c *clientHandler) TagConn(ctx context.Context, _ *stats.ConnTagInfo) context.Context {
	// no-op
	return ctx
}

// HandleRPC implements per-RPC tracing and stats instrumentation.
func (c *clientHandler) HandleRPC(ctx context.Context, rs stats.RPCStats) {
	handleRPC(ctx, rs)
}

// TagRPC implements per-RPC context management.
func (c *clientHandler) TagRPC(ctx context.Context, rti *stats.RPCTagInfo) context.Context {
	var span zipkin.Span

	ep := remoteEndpointFromContext(ctx, c.remoteServiceName)

	name := spanName(rti)
	span, ctx = c.tracer.StartSpanFromContext(ctx, name, zipkin.Kind(model.Client), zipkin.RemoteEndpoint(ep))

	md, ok := metadata.FromOutgoingContext(ctx)
	if ok {
		md = md.Copy()
	} else {
		md = metadata.New(nil)
	}
	_ = b3.InjectGRPC(&md)(span.Context())

	// inject baggage fields from span context into the outgoing gRPC request metadata
	if span.Context().Baggage != nil {
		span.Context().Baggage.Iterate(func(key string, values []string) {
			md.Set(key, values...)
		})
	}

	ctx = metadata.NewOutgoingContext(ctx, md)
	return ctx
}
