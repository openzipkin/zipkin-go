package span

import (
	"context"

	zipkin "github.com/openzipkin/zipkin-go"
)

// FromContext retrieves a Zipkin Span from Go's context propagation
// mechanism if found. If not found, returns nil.
func FromContext(ctx context.Context) zipkin.Span {
	if s, ok := ctx.Value(spanKey).(zipkin.Span); ok {
		return s
	}
	return nil
}

// NewContext stores a Zipkin Span into Go's context propagation mechanism.
func NewContext(ctx context.Context, s zipkin.Span) context.Context {
	return context.WithValue(ctx, spanKey, s)
}

type ctxKey struct{}

var spanKey = ctxKey{}
