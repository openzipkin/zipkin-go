package zipkin

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/openzipkin/zipkin-go/kind"
)

// Span interface
type Span interface {
	GetContext() SpanContext
	SetContext(SpanContext)
	SetLocalEndpoint(*Endpoint)
	SetRemoteEndpoint(*Endpoint)
	Annotate(time.Time, string)
	Tag(string, string)
	Finish()
	SetTimestamp(t time.Time)
	SetDuration(d time.Duration)
	FinishWithTime(time.Time)
	FinishWithDuration(d time.Duration)
}

type spanImpl struct {
	SpanContext
	mtx            sync.RWMutex
	Name           string            `json:"name"`
	Kind           kind.Type         `json:"kind,omitempty"`
	Timestamp      time.Time         `json:"timestamp,omitempty"`
	Duration       time.Duration     `json:"duration,imitempty"`
	Shared         bool              `json:"shared"`
	LocalEndpoint  *Endpoint         `json:"localEndpoint,omitempty"`
	RemoteEndpoint *Endpoint         `json:"remoteEndpoint,omitempty"`
	Annotations    []annotation      `json:"annotations"`
	Tags           map[string]string `json:"tags"`
}

func (s *spanImpl) GetContext() SpanContext {
	return s.SpanContext
}

func (s *spanImpl) SetContext(sc SpanContext) {
	s.SpanContext = sc
}

func (s *spanImpl) SetTimestamp(t time.Time) {
	s.Timestamp = t
}

func (s *spanImpl) SetDuration(d time.Duration) {
	s.Duration = d
}

func (s *spanImpl) SetLocalEndpoint(e *Endpoint) {
	s.LocalEndpoint = e
}

func (s *spanImpl) SetRemoteEndpoint(e *Endpoint) {
	s.RemoteEndpoint = e
}

// Annotate adds a new Annotation to the Span.
func (s *spanImpl) Annotate(t time.Time, value string) {
	a := annotation{
		timestamp: t,
		value:     value,
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
}

func (s *spanImpl) FinishWithTime(t time.Time) {
	s.Duration = t.Sub(s.Timestamp)
}

func (s *spanImpl) FinishWithDuration(d time.Duration) {
	s.Duration = d
}

func (s *spanImpl) MarshalJSON() ([]byte, error) {
	type Alias spanImpl
	return json.Marshal(&struct {
		Timestamp int64 `json:"timestamp,omitempty"`
		Duration  int64 `json:"duration,omitempty"`
		*Alias
	}{
		Timestamp: s.Timestamp.UnixNano() / 1e3,
		Duration:  s.Duration.Nanoseconds() / 1e3,
		Alias:     (*Alias)(s),
	})
}
