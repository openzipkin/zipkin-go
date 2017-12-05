package zipkin

// Transporter interface can be used to provide the Zipkin Tracer with custom
// implementations to publish Zipkin Span data.
type Transporter interface {
	Close() error   // Close the transporter
	Send(SpanModel) // Send Span data to the transporter
}
