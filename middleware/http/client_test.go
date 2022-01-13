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
	"net/http"
	"testing"

	"github.com/openzipkin/zipkin-go"
	httpclient "github.com/openzipkin/zipkin-go/middleware/http"
	"github.com/openzipkin/zipkin-go/reporter/recorder"
)

func TestHTTPClient(t *testing.T) {
	reporter := recorder.NewReporter()
	defer func() {
		_ = reporter.Close()
	}()

	ep, _ := zipkin.NewEndpoint("httpClient", "")
	tracer, err := zipkin.NewTracer(reporter, zipkin.WithLocalEndpoint(ep))
	if err != nil {
		t.Fatalf("unable to create tracer: %+v", err)
	}

	clientTags := map[string]string{
		"client": "testClient",
	}

	transportTags := map[string]string{
		"conf.timeout": "default",
	}

	remoteEndpoint, _ := zipkin.NewEndpoint("google-service", "1.2.3.4:80")
	client, err := httpclient.NewClient(
		tracer,
		httpclient.WithClient(&http.Client{}),
		httpclient.ClientTrace(true),
		httpclient.ClientTags(clientTags),
		httpclient.TransportOptions(httpclient.TransportTags(transportTags)),
		httpclient.WithRemoteEndpoint(remoteEndpoint),
	)
	if err != nil {
		t.Fatalf("unable to create http client: %+v", err)
	}

	req, _ := http.NewRequest("GET", "https://www.google.com", nil)

	res, err := client.DoWithAppSpan(req, "Get Google")
	if err != nil {
		t.Fatalf("unable to execute client request: %+v", err)
	}
	_ = res.Body.Close()

	spans := reporter.Flush()
	if len(spans) < 2 {
		t.Errorf("Span Count want 2+, have %d", len(spans))
	}

	rep := spans[0].RemoteEndpoint
	if rep == nil {
		t.Errorf("Span remoteEndpoint must not nil")
	}
	if rep.ServiceName != remoteEndpoint.ServiceName {
		t.Errorf("Span remoteEndpoint ServiceName want %s, have %s", remoteEndpoint.ServiceName, rep.ServiceName)
	}

	req, _ = http.NewRequest("GET", "https://www.google.com", nil)

	res, err = client.Do(req)
	if err != nil {
		t.Fatalf("unable to execute client request: %+v", err)
	}
	_ = res.Body.Close()

	spans = reporter.Flush()
	if len(spans) == 0 {
		t.Errorf("Span Count want 1+, have 0")
	}

	span := tracer.StartSpan("ParentSpan")

	req, _ = http.NewRequest("GET", "http://www.google.com", nil)

	ctx := zipkin.NewContext(req.Context(), span)

	req = req.WithContext(ctx)

	res, err = client.DoWithAppSpan(req, "ChildSpan")
	if err != nil {
		t.Fatalf("unable to execute client request: %+v", err)
	}
	_ = res.Body.Close()
}
