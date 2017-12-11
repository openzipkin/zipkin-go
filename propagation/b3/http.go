package b3

import (
	"fmt"
	"net/http"

	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/propagation"
)

// ExtractHTTP will extract a span.Context from the HTTP Request if found in
// B3 header format.
func ExtractHTTP(r *http.Request) propagation.Extractor {
	return func() (*model.SpanContext, error) {
		var (
			traceIDHeader      = r.Header.Get(TraceID)
			spanIDHeader       = r.Header.Get(SpanID)
			parentSpanIDHeader = r.Header.Get(ParentSpanID)
			sampledHeader      = r.Header.Get(Sampled)
			flagsHeader        = r.Header.Get(Flags)
		)

		return parseHeaders(
			traceIDHeader, spanIDHeader, parentSpanIDHeader, sampledHeader,
			flagsHeader,
		)
	}
}

// InjectHTTP will inject a span.Context into a HTTP Request
func InjectHTTP(r *http.Request) propagation.Injector {
	return func(sc model.SpanContext) error {
		if (model.SpanContext{}) == sc {
			return ErrEmptyContext
		}

		if sc.Debug {
			r.Header.Set(Flags, "1")
		} else if sc.Sampled != nil {
			// Debug is encoded as X-B3-Flags: 1. Since Debug implies Sampled,
			// so don't also send "X-B3-Sampled: 1".
			if *sc.Sampled {
				r.Header.Set(Sampled, "1")
			} else {
				r.Header.Set(Sampled, "0")
			}
		}

		if !sc.TraceID.Empty() {
			r.Header.Set(TraceID, sc.TraceID.ToHex())
		}

		if sc.ID > 0 {
			r.Header.Set(SpanID, fmt.Sprintf("%016x", sc.ID))
		}

		if sc.ParentID != nil {
			r.Header.Set(ParentSpanID, fmt.Sprintf("%016x", *sc.ParentID))
		}

		return nil
	}
}
