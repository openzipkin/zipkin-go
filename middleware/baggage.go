package middleware

import "github.com/openzipkin/zipkin-go/model"

// BaggageHandler holds the interface for server and client middlewares
// interacting with baggage context propagation implementations.
// A reference implementation can be found in package:
// github.com/openzipkin/zipkin-go/propagation/baggage
type BaggageHandler interface {
	// New returns a fresh BaggageFields implementation primed for usage in a
	// request lifecycle.
	// This method needs to be called by incoming transport middlewares. See
	// middlewares/grpc/server.go and middlewares/http/server.go
	New() model.BaggageFields
}
