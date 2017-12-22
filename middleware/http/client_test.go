package http_test

import (
	"net/http"
	"testing"

	zipkin "github.com/openzipkin/zipkin-go"
	httpclient "github.com/openzipkin/zipkin-go/middleware/http"
	"github.com/openzipkin/zipkin-go/reporter/recorder"
)

func TestHTTPClient(t *testing.T) {
	reporter := recorder.NewReporter()
	defer reporter.Close()

	ep, _ := zipkin.NewEndpoint("httpClient", "")
	tracer, err := zipkin.NewTracer(reporter, zipkin.WithLocalEndpoint(ep))
	if err != nil {
		panic(err)
	}

	clientTags := map[string]string{
		"client": "testClient",
	}

	transportTags := map[string]string{
		"conf.timeout": "default",
	}

	client, err := httpclient.NewClient(
		tracer,
		nil, // if set to nil, NewClient will use the default standard lib *http.Client configuration
		httpclient.ClientTrace(true),
		httpclient.ClientTags(clientTags),
		httpclient.TransportOptions(httpclient.TransportTags(transportTags)),
	)
	if err != nil {
		panic(err)
	}

	req, _ := http.NewRequest("GET", "https://www.google.com", nil)

	res, err := client.DoWithTrace(req, "Get Google")
	if err != nil {
		panic(err)
	}
	res.Body.Close()

	spans := reporter.Flush()
	if len(spans) < 2 {
		t.Errorf("Span Count want 2+, have %d", len(spans))
	}

	req, _ = http.NewRequest("GET", "https://www.google.com", nil)

	res, err = client.Do(req)
	if err != nil {
		panic(err)
	}
	res.Body.Close()

	spans = reporter.Flush()
	if len(spans) == 0 {
		t.Errorf("Span Count want 1+, have 0")
	}
}
