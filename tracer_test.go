package zipkin

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/reporter/log"
)

func TestInvalidTracerOption(t *testing.T) {
	tr, err := NewTracer(log.NewReporter(nil), WithLocalEndpoint(nil))
	if want, have := ErrInvalidEndpoint, err; want != have {
		t.Errorf("expected tracer creation failure: want %+v, have: %+v", want, have)
	}

	if tr != nil {
		t.Errorf("expected tracer to be nil got: %+v", tr)
	}
}

func TestTracerExtractor(t *testing.T) {
	tr, err := NewTracer(log.NewReporter(nil))
	if err != nil {
		t.Fatalf("unable to create tracer instance: %+v", err)
	}

	testErr1 := errors.New("extractor error")
	extractorErr := func() (*model.SpanContext, error) {
		return nil, testErr1
	}

	sc := tr.Extract(extractorErr)

	if want, have := testErr1, sc.Err; want != have {
		t.Errorf("expected extractor error: %+v, got %+v", want, have)
	}

	spanContext := model.SpanContext{}
	extractor := func() (*model.SpanContext, error) {
		return &spanContext, nil
	}

	sc = tr.Extract(extractor)

	if want, have := spanContext, sc; want != have {
		t.Errorf("expected span context: %+v, got %+v", want, have)
	}

	if want, have := &spanContext, &sc; want == have {
		t.Errorf("expected different span context objects")
	}
}

func TestNoopTracer(t *testing.T) {
	tr, err := NewTracer(log.NewReporter(nil))
	if err != nil {
		t.Fatalf("unable to create tracer instance: %+v", err)
	}

	pSC := model.SpanContext{
		TraceID: model.TraceID{
			High: 0,
			Low:  1,
		},
		ID: model.ID(1),
	}

	span := tr.StartSpan("test", Parent(pSC))

	if want, have := reflect.TypeOf(&spanImpl{}), reflect.TypeOf(span); want != have {
		t.Errorf("expected span implementation type: %+v, got %+v", want, have)
	}

	span.Finish()

	tr.SetNoop(true)

	testErr1 := errors.New("extractor error")
	extractor := func() (*model.SpanContext, error) {
		return nil, testErr1
	}

	sc := tr.Extract(extractor)

	if sc.Err != nil {
		t.Errorf("expected extractor noop: got error: %+v", sc.Err)
	}

	span = tr.StartSpan("test", Parent(pSC))

	if want, have := reflect.TypeOf(&noopSpan{}), reflect.TypeOf(span); want != have {
		t.Errorf("expected span implementation type: %+v, got %+v", want, have)
	}

	span.Finish()

	tr.SetNoop(false)

	span = tr.StartSpan("test", Parent(pSC))

	if want, have := reflect.TypeOf(&spanImpl{}), reflect.TypeOf(span); want != have {
		t.Errorf("expected span implementation type: %+v, got %+v", want, have)
	}

	span.Finish()
}

func TestNoopSpan(t *testing.T) {
	tr, err := NewTracer(log.NewReporter(nil), WithNoopSpan(true))
	if err != nil {
		t.Fatalf("unable to create tracer instance: %+v", err)
	}

	sampled := false
	pSC := model.SpanContext{
		TraceID: model.TraceID{
			High: 0,
			Low:  1,
		},
		ID:      model.ID(1),
		Sampled: &sampled,
	}

	span := tr.StartSpan("test", Parent(pSC))

	if want, have := reflect.TypeOf(&noopSpan{}), reflect.TypeOf(span); want != have {
		t.Errorf("expected span implementation type: %+v, got %+v", want, have)
	}

	span.Finish()
}

