package b3

import (
	"strconv"
	"strings"

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

	switch strings.ToLower(sampledHeader) {
	case "0", "false":
		sampled := false
		sc.Sampled = &sampled
	case "1", "true":
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
		if sc.Sampled != nil {
			sc.Sampled = nil
		}
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

	if parentSpanIDHeader != "" {
		if requiredCount == 0 {
			return nil, ErrInvalidScopeParent
		}
		if spanID, err = strconv.ParseUint(parentSpanIDHeader, 16, 64); err != nil {
			return nil, ErrInvalidParentSpanIDHeader
		}
		parentSpanID := model.ID(spanID)
		sc.ParentID = &parentSpanID
	}

	return sc, nil
}
