package b3_test

import (
	"net/http"
	"testing"

	zipkin "github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/propagation/b3"
	"github.com/openzipkin/zipkin-go/reporter/recorder"
)

func TestHTTPExtractFlagsOnly(t *testing.T) {
	r, err := http.NewRequest("test", "", nil)

	if err != nil {
		t.Fatalf("unable to create new HTTP Request: %+v", err)
	}

	r.Header.Set(b3.Flags, "1")
	r.Header.Set(b3.Sampled, "1")

	sc, err := b3.ExtractHTTP(r)()

	if err != nil {
		t.Errorf("unexpected error: %+v", err)
	}

	if sc == nil {
		t.Fatal("expected SpanContext, got nil")
	}

	if want, have := true, sc.Debug; want != have {
		t.Errorf("expected sc.Debug %+v, got: %+v", want, have)
	}

	if sc.Sampled != nil {
		t.Errorf("expected sampled to be nil due to sc.Debug being set, got %+v", *sc.Sampled)
	}

	r, err = http.NewRequest("test", "", nil)

	if err != nil {
		t.Fatalf("unable to create new HTTP Request: %+v", err)
	}

	r.Header.Set(b3.Sampled, "0")

	sc, err = b3.ExtractHTTP(r)()

	if err != nil {
		t.Errorf("unexpected error: %+v", err)
	}

	if sc == nil {
		t.Fatal("expected SpanContext, got nil")
	}

	if sc.Sampled == nil {
		t.Fatal("expected sampled to be set, got nil")
	}

	if want, have := false, *sc.Sampled; want != have {
		t.Errorf("expected sampled %t, got %t", want, have)
	}

	r, err = http.NewRequest("test", "", nil)

	if err != nil {
		t.Fatalf("unable to create new HTTP Request: %+v", err)
	}

	r.Header.Set(b3.Sampled, "1")

	sc, err = b3.ExtractHTTP(r)()

	if err != nil {
		t.Errorf("unexpected error: %+v", err)
	}

	if sc == nil {
		t.Fatal("expected SpanContext, got nil")
	}

	if sc.Sampled == nil {
		t.Fatal("expected sampled to be set, got nil")
	}

	if want, have := true, *sc.Sampled; want != have {
		t.Errorf("expected sampled %t, got %t", want, have)
	}
}

func TestHTTPExtractFlagsErrors(t *testing.T) {
	r, err := http.NewRequest("test", "", nil)

	if err != nil {
		t.Fatalf("unable to create new HTTP Request: %+v", err)
	}

	r.Header.Set(b3.Sampled, "2")

	sc, err := b3.ExtractHTTP(r)()

	if want, have := b3.ErrInvalidSampledHeader, err; want != have {
		t.Errorf("expected error %+v, got %+v", want, have)
	}

	if sc != nil {
		t.Errorf("expected SpanContext to be nil, got: %+v", sc)
	}

	r, err = http.NewRequest("test", "", nil)

	if err != nil {
		t.Fatalf("unable to create new HTTP Request: %+v", err)
	}

	r.Header.Set(b3.Flags, "2")

	sc, err = b3.ExtractHTTP(r)()

	if want, have := b3.ErrInvalidFlagsHeader, err; want != have {
		t.Errorf("expected error %+v, got %+v", want, have)
	}

	if sc != nil {
		t.Errorf("expected SpanContext to be nil, got: %+v", sc)
	}
}

func TestHTTPExtractScope(t *testing.T) {
	recorder := &recorder.ReporterRecorder{}
	tracer, err := zipkin.NewTracer(recorder, zipkin.WithTraceID128Bit(true))
	if err != nil {
		t.Fatalf("unable to create new Tracer: %+v", err)
	}

	for i := 0; i < 1000; i++ {
		parent := tracer.StartSpan("parent")
		child := tracer.StartSpan("child", zipkin.Parent(parent.Context()))
		wantContext := child.Context()

		r, err := http.NewRequest("test", "", nil)
		if err != nil {
			t.Fatalf("unable to create new HTTP Request: %+v", err)
		}

		b3.InjectHTTP(r)(wantContext)

		haveContext, err := b3.ExtractHTTP(r)()

		if err != nil {
			t.Errorf("unexpected error: %+v", err)
		}

		if haveContext == nil {
			t.Fatal("expected SpanContext, got nil")
		}

		if want, have := wantContext.TraceID, haveContext.TraceID; want != have {
			t.Errorf("expected traceid %+v, got %+v", want, have)
		}

		if want, have := wantContext.ID, haveContext.ID; want != have {
			t.Errorf("expected span id %+v, got %+v", want, have)
		}
		if want, have := *wantContext.ParentID, *haveContext.ParentID; want != have {
			t.Errorf("expected parent spanid %+v, got %+v", want, have)
		}

		child.Finish()
		parent.Finish()
	}
}

