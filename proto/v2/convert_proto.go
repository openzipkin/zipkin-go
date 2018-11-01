// Copyright 2018 The OpenZipkin Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*
Package zipkin_proto3 adds support for the Zipkin protobuf definition to allow
Go applications to consume model.SpanModel from protobuf serialized data.
*/
package zipkin_proto3

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/golang/protobuf/proto"

	zipkinmodel "github.com/openzipkin/zipkin-go/model"
)

// ParseSpans parses zipkinmodel.SpanModel values from data serialized by Protobuf3.
// debugWasSet is a boolean that toggles the Debug field of each Span. Its value
// is usually retrieved from the transport headers when the "X-B3-Flags" header has a value of 1.
func ParseSpans(protoBlob []byte, debugWasSet bool) (zss []*zipkinmodel.SpanModel, err error) {
	var listOfSpans ListOfSpans
	if err := proto.Unmarshal(protoBlob, &listOfSpans); err != nil {
		return nil, err
	}
	for _, zps := range listOfSpans.Spans {
		zms, err := protoSpanToModelSpan(zps, debugWasSet)
		if err == nil && zms != nil {
			// TODO: perhaps record or combine these errors?
			zss = append(zss, zms)
		}
	}
	return zss, nil
}

var errNilZipkinSpan = errors.New("expecting a non-nil Span")

// This conversion is done so that we can reuse the logic for converting from
//      Zipkin-Go SpanModel --> opencensusProto.Span
// as was done during Zipkin-Go SpanModel JSON deserialization
// instead of reimplementing zipkin_proto3 --> opencensusProto.Span.
// This conversion shouldn't cause any loss of fidelity.
func protoSpanToModelSpan(zps *Span, debugWasSet bool) (*zipkinmodel.SpanModel, error) {
	if zps == nil {
		return nil, errNilZipkinSpan
	}
	traceID, err := zipkinmodel.TraceIDFromHex(fmt.Sprintf("%x", zps.TraceId))
	if err != nil {
		return nil, fmt.Errorf("TraceID: %v", err)
	}

	parentSpanID, _, err := protoSpanIDToModelSpanID(zps.ParentId)
	if err != nil {
		return nil, fmt.Errorf("ParentID: %v", err)
	}
	spanIDPtr, spanIDBlank, err := protoSpanIDToModelSpanID(zps.Id)
	if err != nil {
		return nil, fmt.Errorf("SpanID: %v", err)
	}
	if spanIDBlank || spanIDPtr == nil {
		// This is a logical error
		return nil, errors.New("SpanID: expected a non-nil spanID")
	}

	zmsc := zipkinmodel.SpanContext{
		TraceID:  traceID,
		ID:       *spanIDPtr,
		ParentID: parentSpanID,
		Debug:    debugWasSet,
	}
	zms := &zipkinmodel.SpanModel{
		SpanContext:    zmsc,
		Name:           zps.Name,
		Kind:           zipkinmodel.Kind(zps.Kind.String()),
		Timestamp:      microsToTime(zps.Timestamp),
		Tags:           zps.Tags,
		Duration:       microsToDuration(zps.Duration),
		LocalEndpoint:  protoEndpointToModelEndpoint(zps.LocalEndpoint),
		RemoteEndpoint: protoEndpointToModelEndpoint(zps.RemoteEndpoint),
		Shared:         zps.Shared,
		Annotations:    protoAnnotationToModelAnnotations(zps.Annotations),
	}

	return zms, nil
}

func microsToDuration(us uint64) time.Duration {
	// us to ns; ns are the units of Duration
	return time.Duration(us * 1e3)
}

func protoEndpointToModelEndpoint(zpe *Endpoint) *zipkinmodel.Endpoint {
	if zpe == nil {
		return nil
	}
	return &zipkinmodel.Endpoint{
		ServiceName: zpe.ServiceName,
		IPv4:        net.IP(zpe.Ipv4),
		IPv6:        net.IP(zpe.Ipv6),
		Port:        uint16(zpe.Port),
	}
}

func protoSpanIDToModelSpanID(spanId []byte) (zid *zipkinmodel.ID, blank bool, err error) {
	if len(spanId) == 0 {
		return nil, true, nil
	}
	if len(spanId) != 8 {
		return nil, true, fmt.Errorf("length(%d) != 8", len(spanId))
	}

	// Converting [8]byte --> uint64
	var u64 uint64
	u64 |= uint64(spanId[7]&0xFF) << 0
	u64 |= uint64(spanId[6]&0xFF) << 8
	u64 |= uint64(spanId[5]&0xFF) << 16
	u64 |= uint64(spanId[4]&0xFF) << 24
	u64 |= uint64(spanId[3]&0xFF) << 32
	u64 |= uint64(spanId[2]&0xFF) << 40
	u64 |= uint64(spanId[1]&0xFF) << 48
	u64 |= uint64(spanId[0]&0xFF) << 56
	zid_ := zipkinmodel.ID(u64)
	return &zid_, false, nil
}

func protoAnnotationToModelAnnotations(zpa []*Annotation) (zma []zipkinmodel.Annotation) {
	for _, za := range zpa {
		if za != nil {
			zma = append(zma, zipkinmodel.Annotation{
				Timestamp: microsToTime(za.Timestamp),
				Value:     za.Value,
			})
		}
	}

	if len(zma) == 0 {
		return nil
	}
	return zma
}

func microsToTime(us uint64) time.Time {
	return time.Unix(0, int64(us*1e3))
}
