package zipkin

// TracerOption allows for functional options.
type TracerOption func(o *TracerOptions) error

// TracerOptions for a Tracer instance.
type TracerOptions struct {
	localEndpoint *Endpoint
	sharedSpans   bool
	sampler       Sampler
	generate      IDGenerator
	defaultTags   map[string]string
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
