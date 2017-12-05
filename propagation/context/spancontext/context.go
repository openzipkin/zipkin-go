/*
Package spancontext implements logic for propagating Zipkin SpanContexts through
Go's context.Context.
*/
package spancontext

import (
	"context"

	zipkin "github.com/openzipkin/zipkin-go"
)

// FromContext retrieves a Zipkin SpanContext from Go's context propagation
// mechanism if found. If not found, returns empty SpanContext.
func FromContext(ctx context.Context) zipkin.SpanContext {
	if s, ok := ctx.Value(spanContextKey).(zipkin.SpanContext); ok {
		return s
	}
	return zipkin.SpanContext{}
}

// NewContext stores a Zipkin SpanContext into Go's context propagation mechanism.
func NewContext(ctx context.Context, sc zipkin.SpanContext) context.Context {
	return context.WithValue(ctx, spanContextKey, sc)
}

type spanCtxKey struct{}

var spanContextKey = spanCtxKey{}
