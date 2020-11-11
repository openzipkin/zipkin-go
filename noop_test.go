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

package zipkin

import (
	"reflect"
	"testing"
	"time"

	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/reporter"
)

func TestNoopContext(t *testing.T) {
	var (
		span     Span
		sc       model.SpanContext
		parentID = model.ID(3)
		tr, _    = NewTracer(
			reporter.NewNoopReporter(),
			WithNoopSpan(true),
			WithSampler(NeverSample),
			WithSharedSpans(true),
		)
	)

	sc = model.SpanContext{
		TraceID:  model.TraceID{High: 1, Low: 2},
		ID:       model.ID(4),
		ParentID: &parentID,
		Debug:    false,     // debug must be false
		Sampled:  new(bool), // bool must be pointer to false
	}

	span = tr.StartSpan("testNoop", Parent(sc), Kind(model.Server))

	noop, ok := span.(*noopSpan)
	if !ok {
		t.Fatalf("Span type want %s, have %s", reflect.TypeOf(&spanImpl{}), reflect.TypeOf(span))
	}

	if have := noop.Context(); !reflect.DeepEqual(sc, have) {
		t.Errorf("Context want %+v, have %+v", sc, have)
	}

	span.Tag("dummy", "dummy")
	span.Annotate(time.Now(), "dummy")
	span.SetName("dummy")
	span.SetRemoteEndpoint(nil)
	span.Flush()
}

func TestIsNoop(t *testing.T) {
	sc := model.SpanContext{
		TraceID: model.TraceID{High: 1, Low: 2},
		ID:      model.ID(3),
		Sampled: new(bool),
	}

	ns := &noopSpan{sc}

	if want, have := true, IsNoop(ns); want != have {
		t.Error("unexpected noop")
	}

	span := &spanImpl{SpanModel: model.SpanModel{SpanContext: sc}}

	if want, have := false, IsNoop(span); want != have {
		t.Error("expected noop")
	}
}
