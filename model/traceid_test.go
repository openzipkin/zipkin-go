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

package model

import (
	"encoding/json"
	"testing"
)

func TestTraceID(t *testing.T) {
	traceID := TraceID{High: 1, Low: 2}
	if len(traceID.String()) != 32 {
		t.Errorf("Expected zero-padded TraceID to have 32 characters")
	}

	b, err := json.Marshal(traceID)
	if err != nil {
		t.Fatalf("Expected successful json serialization, got error: %+v", err)
	}

	if want, have := string(b), `"00000000000000010000000000000002"`; want != have {
		t.Fatalf("Expected json serialization, want %q, have %q", want, have)
	}

	var traceID2 TraceID
	if err = json.Unmarshal(b, &traceID2); err != nil {
		t.Fatalf("Expected successful json deserialization, got error: %+v", err)
	}

	if traceID2.High != traceID.High || traceID2.Low != traceID.Low {
		t.Fatalf("Unexpected traceID2, want: %#v, have %#v", traceID, traceID2)
	}

	have, err := TraceIDFromHex(traceID.String())
	if err != nil {
		t.Fatalf("Expected traceID got error: %+v", err)
	}
	if traceID.High != have.High || traceID.Low != have.Low {
		t.Errorf("Expected %+v, got %+v", traceID, have)
	}

	traceID = TraceID{High: 0, Low: 2}

	if len(traceID.String()) != 16 {
		t.Errorf("Expected zero-padded TraceID to have 16 characters, got %d", len(traceID.String()))
	}

	have, err = TraceIDFromHex(traceID.String())
	if err != nil {
		t.Fatalf("Expected traceID got error: %+v", err)
	}
	if traceID.High != have.High || traceID.Low != have.Low {
		t.Errorf("Expected %+v, got %+v", traceID, have)
	}

	traceID = TraceID{High: 0, Low: 0}

	if !traceID.Empty() {
		t.Errorf("Expected TraceID to be empty")
	}

	if _, err = TraceIDFromHex("12345678901234zz12345678901234zz"); err == nil {
		t.Errorf("Expected error got nil")
	}

	if err = json.Unmarshal([]byte(`"12345678901234zz12345678901234zz"`), &traceID); err == nil {
		t.Errorf("Expected error got nil")
	}
}
