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

package http_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/openzipkin/zipkin-go"
	mw "github.com/openzipkin/zipkin-go/middleware/http"
	"github.com/openzipkin/zipkin-go/reporter/recorder"
)

var (
	lep, _ = zipkin.NewEndpoint("testSvc", "127.0.0.1:0")
)

func httpHandler(code int, headers http.Header, body *bytes.Buffer) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(code)
		for key, value := range headers {
			w.Header().Add(key, value[0])
		}
		_, _ = w.Write(body.Bytes())
	}
}

func TestHTTPHandlerWrapping(t *testing.T) {
	var (
		spanRecorder = &recorder.ReporterRecorder{}
		tr, _        = zipkin.NewTracer(spanRecorder, zipkin.WithLocalEndpoint(lep))
		headers      = make(http.Header)
		spanName     = "wrapper_test"
		code         = 404
		request      *http.Request
	)

	headers.Add("some-key", "some-value")
	headers.Add("other-key", "other-value")

	testCases := []struct {
		method          string
		requestBody     *bytes.Buffer
		responseBody    *bytes.Buffer
		hasRequestSize  bool
		hasResponseSize bool
	}{
		{
			method:          "POST",
			requestBody:     bytes.NewBufferString("incoming data"),
			responseBody:    bytes.NewBufferString("oh oh we have a 404"),
			hasRequestSize:  true,
			hasResponseSize: true,
		},
		{
			method:          "POST",
			requestBody:     bytes.NewBufferString(""),
			responseBody:    bytes.NewBufferString("oh oh we have a 404"),
			hasRequestSize:  false,
			hasResponseSize: true,
		},
		{
			method:          "GET",
			requestBody:     nil,
			responseBody:    bytes.NewBufferString(""),
			hasRequestSize:  false,
			hasResponseSize: false,
		},
	}

	for _, c := range testCases {
		httpRecorder := httptest.NewRecorder()

		var err error
		if c.requestBody == nil {
			request, err = http.NewRequest(c.method, "/test", nil)
		} else {
			request, err = http.NewRequest(c.method, "/test", c.requestBody)
		}
		if err != nil {
			t.Fatalf("unable to create request")
		}

		httpHandlerFunc := httpHandler(code, headers, c.responseBody)

		tags := map[string]string{
			"component": "testServer",
		}
		handler := mw.NewServerMiddleware(
			tr,
			mw.SpanName(spanName),
			mw.TagResponseSize(true),
			mw.ServerTags(tags),
		)(httpHandlerFunc)

		handler.ServeHTTP(httpRecorder, request)

		spans := spanRecorder.Flush()

		if want, have := 1, len(spans); want != have {
			t.Errorf("Expected %d spans, got %d", want, have)
		}

		span := spans[0]

		if want, have := spanName, span.Name; want != have {
			t.Errorf("Expected span name %s, got %s", want, have)
		}

		if c.hasRequestSize {
			if want, have := strconv.Itoa(c.requestBody.Len()), span.Tags["http.request.size"]; want != have {
				t.Errorf("Expected span request size %s, got %s", want, have)
			}
		} else {
			// http.request.size should not be present as request body is empty.
			if _, ok := span.Tags["http.request.size"]; ok {
				t.Errorf("Unexpected span request size")
			}
		}

		if c.hasResponseSize {
			if want, have := strconv.Itoa(c.responseBody.Len()), span.Tags["http.response.size"]; want != have {
				t.Errorf("Expected span response size %s, got %s", want, have)
			}
		} else {
			// http.response.size should not be present as request body is empty.
			if _, ok := span.Tags["http.response.size"]; ok {
				t.Errorf("Unexpected span response size")
			}
		}

		if want, have := strconv.Itoa(code), span.Tags["http.status_code"]; want != have {
			t.Errorf("Expected span status code %s, got %s", want, have)
		}

		if want, have := strconv.Itoa(code), span.Tags["error"]; want != have {
			t.Errorf("Expected span error %q, got %q", want, have)
		}

		if want, have := len(headers), len(httpRecorder.HeaderMap); want != have {
			t.Errorf("Expected http header count %d, got %d", want, have)
		}

		if want, have := code, httpRecorder.Code; want != have {
			t.Errorf("Expected http status code %d, got %d", want, have)
		}

		for key, value := range headers {
			if want, have := value, httpRecorder.HeaderMap.Get(key); want[0] != have {
				t.Errorf("Expected header %s value %s, got %s", key, want, have)
			}
		}

		if want, have := c.responseBody.String(), httpRecorder.Body.String(); want != have {
			t.Errorf("Expected body value %q, got %q", want, have)
		}
	}
}

