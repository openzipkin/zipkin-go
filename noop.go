package zipkin

import (
	"time"

	"github.com/openzipkin/zipkin-go/model"
)

type noopSpan struct {
	model.SpanContext
}

func (n *noopSpan) Context() model.SpanContext { return n.SpanContext }

func (*noopSpan) Annotate(time.Time, string) {}

func (*noopSpan) Tag(string, string) {}

func (*noopSpan) Finish() {}

func (*noopSpan) FinishWithTime(time.Time) {}

func (*noopSpan) FinishWithDuration(time.Duration) {}
