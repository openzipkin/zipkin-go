// +build go1.9

package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"

	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/propagation/b3"
)

type ClientHandler interface {
	stats.Handler
}

type clientHandler struct {
	tracer            *zipkin.Tracer
	rpcHandlers       []RPCHandler
	remoteServiceName string
}

type ClientOption func(*clientHandler)

type RPCHandler func(span zipkin.Span, rpcStats stats.RPCStats)

// WithRPCHandler allows one to add custom logic for handling a stats.RPCStats, e.g.,
// to add additional tags.
func WithRPCHandler(handler RPCHandler) ClientOption {
	return func(c *clientHandler) {
		c.rpcHandlers = append(c.rpcHandlers, handler)
	}
}

// NewClientHandler returns a stats.Handler which can be used with grpc.WithStatsHandler to add
// tracing to a gRPC client. The gRPC method name is used as the span name and by default the only
// tags are the gRPC status code if the call fails. Use WithRPCHandler to add additional tags.
func NewClientHandler(tracer *zipkin.Tracer, options ...ClientOption) ClientHandler {
	c := &clientHandler{
		tracer: tracer,
	}
	for _, option := range options {
		option(c)
	}
	return c
}

// HandleConn exists to satisfy gRPC stats.Handler.
func (c *clientHandler) HandleConn(ctx context.Context, cs stats.ConnStats) {
	// no-op
}

// TagConn exists to satisfy gRPC stats.Handler.
func (c *clientHandler) TagConn(ctx context.Context, cti *stats.ConnTagInfo) context.Context {
	// no-op
	return ctx
}

// HandleRPC implements per-RPC tracing and stats instrumentation.
func (c *clientHandler) HandleRPC(ctx context.Context, rs stats.RPCStats) {
	span := zipkin.SpanFromContext(ctx)

	for _, h := range c.rpcHandlers {
		h(span, rs)
	}

	switch rs := rs.(type) {
	case *stats.End:
		if rs.Error != nil {
			s, ok := status.FromError(rs.Error)
			if ok {
				if s.Code() != codes.OK {
					c := s.Code().String()
					span.Tag("grpc.status_code", c)
					zipkin.TagError.Set(span, c)
				}
			} else {
				zipkin.TagError.Set(span, rs.Error.Error())
			}
		}
		span.Finish()
	}
}

// TagRPC implements per-RPC context management.
func (c *clientHandler) TagRPC(ctx context.Context, rti *stats.RPCTagInfo) context.Context {
	var span zipkin.Span

	name := spanName(rti)
	span, ctx = c.tracer.StartSpanFromContext(ctx, name, zipkin.Kind(model.Client))
	md, ok := metadata.FromOutgoingContext(ctx)
	if ok {
		md = md.Copy()
	} else {
		md = metadata.New(nil)
	}
	_ = b3.InjectGRPC(&md)(span.Context())
	ctx = metadata.NewOutgoingContext(ctx, md)
	return ctx
}
