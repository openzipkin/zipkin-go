package grpc

import (
	"context"
	"strconv"
	"time"

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
	tracer *zipkin.Tracer
}

func NewClientHandler(tracer *zipkin.Tracer) ClientHandler {
	return &clientHandler{
		tracer,
	}
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
	switch rs := rs.(type) {
	case *stats.Begin:
		span.Tag("grpctrace.failfast", strconv.FormatBool(rs.FailFast))
	case *stats.InPayload:
		span.Annotate(time.Now(), "grpctrace.message_receive")
	case *stats.OutPayload:
		span.Annotate(time.Now(), "grpctrace.message_sent")
	case *stats.End:
		if rs.Error != nil {
			s, ok := status.FromError(rs.Error)
			if ok {
				zipkin.TagError.Set(span, s.Message())
			} else {
				zipkin.TagError.Set(span, rs.Error.Error())
			}
		}
		span.Finish()
	}
}

// TagRPC implements per-RPC context management.
func (c *clientHandler) TagRPC(ctx context.Context, rti *stats.RPCTagInfo) context.Context {
	name := spanName(rti)
	span, ctx := c.tracer.StartSpanFromContext(ctx, name, zipkin.Kind(model.Client))
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	} else {
		md = md.Copy()
	}
	_ = b3.InjectGRPC(&md)(span.Context())
	ctx = metadata.NewOutgoingContext(ctx, md)
	return ctx
}
