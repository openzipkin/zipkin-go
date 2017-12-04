package zipkin

import (
	"sync"
	"sync/atomic"
	"time"
)

type spanImpl struct {
	mtx sync.RWMutex
	SpanModel
	tracer          *Tracer
	isSampled       int32 // atomic bool (1 = true, 0 = false)
	explicitContext bool
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

// Finish the span and send to transporter.
func (s *spanImpl) Finish() {
	if atomic.CompareAndSwapInt32(&s.isSampled, 1, 0) {
		s.Duration = time.Since(s.Timestamp)
		s.tracer.options.transport.Send(s.SpanModel)
	}
}

// FinishWithTime allows one to provide the span end time and finishes the span.
func (s *spanImpl) FinishWithTime(t time.Time) {
	if atomic.CompareAndSwapInt32(&s.isSampled, 1, 0) {
		s.Duration = t.Sub(s.Timestamp)
		s.tracer.options.transport.Send(s.SpanModel)
	}
}

// FinishWithDuration allows one to provide the span duration and finishes the
// span.
func (s *spanImpl) FinishWithDuration(d time.Duration) {
	if atomic.CompareAndSwapInt32(&s.isSampled, 1, 0) {
		s.Duration = d
		s.tracer.options.transport.Send(s.SpanModel)
	}
}
