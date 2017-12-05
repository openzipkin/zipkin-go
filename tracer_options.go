package zipkin

import "errors"

// Tracer Option Errors
var (
	ErrInvalidEndpoint             = errors.New("requires valid local endpoint")
	ErrInvalidExtractFailurePolicy = errors.New("invalid extract failure policy provided")
)

// ExtractFailurePolicy deals with Extraction errors
type ExtractFailurePolicy int

// ExtractFailurePolicyOptions
const (
	ExtractFailurePolicyRestart ExtractFailurePolicy = iota
	ExtractFailurePolicyError
	ExtractFailurePolicyTagAndRestart
)

// TracerOption allows for functional options to adjust behavior of the Tracer
// to be created with NewTracer().
type TracerOption func(o *TracerOptions) error

// TracerOptions for a Tracer instance.
type TracerOptions struct {
	localEndpoint        *Endpoint
	sharedSpans          bool
	sampler              Sampler
	generate             IDGenerator
	defaultTags          map[string]string
	unsampledNoop        bool
	extractFailurePolicy ExtractFailurePolicy
	transport            Transporter
}

// WithLocalEndpoint sets the local endpoint of the tracer.
func WithLocalEndpoint(e *Endpoint) TracerOption {
	return func(o *TracerOptions) error {
		if e == nil {
			return ErrInvalidEndpoint
		}
		o.localEndpoint = e
		return nil
	}
}

// WithExtractFailurePolicy allows one to set the ExtractFailurePolicy.
func WithExtractFailurePolicy(p ExtractFailurePolicy) TracerOption {
	return func(o *TracerOptions) error {
		if p < 0 || p > ExtractFailurePolicyTagAndRestart {
			return ErrInvalidExtractFailurePolicy
		}
		o.extractFailurePolicy = p
		return nil
	}
}

// WithNoopSpan if set to true will switch to a NoopSpan implementation
// if the trace is not sampled.
func WithNoopSpan(unsampledNoop bool) TracerOption {
	return func(o *TracerOptions) error {
		o.unsampledNoop = unsampledNoop
		return nil
	}
}

// WithSharedSpans allows to place client-side and server-side annotations
// for a RPC call in the same span (Zipkin V1 behavior) or different spans
// (more in line with other tracing solutions). By default this Tracer
// uses shared host spans (so client-side and server-side in the same span).
func WithSharedSpans(val bool) TracerOption {
	return func(o *TracerOptions) error {
		o.sharedSpans = val
		return nil
	}
}

// WithSampler allows one to set a Sampler function
func WithSampler(sampler Sampler) TracerOption {
	return func(o *TracerOptions) error {
		o.sampler = sampler
		return nil
	}
}

// WithIDGenerator allows one to set a custom ID Generator
func WithIDGenerator(generator IDGenerator) TracerOption {
	return func(o *TracerOptions) error {
		o.generate = generator
		return nil
	}
}

// WithTags allows one to set default tags to be added to each created span
func WithTags(tags map[string]string) TracerOption {
	return func(o *TracerOptions) error {
		for k, v := range tags {
			o.defaultTags[k] = v
		}
		return nil
	}
}
