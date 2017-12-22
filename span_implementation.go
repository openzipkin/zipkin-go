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
	tracer      *Tracer
	mustCollect int32 // used as atomic bool (1 = true, 0 = false)
}

func (s *spanImpl) Context() model.SpanContext {
	return s.SpanContext
}

func (s *spanImpl) SetRemoteEndpoint(e *model.Endpoint) {
	s.mtx.Lock()
	if e == nil {
		s.RemoteEndpoint = nil
	} else {
		s.RemoteEndpoint = &model.Endpoint{}
		*s.RemoteEndpoint = *e
	}
	s.mtx.Unlock()
}

// Annotate adds a new Annotation to the Span.
func (s *spanImpl) Annotate(t time.Time, value string) {
	a := model.Annotation{
		Timestamp: t,
		Value:     value,
	}

	s.mtx.Lock()
	s.Annotations = append(s.Annotations, a)
	s.mtx.Unlock()
}

// Tag sets Tag with given key and value to the Span. If key already exists in
// the span the value will be overridden except for error tags where the first
// value is persisted.
func (s *spanImpl) Tag(key, value string) {
	s.mtx.Lock()

	if key == string(TagError) {
		if _, found := s.Tags[key]; found {
			s.mtx.Unlock()
			return
		}
	}

	s.Tags[key] = value
	s.mtx.Unlock()
}

// Finish the span and send to reporter.
func (s *spanImpl) Finish() {
	if atomic.CompareAndSwapInt32(&s.mustCollect, 1, 0) {
		s.Duration = time.Since(s.Timestamp)
		s.tracer.reporter.Send(s.SpanModel)
	}
}
