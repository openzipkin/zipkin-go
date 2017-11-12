package zipkin

import (
	"sync"
	"time"
)

type spanImpl struct {
	mtx sync.RWMutex
	SpanModel
	tracer    *Tracer
	isSampled bool
}

func (s *spanImpl) Context() SpanContext {
	return s.SpanContext
}

// Annotate adds a new Annotation to the Span.
func (s *spanImpl) Annotate(t time.Time, value string) {
	a := Annotation{
		Timestamp: t,
		Value:     value,
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()

	s.Annotations = append(s.Annotations, a)
}

// Tag sets Tag with given key and value to the Span. If key already exists in
// the span the value will be overridden.
func (s *spanImpl) Tag(key, value string) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	s.Tags[key] = value
}

func (s *spanImpl) Finish() {
	s.Duration = time.Since(s.Timestamp)
	if s.isSampled {
		s.tracer.options.transport.Send(s.SpanModel)
	}
}

func (s *spanImpl) FinishWithTime(t time.Time) {
	s.Duration = t.Sub(s.Timestamp)
	if s.isSampled {
		s.tracer.options.transport.Send(s.SpanModel)
	}
}

func (s *spanImpl) FinishWithDuration(d time.Duration) {
	s.Duration = d
	if s.isSampled {
		s.tracer.options.transport.Send(s.SpanModel)
	}
}