func TestHTTPDefaultSpanName(t *testing.T) {
	var (
		spanRecorder = &recorder.ReporterRecorder{}
		tr, _        = zipkin.NewTracer(spanRecorder, zipkin.WithLocalEndpoint(lep))
		httpRecorder = httptest.NewRecorder()
		requestBuf   = bytes.NewBufferString("incoming data")
		methodType   = "POST"
		code         = ""
	)

	request, err := http.NewRequest(methodType, "/test", requestBuf)
	if err != nil {
		t.Fatalf("unable to create request")
	}

	httpHandlerFunc := httpHandler(200, nil, bytes.NewBufferString(""))

	handler := mw.NewServerMiddleware(tr)(httpHandlerFunc)

	handler.ServeHTTP(httpRecorder, request)

	spans := spanRecorder.Flush()

	if want, have := 1, len(spans); want != have {
		t.Errorf("Expected %d spans, got %d", want, have)
	}

	span := spans[0]

	if want, have := methodType, span.Name; want != have {
		t.Errorf("Expected span name %s, got %s", want, have)
	}

	if want, have := code, span.Tags["http.status_code"]; want != have {
		t.Errorf("Expected span status code %s, got %s", want, have)
	}
}

func TestHTTPRequestSampler(t *testing.T) {
	var (
		spanRecorder    = &recorder.ReporterRecorder{}
		httpRecorder    = httptest.NewRecorder()
		requestBuf      = bytes.NewBufferString("incoming data")
		methodType      = "POST"
		httpHandlerFunc = httpHandler(200, nil, bytes.NewBufferString(""))
	)

	samplers := []func(r *http.Request) *bool{
		nil,
		func(*http.Request) *bool { return mw.Sample() },
		func(*http.Request) *bool { return mw.Discard() },
		func(*http.Request) *bool { return nil },
	}

	for idx, sampler := range samplers {
		tr, _ := zipkin.NewTracer(spanRecorder, zipkin.WithLocalEndpoint(lep), zipkin.WithSampler(zipkin.AlwaysSample))

		request, err := http.NewRequest(methodType, "/test", requestBuf)
		if err != nil {
			t.Fatalf("unable to create request")
		}

		handler := mw.NewServerMiddleware(tr, mw.RequestSampler(sampler))(httpHandlerFunc)

		handler.ServeHTTP(httpRecorder, request)

		spans := spanRecorder.Flush()

		sampledSpans := 0
		if sampler == nil || sampler(request) == nil || *(sampler(request)) {
			sampledSpans = 1
		}

		if want, have := sampledSpans, len(spans); want != have {
			t.Errorf("[%d] Expected %d spans, got %d", idx, want, have)
		}
	}

	for idx, sampler := range samplers {
		tr, _ := zipkin.NewTracer(spanRecorder, zipkin.WithLocalEndpoint(lep), zipkin.WithSampler(zipkin.NeverSample))

		request, err := http.NewRequest(methodType, "/test", requestBuf)
		if err != nil {
			t.Fatalf("unable to create request")
		}

		handler := mw.NewServerMiddleware(tr, mw.RequestSampler(sampler))(httpHandlerFunc)

		handler.ServeHTTP(httpRecorder, request)

		spans := spanRecorder.Flush()

		sampledSpans := 0
		if sampler != nil && sampler(request) != nil && *(sampler(request)) {
			sampledSpans = 1
		}

		if want, have := sampledSpans, len(spans); want != have {
			t.Errorf("[%d] Expected %d spans, got %d", idx, want, have)
		}
	}

}
