package zipkin

// Transporter interface can be used to provide the Zipkin Tracer with custom
// implementations to publish Zipkin Span data.
type Transporter interface {
	Close() error   // Close the transporter
	Send(SpanModel) // Send Span data to the transporter
}

// noopTransport provides noop implementation if no transporter was provided to
// the tracer.
type noopTransport struct{}

// Send implements Transporter
func (t *noopTransport) Send(_ SpanModel) {}

// Close implements Transporter
func (t *noopTransport) Close() error { return nil }
