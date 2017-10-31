package zipkin

import (
	"time"

	"github.com/openzipkin/zipkin-go/kind"
)

// SpanOption ...
type SpanOption func(t *Tracer, s *span)

// Parent SpanOption allows one to provide the SpanContext of the Parent Span.
func Parent(sc SpanContext) SpanOption {
	return func(t *Tracer, s *span) {
		if sc.Empty() {
			return
		}

		s.SpanContext.TraceID = sc.TraceID
		s.SpanContext.Sampled = sc.Sampled
		s.SpanContext.Debug = sc.Debug

		if t.options.sharedSpans && s.Kind == kind.Server {
			// join span
			s.SpanContext.ID = sc.ID
			s.SpanContext.ParentID = sc.ParentID
		} else {
			// regular child span
			s.SpanContext.ID = t.options.generate.SpanID()
			s.SpanContext.ParentID = &sc.ID
		}
	}
}

// WithSpanContext SpanOption allows one to set an explicit SpanContext for the
// span.
func WithSpanContext(sc SpanContext) SpanOption {
	return func(t *Tracer, s *span) {
		s.SpanContext = sc
	}
}

// StartTime uses a given start time.
func StartTime(start time.Time) SpanOption {
	return func(t *Tracer, s *span) {
		s.Timestamp = start
	}
}

// LocalEndpoint overrides the local endpoint. Typically used in CLIENT
// Kind spans.
func LocalEndpoint(e *Endpoint) SpanOption {
	return func(t *Tracer, s *span) {
		s.LocalEndpoint = e
	}
}

// RemoteEndpoint overrides the remote endpoint. Typically used in CLIENT
// Kind spans.
func RemoteEndpoint(e *Endpoint) SpanOption {
	return func(t *Tracer, s *span) {
		s.RemoteEndpoint = e
	}
}

// FinishOption ...
type FinishOption func(s *span)

// FinishTime uses a given finish time.
func FinishTime(t time.Time) FinishOption {
	return func(s *span) {
		s.Duration = t.Sub(s.Timestamp)
	}
}

// Duration uses a given duration.
func Duration(d time.Duration) FinishOption {
	return func(s *span) {
		s.Duration = d
	}
}
