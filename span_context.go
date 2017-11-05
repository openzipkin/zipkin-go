package zipkin

import (
	"errors"
)

// ErrEmptyContext signals the span.Context was empty
var ErrEmptyContext = errors.New("empty request context")

// SpanContext holds the context of a Span.
type SpanContext struct {
	TraceID  TraceID `json:"traceId"`
	ID       ID      `json:"id"`
	ParentID *ID     `json:"parentId,omitempty"`
	Debug    bool    `json:"debug,omitempty"`
	Sampled  *bool   `json:"-"`
	err      error   // extraction error
}

// Empty returns true if SpanContext is zero value struct.
func (sc SpanContext) Empty() bool {
	return (SpanContext{}) == sc
}

// HasTrace returns true if SpanContext holds minimally
// required trace identifiers.
func (sc SpanContext) HasTrace() bool {
	if sc.TraceID.Empty() || sc.ID == 0 {
		return false
	}
	return true
}
