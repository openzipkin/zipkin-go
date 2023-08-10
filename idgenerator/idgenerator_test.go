// Copyright 2022 The OpenZipkin Authors
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

package idgenerator_test

import (
	"testing"

	"github.com/openzipkin/zipkin-go/idgenerator"
	"github.com/openzipkin/zipkin-go/model"
)

func TestRandom64(t *testing.T) {
	var (
		spanID  model.ID
		gen     = idgenerator.NewRandom64()
		traceID = gen.TraceID()
	)

	if traceID.Empty() {
		t.Errorf("Expected valid TraceID, got: %+v", traceID)
	}

	if want, have := uint64(0), traceID.High; want != have {
		t.Errorf("Expected TraceID.High to be 0, got %d", have)
	}

	spanID = gen.SpanID(traceID)

	if want, have := model.ID(traceID.Low), spanID; want != have {
		t.Errorf("Expected root span to have span ID %d, got %d", want, have)
	}

	spanID = gen.SpanID(model.TraceID{})

	if spanID == 0 {
		t.Errorf("Expected child span to have a valid span ID, got 0")
	}
}

func TestRandom128(t *testing.T) {
	var (
		spanID  model.ID
		gen     = idgenerator.NewRandom128()
		traceID = gen.TraceID()
	)

	if traceID.Empty() {
		t.Errorf("Expected valid TraceID, got: %+v", traceID)
	}

	if traceID.Low == 0 {
		t.Error("Expected TraceID.Low to have value, got 0")
	}

	if traceID.High == 0 {
		t.Error("Expected TraceID.High to have value, got 0")
	}

	spanID = gen.SpanID(traceID)

	if want, have := model.ID(traceID.Low), spanID; want != have {
		t.Errorf("Expected root span to have span ID %d, got %d", want, have)
	}

	spanID = gen.SpanID(model.TraceID{})

	if spanID == 0 {
		t.Errorf("Expected child span to have a valid span ID, got 0")
	}
}

func TestRandomTimeStamped(t *testing.T) {
	var (
		spanID  model.ID
		gen     = idgenerator.NewRandomTimestamped()
		traceID = gen.TraceID()
	)

	if traceID.Empty() {
		t.Errorf("Expected valid TraceID, got: %+v", traceID)
	}

	if traceID.Low == 0 {
		t.Error("Expected TraceID.Low to have value, got 0")
	}

	if traceID.High == 0 {
		t.Error("Expected TraceID.High to have value, got 0")
	}

	spanID = gen.SpanID(traceID)

	if want, have := model.ID(traceID.Low), spanID; want != have {
		t.Errorf("Expected root span to have span ID %d, got %d", want, have)
	}

	spanID = gen.SpanID(model.TraceID{})

	if spanID == 0 {
		t.Errorf("Expected child span to have a valid span ID, got 0")
	}

	// test chronological order
	var ids []model.TraceID

	for i := 0; i < 1000; i++ {
		ids = append(ids, gen.TraceID())
	}

	var latestTS uint64
	for idx, traceID := range ids {
		if newVal, oldVal := traceID.High>>32, latestTS; newVal < oldVal {
			t.Errorf("[%d] expected a higher timestamp part in traceid but got: old: %d new: %d", idx, oldVal, newVal)
		}
		latestTS = traceID.High >> 32
	}
}
