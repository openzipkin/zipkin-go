package zipkin

import (
	"encoding/json"
	"time"

	"github.com/openzipkin/zipkin-go/kind"
)

// Span interface
type Span interface {
	Context() SpanContext
	Annotate(time.Time, string)
	Tag(string, string)
	Finish()
	FinishWithTime(time.Time)
	FinishWithDuration(d time.Duration)
}

// SpanContext holds the context of a Span.
type SpanContext struct {
	TraceID  TraceID `json:"traceId"`
	ID       ID      `json:"id"`
	ParentID *ID     `json:"parentId,omitempty"`
	Debug    bool    `json:"debug,omitempty"`
	Sampled  *bool   `json:"-"` // (not marshalled)
	err      error   // extraction error (unexported)
}

// SpanModel Structure
type SpanModel struct {
	SpanContext
	Name           string            `json:"name"`
	Kind           kind.Type         `json:"kind,omitempty"`
	Timestamp      time.Time         `json:"timestamp,omitempty"`
	Duration       time.Duration     `json:"duration,imitempty"`
	Shared         bool              `json:"shared"`
	LocalEndpoint  *Endpoint         `json:"localEndpoint,omitempty"`
	RemoteEndpoint *Endpoint         `json:"remoteEndpoint,omitempty"`
	Annotations    []Annotation      `json:"annotations"`
	Tags           map[string]string `json:"tags"`
}

// MarshalJSON marshalls our Model into the correct format for V2 API
func (s SpanModel) MarshalJSON() ([]byte, error) {
	type Alias SpanModel
	return json.Marshal(&struct {
		Timestamp int64 `json:"timestamp,omitempty"`
		Duration  int64 `json:"duration,omitempty"`
		Alias
	}{
		Timestamp: s.Timestamp.UnixNano() / 1e3,
		Duration:  s.Duration.Nanoseconds() / 1e3,
		Alias:     (Alias)(s),
	})
}
