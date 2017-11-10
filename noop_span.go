package zipkin

import (
	"time"
)

type noopSpan struct {
	SpanContext
}

func (n *noopSpan) Context() SpanContext { return n.SpanContext }

func (*noopSpan) Annotate(time.Time, string) {}

func (*noopSpan) Tag(string, string) {}

func (*noopSpan) Finish() {}

func (*noopSpan) FinishWithTime(time.Time) {}

func (*noopSpan) FinishWithDuration(time.Duration) {}
