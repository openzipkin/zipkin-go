// +build go1.9

package grpc

import (
	"context"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/propagation/b3"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/stats"
)

type serverHandler struct {
	tracer            *zipkin.Tracer
	rpcHandlers       []RPCHandler
}

// A ServerOption can be passed to NewServerHandler to customize the returned handler.
type ServerOption func(*serverHandler)

// WithServerRPCHandler allows one to add custom logic for handling a stats.RPCStats, e.g.,
// to add additional tags.
func WithServerRPCHandler(handler RPCHandler) ServerOption {
	return func(s *serverHandler) {
		s.rpcHandlers = append(s.rpcHandlers, handler)
	}
}

// NewServerHandler returns a stats.Handler which can be used with grpc.WithStatsHandler to add
// tracing to a gRPC server. The gRPC method name is used as the span name and by default the only
// tags are the gRPC status code if the call fails. Use WithServerRPCHandler to add additional tags.
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
func (s *serverHandler) HandleConn(ctx context.Context, cs stats.ConnStats) {
	// no-op
}

// TagConn exists to satisfy gRPC stats.Handler.
func (s *serverHandler) TagConn(ctx context.Context, cti *stats.ConnTagInfo) context.Context {
	// no-op
	return ctx
}

// HandleRPC implements per-RPC tracing and stats instrumentation.
func (s *serverHandler) HandleRPC(ctx context.Context, rs stats.RPCStats) {
	span := zipkin.SpanFromContext(ctx)
	handleRpc(span, rs, s.rpcHandlers)
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

	span := s.tracer.StartSpan(name, zipkin.Kind(model.Server), zipkin.Parent(sc))
	return zipkin.NewContext(ctx, span)
}
