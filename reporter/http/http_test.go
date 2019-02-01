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

package http_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"fmt"
	"time"

	"strings"

	"github.com/openzipkin/zipkin-go/idgenerator"
	"github.com/openzipkin/zipkin-go/model"
	zipkinhttp "github.com/openzipkin/zipkin-go/reporter/http"
)

func TestSpanIsBeingReported(t *testing.T) {
	idGen := idgenerator.NewRandom64()
	traceID := idGen.TraceID()

	nSpans := 2
	var aSpans []model.SpanModel
	var eSpans []string

	for i := 0; i < nSpans; i++ {
		span := model.SpanModel{
			SpanContext: model.SpanContext{
				TraceID: traceID,
				ID:      idGen.SpanID(traceID),
			},
			Name:      "name",
			Kind:      model.Client,
			Timestamp: time.Now(),
		}

		aSpans = append(aSpans, span)
		eSpans = append(
			eSpans,
			fmt.Sprintf(
				`{"timestamp":%d,"traceId":"%s","id":"%s","name":"%s","kind":"%s"}`,
				span.Timestamp.Round(time.Microsecond).UnixNano()/1e3,
				span.SpanContext.TraceID,
				span.SpanContext.ID,
				span.Name,
				span.Kind,
			),
		)
	}

	eSpansPayload := fmt.Sprintf("[%s]", strings.Join(eSpans, ","))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected 'POST' request, got '%s'", r.Method)
		}

		aSpanPayload, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Errorf("unexpected error: %s", err.Error())
		}

		if eSpansPayload != string(aSpanPayload) {
			t.Errorf("unexpected span payload \nwant %s, \nhave %s\n", eSpansPayload, string(aSpanPayload))
		}
	}))

	defer ts.Close()

	rep := zipkinhttp.NewReporter(ts.URL)
	defer rep.Close()

	for _, span := range aSpans {
		rep.Send(span)
	}
}
