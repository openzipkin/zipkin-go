package b3

import "errors"

// Common Header Extraction / Injection errors
var (
	ErrInvalidSampledHeader      = errors.New("invalid B3 Sampled header found")
	ErrInvalidFlagsHeader        = errors.New("invalid B3 Flags header found")
	ErrInvalidTraceIDHeader      = errors.New("invalid B3 TraceID header found")
	ErrInvalidSpanIDHeader       = errors.New("invalid B3 SpanID header found")
	ErrInvalidParentSpanIDHeader = errors.New("invalid B3 ParentSpanID header found")
	ErrInvalidScope              = errors.New("require either both TraceID and SpanID or none")
	ErrEmptyContext              = errors.New("empty request context")
)

const (
	b3TraceID      = "x-b3-traceid"
	b3SpanID       = "x-b3-spanid"
	b3ParentSpanID = "x-b3-parentspanid"
	b3Sampled      = "x-b3-sampled"
	b3Flags        = "x-b3-flags"
)
