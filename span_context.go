package zipkin

import (
	"fmt"
	"strconv"
)

// SpanContext holds the context of a Span.
type SpanContext struct {
	TraceID  TraceID `json:"traceId"`
	ParentID *SpanID `json:"parentId,omitempty"`
	ID       SpanID  `json:"id"`
	Sampled  *bool   `json:"-"`
	Debug    bool    `json:"debug,omitempty"`
}

// Empty returns true if SpanContext is zero value struct.
func (s SpanContext) Empty() bool {
	return (SpanContext{}) == s
}

// SpanID type
type SpanID uint64

// MarshalJSON serializes SpanID to HEX.
func (s SpanID) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(
		"%016q", strconv.FormatUint(uint64(s), 16),
	)), nil
}
