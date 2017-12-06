package zipkin

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/openzipkin/zipkin-go/model"
)

type spanImpl struct {
	mtx sync.RWMutex
	model.SpanModel
	tracer    *Tracer
	isSampled int32 // atomic bool (1 = true, 0 = false)
}

func (s *spanImpl) Context() model.SpanContext {
	return s.SpanContext
}

// Annotate adds a new Annotation to the Span.
func (s *spanImpl) Annotate(t time.Time, value string) {
	a := model.Annotation{
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

// Finish the span and send to reporter.
func (s *spanImpl) Finish() {
	if atomic.CompareAndSwapInt32(&s.isSampled, 1, 0) {
		s.Duration = time.Since(s.Timestamp)
		s.tracer.reporter.Send(s.SpanModel)
	}
}
