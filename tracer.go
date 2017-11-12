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
	// set default tracer options
	opts := &TracerOptions{
		sharedSpans: true,
		sampler:     alwaysSample,
		generate:    &RandomID64{},
		defaultTags: make(map[string]string),
		transport:   &NoopTransport{},
	}

	// process functional options
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
	s := &spanImpl{
		SpanModel: SpanModel{
			Kind:          kind,
			Name:          name,
			Timestamp:     time.Now(),
			LocalEndpoint: t.options.localEndpoint,
			Annotations:   make([]Annotation, 0),
			Tags:          make(map[string]string),
		},
		tracer: t,
	}

	for _, option := range options {
		option(t, s)
	}

	// test if extraction resulted in an error
	if s.SpanContext.err != nil {
		switch t.options.extractFailurePolicy {
		case ExtractFailurePolicyRestart:
		case ExtractFailurePolicyError:
			panic(s.SpanContext.err)
		case ExtractFailurePolicyTagAndRestart:
			s.Tags["error.extract"] = s.SpanContext.err.Error()
		default:
			panic(ErrInvalidExtractFailurePolicy)
		}
		// restart the trace
		s.SpanContext.TraceID = t.options.generate.TraceID()
		s.SpanContext.ID = t.options.generate.SpanID()
		s.SpanContext.ParentID = nil
		s.SpanContext.err = nil
	} else if s.SpanContext.TraceID.Empty() || s.SpanContext.ID == 0 {
		// create root span
		s.SpanContext.TraceID = t.options.generate.TraceID()
		s.SpanContext.ID = t.options.generate.SpanID()
	}

	if !s.SpanContext.Debug && s.Sampled == nil {
		// deferred sampled context found, invoke sampler
		sampled := t.options.sampler(s.SpanContext.TraceID.Low)
		s.SpanContext.Sampled = &sampled
		s.isSampled = sampled
	} else {
		s.isSampled = s.SpanContext.Debug || *s.Sampled
	}

	if t.options.unsampledNoop && !s.SpanContext.Debug &&
		(s.SpanContext.Sampled == nil || !*s.SpanContext.Sampled) {
		// trace not being sampled and noop requested
		return &noopSpan{
			SpanContext: s.SpanContext,
		}
	}

	// add default tags to span
	for k, v := range t.options.defaultTags {
		s.Tag(k, v)
	}

	return s
}

// Extract extracts a SpanContext using the provided Extractor function
func (t *Tracer) Extract(extractor Extractor) (sc SpanContext) {
	psc, err := extractor()
	if psc != nil {
		sc = *psc
	}
	sc.err = err
	return
}
