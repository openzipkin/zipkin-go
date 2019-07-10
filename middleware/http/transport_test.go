package http

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	zipkin "github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/reporter/recorder"
)

type errRoundTripper struct {
	err error
}

func (r errRoundTripper) RoundTrip(_ *http.Request) (*http.Response, error) {
	return nil, r.err
}

func TestRoundTripErrHandlingForRoundTripError(t *testing.T) {
	expectedErr := errors.New("error message")
	tracer, err := zipkin.NewTracer(nil)
	if err != nil {
		t.Fatalf("unexpected error when creating tracer: %v", err)
	}
	req, _ := http.NewRequest("GET", "localhost", nil)
	transport, _ := NewTransport(
		tracer,
		TransportErrHandler(func(_ zipkin.Span, err error, statusCode int) {
			if want, have := expectedErr, err; want != have {
				t.Errorf("unexpected error, want %q, have %q", want, have)
			}
		}),
		RoundTripper(&errRoundTripper{err: expectedErr}),
	)

	_, err = transport.RoundTrip(req)
	if err == nil {
		t.Fatalf("expected error: %v", expectedErr)
	}
}

func TestRoundTripErrHandlingForStatusCode(t *testing.T) {
	tcs := []struct {
		actualStatusCode int
		expectedError    int
	}{
		// we start on 200, if we pass 100 it will wait until timeout.
		{
			actualStatusCode: 200,
		},
		{
			actualStatusCode: 301,
		},
		{
			actualStatusCode: 403,
			expectedError:    403,
		},
		{
			actualStatusCode: 504,
			expectedError:    504,
		},
	}

	for _, tc := range tcs {
		srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			rw.WriteHeader(tc.actualStatusCode)
		}))

		tracer, err := zipkin.NewTracer(nil)
		if err != nil {
			t.Fatalf("unexpected error when creating tracer: %v", err)
		}
		req, _ := http.NewRequest("GET", srv.URL, nil)
		transport, _ := NewTransport(
			tracer,
			TransportErrHandler(func(_ zipkin.Span, err error, statusCode int) {
				if want, have := tc.expectedError, statusCode; want != 0 && want != have {
					t.Errorf("unexpected status code, want %d, have %d", want, have)
				}
			}),
		)

		_, err = transport.RoundTrip(req)
		if err != nil {
			t.Fatalf("unexpected error in the round trip: %v", err)
		}

		srv.Close()
	}
}

func TestRoundTripErrResponseReadingSuccess(t *testing.T) {
	expectedBody := []byte("message")
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(500)
		rw.Write(expectedBody)
	}))
	defer srv.Close()

	tracer, err := zipkin.NewTracer(nil)
	if err != nil {
		t.Fatalf("unexpected error when creating tracer: %v", err)
	}
	req, _ := http.NewRequest("GET", srv.URL, nil)
	transport, _ := NewTransport(
		tracer,
		TransportErrResponseReader(func(_ zipkin.Span, br io.Reader) {
			body, _ := ioutil.ReadAll(br)
			if want, have := expectedBody, body; string(want) != string(have) {
				t.Errorf("unexpected body, want %q, have %q", want, have)
			}
		}),
	)

	res, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	actualBody, _ := ioutil.ReadAll(res.Body)
	if want, have := expectedBody, actualBody; string(expectedBody) != string(actualBody) {
		t.Errorf("unexpected body: want %s, have %s", want, have)
	}
}

func TestTransportRequestSamplerOverridesSamplingFromContext(t *testing.T) {
	cases := []struct {
		Sampler          func(uint64) bool
		RequestSampler   RequestSamplerFunc
		ExpectedSampling string
	}{
		// Test proper handling when there is no RequestSampler
		{
			Sampler:          zipkin.AlwaysSample,
			RequestSampler:   nil,
			ExpectedSampling: "1",
		},
		// Test proper handling when there is no RequestSampler
		{
			Sampler:          zipkin.NeverSample,
			RequestSampler:   nil,
			ExpectedSampling: "0",
		},
		// Test RequestSampler override sample -> no sample
		{
			Sampler:          zipkin.AlwaysSample,
			RequestSampler:   func(_ *http.Request) *bool { return Discard() },
			ExpectedSampling: "0",
		},
		// Test RequestSampler override no sample -> sample
		{
			Sampler:          zipkin.NeverSample,
			RequestSampler:   func(_ *http.Request) *bool { return Sample() },
			ExpectedSampling: "1",
		},
		// Test RequestSampler pass through of sampled decision
		{
			Sampler: zipkin.AlwaysSample,
			RequestSampler: func(r *http.Request) *bool {
				return nil
			},
			ExpectedSampling: "1",
		},
		// Test RequestSampler pass through of not sampled decision
		{
			Sampler: zipkin.NeverSample,
			RequestSampler: func(r *http.Request) *bool {
				return nil
			},
			ExpectedSampling: "0",
		},
	}

	for i, c := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			if want, have := c.ExpectedSampling, r.Header.Get("x-b3-sampled"); want != have {
				t.Errorf("unexpected sampling decision in case #%d, want %q, have %q", i, want, have)
			}
		}))

		// we need to use a valid reporter or Tracer will implement noop mode which makes this test invalid
		rep := recorder.NewReporter()

		tracer, err := zipkin.NewTracer(rep, zipkin.WithSampler(c.Sampler))
		if err != nil {
			t.Fatalf("unexpected error when creating tracer: %v", err)
		}

		sp := tracer.StartSpan("op1")
		defer sp.Finish()
		ctx := zipkin.NewContext(context.Background(), sp)

		req, _ := http.NewRequest("GET", srv.URL, nil)
		transport, _ := NewTransport(
			tracer,
			TransportRequestSampler(c.RequestSampler),
		)

		_, err = transport.RoundTrip(req.WithContext(ctx))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		rep.Close()
		srv.Close()
	}
}
