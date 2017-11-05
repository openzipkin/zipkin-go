package b3

import "errors"

// Common Header Extraction errors
var (
	ErrInvalidSampledHeader      = errors.New("invalid B3 Sampled header found")
	ErrInvalidFlagsHeader        = errors.New("invalid B3 Flags header found")
	ErrInvalidTraceIDHeader      = errors.New("invalid B3 TraceID header found")
	ErrInvalidSpanIDHeader       = errors.New("invalid B3 SpanID header found")
	ErrInvalidParentSpanIDHeader = errors.New("invalid B3 ParentSpanID header found")
	ErrInvalidScope              = errors.New("require either both TraceID and SpanID or none")
)

const (
	b3TraceID      = "X-B3-TraceId"
	b3SpanID       = "X-B3-SpanId"
	b3ParentSpanID = "X-B3-ParentSpanId"
	b3Sampled      = "X-B3-Sampled"
	b3Flags        = "X-B3-Flags"
)
