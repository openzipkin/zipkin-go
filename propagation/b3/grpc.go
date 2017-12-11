package b3

import (
	"fmt"

	"google.golang.org/grpc/metadata"

	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/propagation"
)

// ExtractGRPC will extract a span.Context from the gRPC Request metadata if
// found in B3 header format.
func ExtractGRPC(md *metadata.MD) propagation.Extractor {
	return func() (*model.SpanContext, error) {
		var (
			traceIDHeader      = getGRPCHeader(md, b3TraceID)
			spanIDHeader       = getGRPCHeader(md, b3SpanID)
			parentSpanIDHeader = getGRPCHeader(md, b3ParentSpanID)
			sampledHeader      = getGRPCHeader(md, b3Sampled)
			flagsHeader        = getGRPCHeader(md, b3Flags)
		)

		return parseHeaders(
			traceIDHeader, spanIDHeader, parentSpanIDHeader, sampledHeader,
			flagsHeader,
		)
	}
}

// InjectGRPC will inject a span.Context into gRPC metadata.
func InjectGRPC(md *metadata.MD) propagation.Injector {
	return func(sc model.SpanContext) error {
		if (model.SpanContext{}) == sc {
			return ErrEmptyContext
		}

		if sc.Debug {
			setGRPCHeader(md, b3Flags, "1")
		} else if sc.Sampled != nil {
			// Debug is encoded as X-B3-Flags: 1. Since Debug implies Sampled,
			// so don't also send "X-B3-Sampled: 1".
			if *sc.Sampled {
				setGRPCHeader(md, b3Sampled, "1")
			} else {
				setGRPCHeader(md, b3Sampled, "0")
			}
		}

		if !sc.TraceID.Empty() {
			setGRPCHeader(md, b3TraceID, sc.TraceID.ToHex())
		}

		if sc.ID > 0 {
			setGRPCHeader(md, b3SpanID, fmt.Sprintf("%016x", sc.ID))
		}

		if sc.ParentID != nil {
			setGRPCHeader(md, b3ParentSpanID, fmt.Sprintf("%016x", *sc.ParentID))
		}

		return nil
	}
}

func getGRPCHeader(md *metadata.MD, key string) string {
	v := (*md)[key]
	if len(v) < 1 {
		return ""
	}
	return v[0]
}

func setGRPCHeader(md *metadata.MD, key, value string) {
	(*md)[key] = append((*md)[key], value)
}
