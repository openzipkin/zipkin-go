package http

import (
	"log"
	"net/http"

	"context"

	"github.com/openzipkin/zipkin-go"
)

var ctx context.Context

func ExampleNewTransport() {
	// initializes a tracer
	tracer, _ := zipkin.NewTracer(nil)

	// initializes the transport which is going to create the spans per each request
	transport, err := NewTransport(tracer)
	if err != nil {
		log.Fatalf("unable to create the transport: %+v\n", err)
	}

	// creates the client and injects the transport
	c := &http.Client{
		Transport: transport,
	}

	// creates the request
	request, err := http.NewRequest("GET", "http://example.com", nil)
	if err != nil {
		log.Fatalf("unable to create the request: %+v\n", err)
	}

	// it is required to use the `Do` method and pass the request along
	// as the request carries the context whereas in `Post`, `Get`, `Head`
	c.Do(request.WithContext(ctx))

	// Output:
}
