package zipkin

import (
	"time"

	"github.com/openzipkin/zipkin-go/model"
)

// Span interface as returned by Tracer.StartSpan()
type Span interface {
	Context() model.SpanContext
	SetRemoteEndpoint(model.Endpoint)
	Annotate(time.Time, string)
	Tag(string, string)
	Finish()
}