func TestUnsampledSpan(t *testing.T) {
	tr, err := NewTracer(log.NewReporter(nil))
	if err != nil {
		t.Fatalf("unable to create tracer instance: %+v", err)
	}

	sampled := false
	pSC := model.SpanContext{
		TraceID: model.TraceID{
			High: 0,
			Low:  1,
		},
		ID:      model.ID(1),
		Sampled: &sampled,
	}

	span := tr.StartSpan("test", Parent(pSC))

	if want, have := reflect.TypeOf(&spanImpl{}), reflect.TypeOf(span); want != have {
		t.Errorf("expected span implementation type: %+v, got %+v", want, have)
	}

	cSC := span.Context()

	if cSC.Err != nil {
		t.Errorf("expected Err to be nil, got %+v", cSC.Err)
	}

	if want, have := pSC.Debug, cSC.Debug; want != have {
		t.Errorf("expected Debug %t, got %t", want, have)
	}

	if want, have := pSC.TraceID, cSC.TraceID; want != have {
		t.Errorf("expected TraceID: %+v, got: %+v", want, have)
	}

	if cSC.ID == 0 {
		t.Error("expected valid ID")
	}

	if cSC.ParentID == nil {
		t.Error("expected valid ParentID, got nil")
	} else if want, have := pSC.ID, *cSC.ParentID; want != have {
		t.Errorf("expected ParentID: %+v, got: %+v", want, have)
	}

	if cSC.Sampled == nil {
		t.Error("expected explicit Sampled value, got nil")
	} else if *cSC.Sampled {
		t.Errorf("expected Sampled value false, got %+v", *cSC.Sampled)
	}

	if want, have := int32(0), span.(*spanImpl).mustCollect; want != have {
		t.Errorf("expected mustCollect %d, got %d", want, have)
	}

	span.Finish()
}

func TestDefaultTags(t *testing.T) {
	var (
		scTagKey   = "spanScopedTag"
		scTagValue = "spanPayload"
		tags       = make(map[string]string)
	)
	tags["platform"] = "zipkin_test"
	tags["version"] = "1.0"

	tr, err := NewTracer(log.NewReporter(nil), WithTags(tags))
	if err != nil {
		t.Fatalf("unable to create tracer instance: %+v", err)
	}

	pSC := model.SpanContext{
		TraceID: model.TraceID{
			High: 0,
			Low:  1,
		},
		ID: model.ID(1),
	}

	span := tr.StartSpan("test", Kind(model.Server), Parent(pSC))
	span.Tag(scTagKey, scTagValue)

	foundTags := span.(*spanImpl).Tags

	for key, value := range tags {
		foundValue, foundKey := foundTags[key]
		if !foundKey {
			t.Errorf("expected tag %q = %q, got key not found", key, value)
		} else if value != foundValue {
			t.Errorf("expected tag %q = %q, got %q = %q", key, value, key, foundValue)
		}
	}

	foundValue, foundKey := foundTags[scTagKey]
	if !foundKey {
		t.Errorf("expected tag %q = %q, got key not found", scTagKey, scTagValue)
	} else if want, have := scTagValue, foundValue; want != have {
		t.Errorf("expected tag %q = %q, got %q = %q", scTagKey, scTagValue, scTagKey, foundValue)
	}
}

func TestDebugFlagWithoutParentTrace(t *testing.T) {
	/*
	   Test handling of a single Debug flag without an existing trace
	*/
	tr, err := NewTracer(log.NewReporter(nil), WithSharedSpans(true))
	if err != nil {
		t.Fatalf("unable to create tracer instance: %+v", err)
	}

	pSC := model.SpanContext{
		Debug: true,
	}

	span := tr.StartSpan("test", Parent(pSC))

	cSC := span.Context()

	if cSC.Err != nil {
		t.Errorf("expected Err to be nil, got %+v", cSC.Err)
	}

	if want, have := pSC.Debug, cSC.Debug; want != have {
		t.Errorf("expected Debug %t, got %t", want, have)
	}

	if want, have := false, cSC.TraceID.Empty(); want != have {
		t.Error("expected valid TraceID")
	}

	if cSC.ID == 0 {
		t.Error("expected valid ID")
	}

	if cSC.ParentID != nil {
		t.Errorf("expected empty ParentID, got: %+v", cSC.ParentID)
	}

	if cSC.Sampled != nil {
		t.Errorf("expected Sampled to be nil, got: %+v", cSC.Sampled)
	}

	if want, have := int32(1), span.(*spanImpl).mustCollect; want != have {
		t.Errorf("expected mustCollect %d, got %d", want, have)
	}
}

