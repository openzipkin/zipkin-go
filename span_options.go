package zipkin

import (
	"time"

	"github.com/openzipkin/zipkin-go/kind"
)

// SpanOption ...
type SpanOption func(t *Tracer, s *spanImpl)

// Kind sets the kind of the span.
func Kind(k kind.Type) SpanOption {
	return func(t *Tracer, s *spanImpl) {
		s.Kind = k
	}
}

// Parent will return a parent span context given parent's extracted context
func Parent(sc SpanContext) SpanOption {
	return func(t *Tracer, s *spanImpl) {
		s.SpanContext = sc
		s.explicitContext = false
	}
}

// WithSpanContext SpanOption allows one to set an explicit SpanContext for the
// span.
func WithSpanContext(sc SpanContext) SpanOption {
	return func(t *Tracer, s *spanImpl) {
		s.SpanContext = sc
		s.explicitContext = true
	}
}

// StartTime uses a given start time.
func StartTime(start time.Time) SpanOption {
	return func(t *Tracer, s *spanImpl) {
		s.Timestamp = start
	}
}

// LocalEndpoint overrides the local endpoint. Typically used in CLIENT
// Kind spans.
func LocalEndpoint(e *Endpoint) SpanOption {
	return func(t *Tracer, s *spanImpl) {
		s.LocalEndpoint = e
	}
}

// RemoteEndpoint overrides the remote endpoint. Typically used in CLIENT
// Kind spans.
func RemoteEndpoint(e *Endpoint) SpanOption {
	return func(t *Tracer, s *spanImpl) {
		s.RemoteEndpoint = e
	}
}
