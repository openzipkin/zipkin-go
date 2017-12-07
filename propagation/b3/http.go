package b3

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/propagation"
)

// ExtractHTTP will extract a span.Context from the HTTP Request if found in
// B3 header format
func ExtractHTTP(r *http.Request) propagation.Extractor {
	return func() (*model.SpanContext, error) {
		var (
			err                error
			spanID             uint64
			requiredCount      int
			traceIDHeader      = r.Header.Get(b3TraceID)
			spanIDHeader       = r.Header.Get(b3SpanID)
			parentSpanIDHeader = r.Header.Get(b3ParentSpanID)
			sampledHeader      = r.Header.Get(b3Sampled)
			flagsHeader        = r.Header.Get(b3Flags)
		)

		sc := &model.SpanContext{}

		switch sampledHeader {
		case "0":
			sampled := false
			sc.Sampled = &sampled
		case "1":
			sampled := true
			sc.Sampled = &sampled
		case "":
			// sc.Sampled = nil
		default:
			return nil, ErrInvalidSampledHeader
		}

		switch flagsHeader {
		case "", "0":
			// sc.Debug = false
		case "1":
			sc.Debug = true
		default:
			return nil, ErrInvalidFlagsHeader
		}

		if traceIDHeader != "" {
			requiredCount++
			if sc.TraceID, err = model.TraceIDFromHex(traceIDHeader); err != nil {
				return nil, ErrInvalidTraceIDHeader
			}
		}

		if spanIDHeader != "" {
			requiredCount++
			if spanID, err = strconv.ParseUint(spanIDHeader, 16, 64); err != nil {
				return nil, ErrInvalidSpanIDHeader
			}
			sc.ID = model.ID(spanID)
		}

		if requiredCount != 0 && requiredCount != 2 {
			return nil, ErrInvalidScope
		}

		if requiredCount == 2 && parentSpanIDHeader != "" {
			if spanID, err = strconv.ParseUint(parentSpanIDHeader, 16, 64); err != nil {
				return nil, ErrInvalidParentSpanIDHeader
			}
			parentSpanID := model.ID(spanID)
			sc.ParentID = &parentSpanID
		}

		return sc, nil
	}
}

// InjectHTTP will inject a span.Context into a HTTP Request
func InjectHTTP(r *http.Request) propagation.Injector {
	return func(sc model.SpanContext) error {
		if (model.SpanContext{}) == sc {
			return ErrEmptyContext
		}

		if sc.Debug {
			r.Header.Set(b3Flags, "1")
		} else if sc.Sampled != nil {
			// Debug is encoded as X-B3-Flags: 1. Since Debug implies Sampled,
			// so don't also send "X-B3-Sampled: 1".
			if *sc.Sampled {
				r.Header.Set(b3Sampled, "1")
			} else {
				r.Header.Set(b3Sampled, "0")
			}
		}

		if !sc.TraceID.Empty() {
			r.Header.Set(b3TraceID, sc.TraceID.ToHex())
		}

		if sc.ID > 0 {
			r.Header.Set(b3SpanID, fmt.Sprintf("%016x", sc.ID))
		}

		if sc.ParentID != nil {
			r.Header.Set(b3ParentSpanID, fmt.Sprintf("%016x", *sc.ParentID))
		}

		return nil
	}
}
