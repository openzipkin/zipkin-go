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
	rpcHandlers       []RPCHandler
	remoteServiceName string
}

// A ClientOption can be passed to NewClientHandler to customize the returned handler.
type ClientOption func(*clientHandler)

// WithClientRPCHandler allows one to add custom logic for handling a stats.RPCStats, e.g.,
// to add additional tags.
func WithClientRPCHandler(handler RPCHandler) ClientOption {
	return func(c *clientHandler) {
		c.rpcHandlers = append(c.rpcHandlers, handler)
	}
}

// WithClientRemoteServiceName will set the value for the remote endpoint's service name on
// all spans.
func WithClientRemoteServiceName(name string) ClientOption {
	return func(c *clientHandler) {
		c.remoteServiceName = name
	}
}

// NewClientHandler returns a stats.Handler which can be used with grpc.WithStatsHandler to add
// tracing to a gRPC client. The gRPC method name is used as the span name and by default the only
// tags are the gRPC status code if the call fails. Use WithClientRPCHandler to add additional tags.
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
	handleRPC(span, rs, c.rpcHandlers)
}

// TagRPC implements per-RPC context management.
func (c *clientHandler) TagRPC(ctx context.Context, rti *stats.RPCTagInfo) context.Context {
	var span zipkin.Span

	ep, _ := zipkin.NewEndpoint(c.remoteServiceName, "")

	name := spanName(rti)
	span, ctx = c.tracer.StartSpanFromContext(ctx, name, zipkin.Kind(model.Client), zipkin.RemoteEndpoint(ep))

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
