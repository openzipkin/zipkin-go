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

package log_test

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"github.com/openzipkin/zipkin-go/logging"
	"github.com/openzipkin/zipkin-go/model"
	logreporter "github.com/openzipkin/zipkin-go/reporter/log"
)

func newSpan() model.SpanModel {
	return model.SpanModel{
		SpanContext: model.SpanContext{
			ID:      model.ID(1),
			TraceID: model.TraceID{Low: 2},
		},
		Name:     "test",
		Duration: 100,
	}
}

func TestNewReporterNilLoggerDefaults(t *testing.T) {
	rep := logreporter.NewReporter(nil)
	if rep == nil {
		t.Fatal("expected non-nil reporter")
	}
	rep.Send(newSpan())
	if err := rep.Close(); err != nil {
		t.Fatalf("unexpected error closing reporter: %v", err)
	}
}

func TestNewReporterOutputsSpans(t *testing.T) {
	var buf bytes.Buffer
	rep := logreporter.NewReporter(log.New(&buf, "", 0))
	rep.Send(newSpan())
	if !strings.Contains(buf.String(), `"name": "test"`) {
		t.Errorf("expected span JSON in output, got: %s", buf.String())
	}
}

func TestNewReporterLogLevelSuppressesOutput(t *testing.T) {
	var buf bytes.Buffer
	rep := logreporter.NewReporter(
		log.New(&buf, "", 0),
		logreporter.LogLevel(logging.ErrorLevel),
	)
	rep.Send(newSpan())
	if buf.Len() != 0 {
		t.Errorf("expected no output at ErrorLevel, got: %s", buf.String())
	}
}
