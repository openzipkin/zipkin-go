// Copyright 2019 The OpenZipkin Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package b3_test

import (
	"net/http"
	"testing"

	zipkin "github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/propagation/b3"
	"github.com/openzipkin/zipkin-go/reporter/recorder"
)

const (
	invalidID = "invalid_data"
)

func TestHTTPExtractFlagsOnly(t *testing.T) {
	r := newHTTPRequest(t)

	r.Header.Set(b3.Flags, "1")

	sc, err := b3.ExtractHTTP(r)()
	if err != nil {
		t.Fatalf("ExtractHTTP failed: %+v", err)
	}

	if want, have := true, sc.Debug; want != have {
		t.Errorf("sc.Debug want %+v, have %+v", want, have)
	}
}

func TestHTTPExtractSampledOnly(t *testing.T) {
	r := newHTTPRequest(t)

	r.Header.Set(b3.Sampled, "0")

	sc, err := b3.ExtractHTTP(r)()
	if err != nil {
		t.Fatalf("ExtractHTTP failed: %+v", err)
	}

	if sc.Sampled == nil {
		t.Fatalf("Sampled want %t, have nil", false)
	}

	if want, have := false, *sc.Sampled; want != have {
		t.Errorf("Sampled want %t, have %t", want, have)
	}

	r = newHTTPRequest(t)

	r.Header.Set(b3.Sampled, "1")

	sc, err = b3.ExtractHTTP(r)()
	if err != nil {
		t.Fatalf("ExtractHTTP failed: %+v", err)
	}

	if sc.Sampled == nil {
		t.Fatalf("Sampled want %t, have nil", true)
	}

	if want, have := true, *sc.Sampled; want != have {
		t.Errorf("Sampled want %t, have %t", want, have)
	}
}

func TestHTTPExtractFlagsAndSampledOnly(t *testing.T) {
	r := newHTTPRequest(t)

	r.Header.Set(b3.Flags, "1")
	r.Header.Set(b3.Sampled, "1")

	sc, err := b3.ExtractHTTP(r)()
	if err != nil {
		t.Fatalf("ExtractHTTP failed: %+v", err)
	}

	if want, have := true, sc.Debug; want != have {
		t.Errorf("Debug want %+v, have %+v", want, have)
	}

	// Sampled should not be set when sc.Debug is set.
	if sc.Sampled != nil {
		t.Errorf("Sampled want nil, have %+v", *sc.Sampled)
	}
}

func TestHTTPExtractSampledErrors(t *testing.T) {
	r := newHTTPRequest(t)

	r.Header.Set(b3.Sampled, "2")

	sc, err := b3.ExtractHTTP(r)()

	if want, have := b3.ErrInvalidSampledHeader, err; want != have {
		t.Errorf("SpanContext Error want %+v, have %+v", want, have)
	}

	if sc != nil {
		t.Errorf("SpanContext want nil, have: %+v", sc)
	}
}

func TestHTTPExtractFlagsErrors(t *testing.T) {
	values := map[string]bool{
		"1":    true,  // only acceptable Flags value, debug switches to true
		"true": false, // true is not a valid value for Flags
		"3":    false, // Flags is not a bitset
		"6":    false, // Flags is not a bitset
		"7":    false, // Flags is not a bitset
	}
	for value, debug := range values {
		r := newHTTPRequest(t)
		r.Header.Set(b3.Flags, value)
		spanContext, err := b3.ExtractHTTP(r)()
		if err != nil {
			// Flags should not trigger failed extraction
			t.Fatalf("ExtractHTTP failed: %+v", err)
		}
		if want, have := debug, spanContext.Debug; want != have {
			t.Errorf("SpanContext Error want %t, have %t", want, have)
		}
	}
}

