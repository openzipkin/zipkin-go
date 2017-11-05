package zipkin

import (
	"time"

	"github.com/openzipkin/zipkin-go/kind"
)

type noopSpan struct {
	SpanContext
}

func (n *noopSpan) GetKind() kind.Type { return "" }

func (n *noopSpan) GetContext() SpanContext { return n.SpanContext }

func (n *noopSpan) SetContext(sc SpanContext) { n.SpanContext = sc }

func (n *noopSpan) SetTimestamp(t time.Time) {}

func (n *noopSpan) SetDuration(d time.Duration) {}

func (n *noopSpan) SetLocalEndpoint(*Endpoint) {}

func (n *noopSpan) SetRemoteEndpoint(*Endpoint) {}

func (*noopSpan) Annotate(time.Time, string) {}

func (*noopSpan) Tag(string, string) {}

func (*noopSpan) Finish() {}

func (*noopSpan) FinishWithTime(time.Time) {}

func (*noopSpan) FinishWithDuration(time.Duration) {}