func TestParentSpanInSharedMode(t *testing.T) {
	tr, err := NewTracer(log.NewReporter(nil), WithSharedSpans(true))
	if err != nil {
		t.Fatalf("unable to create tracer instance: %+v", err)
	}

	parentID := model.ID(1)

	pSC := model.SpanContext{
		TraceID: model.TraceID{
			High: 0,
			Low:  1,
		},
		ID:       model.ID(2),
		ParentID: &parentID,
	}

	span := tr.StartSpan("test", Kind(model.Server), Parent(pSC))

	cSC := span.Context()

	if cSC.Err != nil {
		t.Errorf("expected Err to be nil, got %+v", cSC.Err)
	}

	if want, have := pSC.Debug, cSC.Debug; want != have {
		t.Errorf("expected Debug %t, got %t", want, have)
	}

	if want, have := pSC.TraceID, cSC.TraceID; want != have {
		t.Errorf("expected TraceID: %+v, got: %+v", want, have)
	}

	if want, have := pSC.ID, cSC.ID; want != have {
		t.Errorf("expected ID: %+v, got: %+v", want, have)
	}

	if cSC.ParentID == nil {
		t.Error("expected valid ParentID, got nil")
	} else if want, have := parentID, *cSC.ParentID; want != have {
		t.Errorf("expected ParentID: %+v, got: %+v", want, have)
	}

	if cSC.Sampled == nil {
		t.Error("expected explicit Sampled value, got nil")
	} else if !*cSC.Sampled {
		t.Errorf("expected Sampled value true, got %+v", *cSC.Sampled)
	}

	if want, have := int32(1), span.(*spanImpl).mustCollect; want != have {
		t.Errorf("expected mustCollect %d, got %d", want, have)
	}
}

func TestParentSpanInSpanPerNodeMode(t *testing.T) {
	tr, err := NewTracer(log.NewReporter(nil), WithSharedSpans(false))
	if err != nil {
		t.Fatalf("unable to create tracer instance: %+v", err)
	}

	pSC := model.SpanContext{
		TraceID: model.TraceID{
			High: 0,
			Low:  1,
		},
		ID: model.ID(1),
	}

	span := tr.StartSpan("test", Kind(model.Server), Parent(pSC))

	cSC := span.Context()

	if cSC.Err != nil {
		t.Errorf("expected Err to be nil, got %+v", cSC.Err)
	}

	if want, have := pSC.Debug, cSC.Debug; want != have {
		t.Errorf("expected Debug %t, got %t", want, have)
	}

	if want, have := pSC.TraceID, cSC.TraceID; want != have {
		t.Errorf("expected TraceID: %+v, got: %+v", want, have)
	}

	if cSC.ID == 0 {
		t.Error("expected valid ID")
	}

	if cSC.ParentID == nil {
		t.Error("expected valid ParentID, got nil")
	} else if want, have := pSC.ID, *cSC.ParentID; want != have {
		t.Errorf("expected ParentID: %+v, got: %+v", want, have)
	}

	if cSC.Sampled == nil {
		t.Error("expected explicit Sampled value, got nil")
	} else if !*cSC.Sampled {
		t.Errorf("expected Sampled value true, got %+v", *cSC.Sampled)
	}

	if want, have := int32(1), span.(*spanImpl).mustCollect; want != have {
		t.Errorf("expected mustCollect %d, got %d", want, have)
	}
}

func TestStartSpanFromContext(t *testing.T) {
	tr, err := NewTracer(log.NewReporter(nil), WithSharedSpans(true))
	if err != nil {
		t.Fatalf("unable to create tracer instance: %+v", err)
	}

	cSpan := tr.StartSpan("test", Kind(model.Client))

	ctx := NewContext(context.Background(), cSpan)

	sSpan, _ := tr.StartSpanFromContext(ctx, "testChild", Kind(model.Server))

	cS, sS := cSpan.(*spanImpl), sSpan.(*spanImpl)

	if want, have := model.Client, cS.Kind; want != have {
		t.Errorf("expected Kind: %+v, got: %+v", want, have)
	}

	if want, have := model.Server, sS.Kind; want != have {
		t.Errorf("expected Kind: %+v, got: %+v", want, have)
	}

	if want, have := cS.TraceID, sS.TraceID; want != have {
		t.Errorf("expected TraceID: %+v, got: %+v", want, have)
	}

	if want, have := cS.ID, sS.ID; want != have {
		t.Errorf("expected Span ID: %+v, got: %+v", want, have)
	}

	if want, have := cS.ParentID, sS.ParentID; want != have {
		t.Errorf("expected Span ID: %+v, got: %+v", want, have)
	}

}