func TestHTTPExtractIdentifierErrors(t *testing.T) {
	r, err := http.NewRequest("test", "", nil)

	if err != nil {
		t.Fatalf("unable to create new HTTP Request: %+v", err)
	}

	r.Header.Set(b3.TraceID, "invaliddata")

	_, err = b3.ExtractHTTP(r)()

	if want, have := b3.ErrInvalidTraceIDHeader, err; want != have {
		t.Errorf("expected traceid error %+v, got %+v", want, have)
	}

	r, err = http.NewRequest("test", "", nil)

	if err != nil {
		t.Fatalf("unable to create new HTTP Request: %+v", err)
	}

	r.Header.Set(b3.TraceID, "1")
	r.Header.Set(b3.SpanID, "invaliddata")

	_, err = b3.ExtractHTTP(r)()

	if want, have := b3.ErrInvalidSpanIDHeader, err; want != have {
		t.Errorf("expected spanid error %+v, got %+v", want, have)
	}

	r, err = http.NewRequest("test", "", nil)

	if err != nil {
		t.Fatalf("unable to create new HTTP Request: %+v", err)
	}

	r.Header.Set(b3.TraceID, "1")

	_, err = b3.ExtractHTTP(r)()

	if want, have := b3.ErrInvalidScope, err; want != have {
		t.Errorf("expected scope error %+v, got %+v", want, have)
	}

	r, err = http.NewRequest("test", "", nil)

	if err != nil {
		t.Fatalf("unable to create new HTTP Request: %+v", err)
	}

	r.Header.Set(b3.ParentSpanID, "1")

	_, err = b3.ExtractHTTP(r)()

	if want, have := b3.ErrInvalidScopeParent, err; want != have {
		t.Errorf("expected scope error %+v, got %+v", want, have)
	}

	r, err = http.NewRequest("test", "", nil)

	if err != nil {
		t.Fatalf("unable to create new HTTP Request: %+v", err)
	}

	r.Header.Set(b3.TraceID, "1")
	r.Header.Set(b3.SpanID, "2")
	r.Header.Set(b3.ParentSpanID, "invaliddata")

	_, err = b3.ExtractHTTP(r)()

	if want, have := b3.ErrInvalidParentSpanIDHeader, err; want != have {
		t.Errorf("expected scope error %+v, got %+v", want, have)
	}

}

func TestHTTPInject(t *testing.T) {
	if want, have := b3.ErrEmptyContext, b3.InjectHTTP(nil)(model.SpanContext{}); want != have {
		t.Errorf("expected error %+v, got %+v", want, have)
	}

	r, err := http.NewRequest("test", "", nil)

	if err != nil {
		t.Fatalf("unable to create new HTTP Request: %+v", err)
	}

	sc := model.SpanContext{
		Debug: true,
	}

	b3.InjectHTTP(r)(sc)

	if want, have := "1", r.Header.Get(b3.Flags); want != have {
		t.Errorf("expected B3 flags %s, got %s", want, have)
	}

	sampled := false
	sc = model.SpanContext{
		Sampled: &sampled,
	}

	r, err = http.NewRequest("test", "", nil)

	if err != nil {
		t.Fatalf("unable to create new HTTP Request: %+v", err)
	}

	b3.InjectHTTP(r)(sc)

	if want, have := "", r.Header.Get(b3.Sampled); want != have {
		t.Errorf("expected empty B3 sampled header, got %s", have)
	}

	sampled = false
	sc = model.SpanContext{
		TraceID: model.TraceID{Low: 1},
		ID:      model.ID(2),
		Sampled: &sampled,
	}

	r, err = http.NewRequest("test", "", nil)

	if err != nil {
		t.Fatalf("unable to create new HTTP Request: %+v", err)
	}

	b3.InjectHTTP(r)(sc)

	if want, have := "0", r.Header.Get(b3.Sampled); want != have {
		t.Errorf("expected B3 sampled %s, got %s", want, have)
	}

	sampled = true
	sc = model.SpanContext{
		TraceID: model.TraceID{Low: 1},
		ID:      model.ID(2),
		Debug:   true,
		Sampled: &sampled,
	}

	r, err = http.NewRequest("test", "", nil)

	if err != nil {
		t.Fatalf("unable to create new HTTP Request: %+v", err)
	}

	b3.InjectHTTP(r)(sc)
	if want, have := "", r.Header.Get(b3.Sampled); want != have {
		t.Errorf("expected empty B3 sampled header, got %s", have)
	}

}
