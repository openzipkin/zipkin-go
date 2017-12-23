package zipkin

import (
	"time"

	"github.com/openzipkin/zipkin-go/model"
)

// Span interface as returned by Tracer.StartSpan()
type Span interface {
	// Context returns the Span's SpanContext.
	Context() model.SpanContext

	// SetName updates the Span's name.
	SetName(string)

	// SetRemoteEndpoint updates the Span's Remote Endpoint.
	SetRemoteEndpoint(*model.Endpoint)

	// Annotate adds a timed event to the Span.
	Annotate(time.Time, string)

	// Tag sets Tag with given key and value to the Span. If key already exists in
	// the Span the value will be overridden except for error tags where the first
	// value is persisted.
	Tag(string, string)

	// Finish the Span and send to Reporter.
	Finish()
}
