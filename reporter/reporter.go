/*
Package reporter holds the Reporter interface which is used by the Zipkin
Tracer to send finished spans.

Subpackages of package reporter contain officially supported standard
reporter implementations.
*/
package reporter

import "github.com/openzipkin/zipkin-go/model"

// Reporter interface can be used to provide the Zipkin Tracer with custom
// implementations to publish Zipkin Span data.
type Reporter interface {
	Send(model.SpanModel) // Send Span data to the reporter
	Close() error         // Close the reporter
}
