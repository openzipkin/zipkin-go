package zipkin

import (
	"time"

	"github.com/openzipkin/zipkin-go/model"
)

// SpanOption allows for functional options to adjust behavior and payload of
// the Span to be created with tracer.StartSpan().
type SpanOption func(t *Tracer, s *spanImpl)

// Kind sets the kind of the span being created..
func Kind(kind model.Kind) SpanOption {
	return func(t *Tracer, s *spanImpl) {
		s.Kind = kind
	}
}

// Parent will use provided SpanContext as parent to the span being created.
func Parent(sc model.SpanContext) SpanOption {
	return func(t *Tracer, s *spanImpl) {
		s.SpanContext = sc
		s.explicitContext = false
	}
}

// WithSpanContext allows one to set an explicit SpanContext for the span being
// created.
func WithSpanContext(sc model.SpanContext) SpanOption {
	return func(t *Tracer, s *spanImpl) {
		s.SpanContext = sc
		s.explicitContext = true
	}
}

// StartTime uses a given start time for the span being created.
func StartTime(start time.Time) SpanOption {
	return func(t *Tracer, s *spanImpl) {
		s.Timestamp = start
	}
}

// LocalEndpoint overrides the local endpoint of the span being created.
// Typically used in CLIENT Kind spans.
func LocalEndpoint(e *model.Endpoint) SpanOption {
	return func(t *Tracer, s *spanImpl) {
		s.LocalEndpoint = e
	}
}

// RemoteEndpoint sets the remote endpoint of the span being created.
func RemoteEndpoint(e *model.Endpoint) SpanOption {
	return func(t *Tracer, s *spanImpl) {
		s.RemoteEndpoint = e
	}
}
