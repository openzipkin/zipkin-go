package zipkin

import (
	"time"

	"github.com/openzipkin/zipkin-go/model"
)

type noopSpan struct {
	model.SpanContext
	tracer *Tracer
}

func (n *noopSpan) Tracer() *Tracer { return n.tracer }

func (n *noopSpan) Context() model.SpanContext { return n.SpanContext }

func (n *noopSpan) SetName(string) {}

func (*noopSpan) SetRemoteEndpoint(*model.Endpoint) {}

func (*noopSpan) Annotate(time.Time, string) {}

func (*noopSpan) Tag(string, string) {}

func (*noopSpan) Finish() {}

func (*noopSpan) Flush() {}
