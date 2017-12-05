/*
Package transport holds the Transporter interface which is used by the Zipkin
Tracer to send finished spans.

Subpackages of package transport contain officially supported standard
transport implementations.
*/
package transport

import "github.com/openzipkin/zipkin-go/model"

// Transporter interface can be used to provide the Zipkin Tracer with custom
// implementations to publish Zipkin Span data.
type Transporter interface {
	Close() error         // Close the transporter
	Send(model.SpanModel) // Send Span data to the transporter
}
