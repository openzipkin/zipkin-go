package zipkin

import (
	"context"
	"testing"
)

func TestSpanOrNoopFromContext(t *testing.T) {
	var (
		ctx   = context.Background()
		tr, _ = NewTracer(nil, WithLocalEndpoint(nil))
		span  = tr.StartSpan("test")
	)

	if want, have := defaultNoopSpan, SpanOrNoopFromContext(ctx); want != have {
		t.Errorf("Invalid response want %+v, have %+v", want, have)
	}

	ctx = NewContext(ctx, span)

	if want, have := span, SpanOrNoopFromContext(ctx); want != have {
		t.Errorf("Invalid response want %+v, have %+v", want, have)
	}

}
