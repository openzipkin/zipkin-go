package b3

import (
	"strconv"

	"github.com/openzipkin/zipkin-go/model"
)

func parseHeaders(
	traceIDHeader, spanIDHeader, parentSpanIDHeader, sampledHeader, flagsHeader string,
) (*model.SpanContext, error) {
	var (
		err           error
		spanID        uint64
		requiredCount int
		sc            = &model.SpanContext{}
	)

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
