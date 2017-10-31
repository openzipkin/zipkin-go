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
	Annotate(string, time.Time)
	Tag(string, string)
	Finish()
	FinishWith(options ...FinishOption)
}

// span implementation
type span struct {
	SpanContext
	mtx            sync.RWMutex
	Name           string            `json:"name"`
	Kind           kind.Type         `json:"kind,omitempty"`
	Timestamp      time.Time         `json:"timestamp,omitempty"`
	Duration       time.Duration     `json:"duration,imitempty"`
	Shared         bool              `json:"shared,omitempty"`
	LocalEndpoint  *Endpoint         `json:"localEndpoint,omitempty"`
	RemoteEndpoint *Endpoint         `json:"remoteEndpoint,omitempty"`
	Annotations    []annotation      `json:"annotations"`
	Tags           map[string]string `json:"tags"`
}

func (s *span) GetContext() SpanContext {
	return s.SpanContext
}

// Annotate adds a new Annotation to the Span.
func (s *span) Annotate(value string, t time.Time) {
	a := annotation{
		value:     value,
		timestamp: t,
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()

	s.Annotations = append(s.Annotations, a)
}

// Tag sets Tag with given key and value to the Span. If key already exists in
// the span the value will be overridden.
func (s *span) Tag(key, value string) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	s.Tags[key] = value
}

func (s *span) Finish() {
	s.Duration = time.Since(s.Timestamp)
}

func (s *span) FinishWith(options ...FinishOption) {
	for _, option := range options {
		option(s)
	}
}

func (s *span) MarshalJSON() ([]byte, error) {
	type Alias span
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
