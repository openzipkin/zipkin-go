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

package recorder

import (
	"testing"

	"github.com/openzipkin/zipkin-go/model"
)

func TestFlushInRecorderSuccess(t *testing.T) {
	rec := NewReporter()

	span := model.SpanModel{}
	rec.Send(span)

	if len(rec.spans) != 1 {
		t.Fatalf("Span Count want 1, have %d", len(rec.spans))
	}

	rec.Flush()

	if len(rec.spans) != 0 {
		t.Fatalf("Span Count want 0, have %d", len(rec.spans))
	}
}

func TestCloseInRecorderSuccess(t *testing.T) {
	rec := NewReporter()

	span := model.SpanModel{}
	rec.Send(span)

	if len(rec.spans) != 1 {
		t.Fatalf("Span Count want 1, have %d", len(rec.spans))
	}

	rec.Close()

	if len(rec.spans) != 0 {
		t.Fatalf("Span Count want 0, have %d", len(rec.spans))
	}
}
