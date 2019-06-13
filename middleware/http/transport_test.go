package http

import (
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	zipkin "github.com/openzipkin/zipkin-go"
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