func TestHTTPExtractScope(t *testing.T) {
	recorder := &recorder.ReporterRecorder{}
	defer recorder.Close()

	tracer, err := zipkin.NewTracer(recorder, zipkin.WithTraceID128Bit(true))
	if err != nil {
		t.Fatalf("Tracer failed: %+v", err)
	}

	iterations := 1000
	for i := 0; i < iterations; i++ {
		var (
			parent      = tracer.StartSpan("parent")
			child       = tracer.StartSpan("child", zipkin.Parent(parent.Context()))
			wantContext = child.Context()
		)

		r := newHTTPRequest(t)

		b3.InjectHTTP(r)(wantContext)

		haveContext, err := b3.ExtractHTTP(r)()
		if err != nil {
			t.Errorf("ExtractHTTP failed: %+v", err)
		}

		if haveContext == nil {
			t.Fatal("SpanContext want valid value, have nil")
		}

		if want, have := wantContext.TraceID, haveContext.TraceID; want != have {
			t.Errorf("TraceID want %+v, have %+v", want, have)
		}

		if want, have := wantContext.ID, haveContext.ID; want != have {
			t.Errorf("ID want %+v, have %+v", want, have)
		}
		if want, have := *wantContext.ParentID, *haveContext.ParentID; want != have {
			t.Errorf("ParentID want %+v, have %+v", want, have)
		}

		child.Finish()
		parent.Finish()
	}

	// check if we have all spans (2x the iterations: parent+child span)
	if want, have := 2*iterations, len(recorder.Flush()); want != have {
		t.Errorf("Recorded Span Count want %d, have %d", want, have)
	}
}

func TestHTTPExtractTraceIDError(t *testing.T) {
	r := newHTTPRequest(t)

	r.Header.Set(b3.TraceID, invalidID)

	_, err := b3.ExtractHTTP(r)()

	if want, have := b3.ErrInvalidTraceIDHeader, err; want != have {
		t.Errorf("ExtractHTTP Error want %+v, have %+v", want, have)
	}
}

func TestHTTPExtractSpanIDError(t *testing.T) {
	r := newHTTPRequest(t)

	r.Header.Set(b3.SpanID, invalidID)

	_, err := b3.ExtractHTTP(r)()

	if want, have := b3.ErrInvalidSpanIDHeader, err; want != have {
		t.Errorf("ExtractHTTP Error want %+v, have %+v", want, have)
	}
}

func TestHTTPExtractTraceIDOnlyError(t *testing.T) {
	r := newHTTPRequest(t)

	r.Header.Set(b3.TraceID, "1")

	_, err := b3.ExtractHTTP(r)()

	if want, have := b3.ErrInvalidScope, err; want != have {
		t.Errorf("ExtractHTTP Error want %+v, have %+v", want, have)
	}
}

func TestHTTPExtractSpanIDOnlyError(t *testing.T) {
	r := newHTTPRequest(t)

	r.Header.Set(b3.SpanID, "1")

	_, err := b3.ExtractHTTP(r)()

	if want, have := b3.ErrInvalidScope, err; want != have {
		t.Errorf("ExtractHTTP Error want %+v, have %+v", want, have)
	}
}

func TestHTTPExtractParentIDOnlyError(t *testing.T) {
	r := newHTTPRequest(t)

	r.Header.Set(b3.ParentSpanID, "1")

	_, err := b3.ExtractHTTP(r)()

	if want, have := b3.ErrInvalidScopeParent, err; want != have {
		t.Errorf("ExtractHTTP Error want %+v, have %+v", want, have)
	}
}

func TestHTTPExtractInvalidParentIDError(t *testing.T) {
	r := newHTTPRequest(t)

	r.Header.Set(b3.TraceID, "1")
	r.Header.Set(b3.SpanID, "2")
	r.Header.Set(b3.ParentSpanID, invalidID)

	_, err := b3.ExtractHTTP(r)()

	if want, have := b3.ErrInvalidParentSpanIDHeader, err; want != have {
		t.Errorf("ExtractHTTP Error want %+v, have %+v", want, have)
	}
}

func TestHTTPExtractSingleFailsAndMultipleFallsbackSuccessfully(t *testing.T) {
	r := newHTTPRequest(t)

	r.Header.Set(b3.Context, "invalid")
	r.Header.Set(b3.TraceID, "1")
	r.Header.Set(b3.SpanID, "2")

	_, err := b3.ExtractHTTP(r)()

	if err != nil {
		t.Errorf("ExtractHTTP Unexpected error %+v", err)
	}
}

