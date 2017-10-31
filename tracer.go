package zipkin

import (
	"time"

	"github.com/openzipkin/zipkin-go/kind"
)

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
		defaultTags: make(map[string]string),
	}

	for _, option := range options {
		if err := option(opts); err != nil {
			return nil, err
		}
	}

	return &Tracer{options: *opts}, nil
}

// StartSpan creates and starts a span
func (t *Tracer) StartSpan(
	name string, kind kind.Type, options ...SpanOption,
) Span {
	s := &span{
		Name:          name,
		Kind:          kind,
		Timestamp:     time.Now(),
		Shared:        t.options.sharedSpans,
		LocalEndpoint: t.options.localEndpoint,
		Tags:          make(map[string]string),
	}

	for k, v := range t.options.defaultTags {
		s.Tag(k, v)
	}

	for _, option := range options {
		option(t, s)
	}

	if s.SpanContext.Empty() {
		// our SpanContext is empty, create root span
		s.SpanContext.TraceID = t.options.generate.TraceID()
		s.SpanContext.ID = t.options.generate.SpanID()
		// invoke sampler
		sampled := t.options.sampler(s.SpanContext.TraceID.Low)
		s.SpanContext.Sampled = &sampled
	}

	return s
}
