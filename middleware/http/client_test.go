package http_test

import (
	"net/http"
	"testing"

	zipkin "github.com/openzipkin/zipkin-go"
	httpmiddleware "github.com/openzipkin/zipkin-go/middleware/http"
	"github.com/openzipkin/zipkin-go/reporter/recorder"
)

func TestMiddleware(t *testing.T) {
	reporter := recorder.NewReporter()
	defer reporter.Close()

	ep, _ := zipkin.NewEndpoint("httpClient", "")
	tracer, err := zipkin.NewTracer(reporter, zipkin.WithLocalEndpoint(ep))
	if err != nil {
		panic(err)
	}

	client, err := httpmiddleware.NewClient(tracer, nil, httpmiddleware.WithHTTPTrace(true))
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
