package zipkin

// Extractor function signature
type Extractor func() (*SpanContext, error)

// Injector function signature
type Injector func(SpanContext) error
