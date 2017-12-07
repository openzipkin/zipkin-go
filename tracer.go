package zipkin

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/openzipkin/zipkin-go/idgenerator"
	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/propagation"
	"github.com/openzipkin/zipkin-go/reporter"
)

// Tracer is our Zipkin tracer implementation.
type Tracer struct {
	noop                 int32 // used as atomic bool (1 = true, 0 = false)
	localEndpoint        *model.Endpoint
	sharedSpans          bool
	sampler              Sampler
	generate             idgenerator.IDGenerator
	defaultTags          map[string]string
	unsampledNoop        bool
	extractFailurePolicy ExtractFailurePolicy
	reporter             reporter.Reporter
}

// NewTracer returns a new Zipkin Tracer.
func NewTracer(reporter reporter.Reporter, options ...TracerOption) (*Tracer, error) {
	// set default tracer options
	t := &Tracer{
		sharedSpans: true,
		sampler:     alwaysSample,
		generate:    idgenerator.NewRandom64(),
		defaultTags: make(map[string]string),
		reporter:    reporter,
	}

	// process functional options
	for _, option := range options {
		if err := option(t); err != nil {
			return nil, err
		}
	}

	return t, nil
}

// StartSpanFromContext creates and starts a span using the span found in
// context as parent. If no parent span is found a root span is created.
func (t *Tracer) StartSpanFromContext(ctx context.Context, name string, options ...SpanOption) (Span, context.Context) {
	if parentSpan := SpanFromContext(ctx); parentSpan != nil {
		options = append(options, Parent(parentSpan.Context()))
	}
	span := t.StartSpan(name, options...)
	return span, NewContext(ctx, span)
}

// StartSpan creates and starts a span.
func (t *Tracer) StartSpan(name string, options ...SpanOption) Span {
	if atomic.LoadInt32(&t.noop) == 1 {
		return &noopSpan{}
	}
	s := &spanImpl{
		SpanModel: model.SpanModel{
			Kind:          model.Undetermined,
			Name:          name,
			Timestamp:     time.Now(),
			LocalEndpoint: t.localEndpoint,
			Annotations:   make([]model.Annotation, 0),
			Tags:          make(map[string]string),
		},
		tracer: t,
	}

	for _, option := range options {
		option(t, s)
	}

	if (model.SpanContext{}) != s.SpanContext {
		// we received a parent SpanContext
		if t.sharedSpans && s.Kind == model.Server {
			// join span
			s.Shared = true
		} else {
			// regular child span
			parentID := s.ID
			s.ParentID = &parentID
			s.ID = t.generate.SpanID(model.TraceID{})
		}
	}

	// test if extraction resulted in an error
	if s.SpanContext.Err != nil {
		switch t.extractFailurePolicy {
		case ExtractFailurePolicyRestart:
		case ExtractFailurePolicyError:
			panic(s.SpanContext.Err)
		case ExtractFailurePolicyTagAndRestart:
			s.Tags["error.extract"] = s.SpanContext.Err.Error()
		default:
			panic(ErrInvalidExtractFailurePolicy)
		}
		// restart the trace
		s.SpanContext.TraceID = t.generate.TraceID()
		s.SpanContext.ID = t.generate.SpanID(s.SpanContext.TraceID)
		s.SpanContext.ParentID = nil
	} else if s.SpanContext.TraceID.Empty() || s.SpanContext.ID == 0 {
		// create root span
		s.SpanContext.TraceID = t.generate.TraceID()
		s.SpanContext.ID = t.generate.SpanID(s.SpanContext.TraceID)
	}

	if !s.SpanContext.Debug && s.Sampled == nil {
		// deferred sampled context found, invoke sampler
		sampled := t.sampler(s.SpanContext.TraceID.Low)
		s.SpanContext.Sampled = &sampled
		if sampled {
			s.isSampled = 1
		}
	} else {
		if s.SpanContext.Debug || *s.Sampled {
			s.isSampled = 1
		}
	}

	if t.unsampledNoop && !s.SpanContext.Debug &&
		(s.SpanContext.Sampled == nil || !*s.SpanContext.Sampled) {
		// trace not being sampled and noop requested
		return &noopSpan{
			SpanContext: s.SpanContext,
		}
	}

	// add default tags to span
	for k, v := range t.defaultTags {
		s.Tag(k, v)
	}

	return s
}

// Extract extracts a SpanContext using the provided Extractor function.
func (t *Tracer) Extract(extractor propagation.Extractor) (sc model.SpanContext) {
	if atomic.LoadInt32(&t.noop) == 1 {
		return
	}
	psc, err := extractor()
	if psc != nil {
		sc = *psc
	}
	sc.Err = err
	return
}

// SetNoop allows for killswitch behavior. If set to true the tracer will return
// noopSpans and all data is dropped. This allows operators to stop tracing in
// risk scenarios. Set back to false to resume tracing.
func (t *Tracer) SetNoop(noop bool) {
	if noop {
		atomic.CompareAndSwapInt32(&t.noop, 0, 1)
	} else {
		atomic.CompareAndSwapInt32(&t.noop, 1, 0)
	}
}