func TestHTTPExtractSingleFailsAndMultipleFallsbackFailing(t *testing.T) {
	r := newHTTPRequest(t)

	r.Header.Set(b3.Context, "0000000000000001-0000000000000005-x")
	r.Header.Set(b3.TraceID, "1")
	r.Header.Set(b3.SpanID, "5")
	r.Header.Set(b3.ParentSpanID, invalidID)

	_, err := b3.ExtractHTTP(r)()

	if want, have := b3.ErrInvalidSampledByte, err; want != have {
		t.Errorf("HTTPExtract Error want %+v, have %+v", want, have)
	}
}

func TestHTTPInjectEmptyContextError(t *testing.T) {
	err := b3.InjectHTTP(nil)(model.SpanContext{})

	if want, have := b3.ErrEmptyContext, err; want != have {
		t.Errorf("HTTPInject Error want %+v, have %+v", want, have)
	}
}

func TestHTTPInjectDebugOnly(t *testing.T) {
	r := newHTTPRequest(t)

	sc := model.SpanContext{
		Debug: true,
	}

	b3.InjectHTTP(r)(sc)

	if want, have := "1", r.Header.Get(b3.Flags); want != have {
		t.Errorf("Flags want %s, have %s", want, have)
	}
}

func TestHTTPInjectSampledOnly(t *testing.T) {
	r := newHTTPRequest(t)

	sampled := false
	sc := model.SpanContext{
		Sampled: &sampled,
	}

	b3.InjectHTTP(r)(sc)

	if want, have := "0", r.Header.Get(b3.Sampled); want != have {
		t.Errorf("Sampled want %s, have %s", want, have)
	}
}

func TestHTTPInjectUnsampledTrace(t *testing.T) {
	r := newHTTPRequest(t)

	sampled := false
	sc := model.SpanContext{
		TraceID: model.TraceID{Low: 1},
		ID:      model.ID(2),
		Sampled: &sampled,
	}

	b3.InjectHTTP(r)(sc)

	if want, have := "0", r.Header.Get(b3.Sampled); want != have {
		t.Errorf("Sampled want %s, have %s", want, have)
	}
}

func TestHTTPInjectSampledAndDebugTrace(t *testing.T) {
	r := newHTTPRequest(t)

	sampled := true
	sc := model.SpanContext{
		TraceID: model.TraceID{Low: 1},
		ID:      model.ID(2),
		Debug:   true,
		Sampled: &sampled,
	}

	b3.InjectHTTP(r)(sc)

	if want, have := "", r.Header.Get(b3.Sampled); want != have {
		t.Errorf("Sampled want empty, have %s", have)
	}

	if want, have := "1", r.Header.Get(b3.Flags); want != have {
		t.Errorf("Debug want %s, have %s", want, have)
	}
}

func TestHTTPInjectWithSingleOnlyHeaders(t *testing.T) {
	r := newHTTPRequest(t)

	sampled := true
	sc := model.SpanContext{
		TraceID: model.TraceID{Low: 1},
		ID:      model.ID(7),
		Debug:   true,
		Sampled: &sampled,
	}

	b3.InjectHTTP(r, b3.WithSingleHeaderOnly())(sc)

	if want, have := "", r.Header.Get(b3.TraceID); want != have {
		t.Errorf("TraceID want empty, have %s", have)
	}

	if want, have := "0000000000000001-0000000000000007-d", r.Header.Get(b3.Context); want != have {
		t.Errorf("Context want %s, have %s", want, have)
	}
}
func TestHTTPInjectWithBothSingleAndMultipleHeaders(t *testing.T) {
	r := newHTTPRequest(t)

	sampled := true
	sc := model.SpanContext{
		TraceID: model.TraceID{Low: 1},
		ID:      model.ID(3),
		Debug:   true,
		Sampled: &sampled,
	}

	b3.InjectHTTP(r, b3.WithSingleAndMultiHeader())(sc)

	if want, have := "0000000000000001", r.Header.Get(b3.TraceID); want != have {
		t.Errorf("Trace ID want %s, have %s", want, have)
	}

	if want, have := "0000000000000001-0000000000000003-d", r.Header.Get(b3.Context); want != have {
		t.Errorf("Context want %s, have %s", want, have)
	}
}

func newHTTPRequest(t *testing.T) *http.Request {
	r, err := http.NewRequest("test", "", nil)
	if err != nil {
		t.Fatalf("HTTP Request failed: %+v", err)
	}
	return r
}
