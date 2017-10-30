package zipkin

import "time"

// Tracer is our Zipkin tracer implementation.
type Tracer struct {
	options TracerOptions
}

// NewTracer returns a new Zipkin Tracer.
func NewTracer(options ...TracerOption) (*Tracer, error) {
	opts := &TracerOptions{
		sharedSpans: true,
		sampler:     alwaysSample,
		generate:    &RandomID64{},
	}

	for _, option := range options {
		if err := option(opts); err != nil {
			return nil, err
		}
	}

	return &Tracer{options: *opts}, nil
}

// StartSpan creates and starts a span
func (t *Tracer) StartSpan(name string) Span {
	return &span{
		SpanContext: SpanContext{
			TraceID: t.options.generate.TraceID(),
			ID:      t.options.generate.SpanID(),
		},
		LocalEndpoint: t.options.localEndpoint,
		Timestamp:     time.Now(),
		Name:          name,
		Tags:          make(map[string]string),
	}
}
