// Copyright 2021 The OpenZipkin Authors
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

package zipkin_proto3_test

import (
	"bytes"
	"encoding/json"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	zipkinmodel "github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/proto/zipkin_proto3"
)

func TestParseSpans(t *testing.T) {
	// 1. Generate some spans then serialize them with protobuf
	protoBlob, err := proto.Marshal(payloadFromWild)
	if err != nil {
		t.Fatalf("Failed to parse payload from wild: %v", err)
	}

	got, err := zipkin_proto3.ParseSpans(protoBlob, true)
	if err != nil {
		t.Fatalf("Failed to parse spans from protobuf blob: %v", err)
	}

	want := []*zipkinmodel.SpanModel{
		{
			SpanContext: zipkinmodel.SpanContext{
				TraceID: zipkinmodel.TraceID{
					High: 0x7F6F5F4F3F2F1F0F,
					Low:  0xF7F6F5F4F3F2F1F0,
				},
				ID:       0xF7F6F5F4F3F2F1F0,
				ParentID: idPtr(0xF7F6F5F4F3F2F1F0),
				Debug:    true,
			},
			Name:      "ProtoSpan1",
			Timestamp: now,
			Duration:  12 * time.Second,
			Shared:    false,
			Kind:      zipkinmodel.Consumer,
			LocalEndpoint: &zipkinmodel.Endpoint{
				ServiceName: "svc-1",
				IPv4:        net.IP{0xC0, 0xA8, 0x00, 0x01},
				Port:        8009,
			},
			RemoteEndpoint: &zipkinmodel.Endpoint{
				ServiceName: "memcached",
				IPv6:        net.IP{0xFE, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x14, 0x53, 0xa7, 0x7c, 0xda, 0x4d, 0xd2, 0x1b},
				Port:        11211,
			},
		},
		{
			SpanContext: zipkinmodel.SpanContext{
				TraceID: zipkinmodel.TraceID{
					High: 0x7A6A5A4A3A2A1A0A,
					Low:  0xC7C6C5C4C3C2C1C0,
				},
				ID:       0x6766656463626160,
				ParentID: idPtr(0x1716151413121110),
				Debug:    true,
			},
			Name:      "CacheWarmUp",
			Timestamp: minus10hr5ms,
			Kind:      zipkinmodel.Producer,
			Duration:  7 * time.Second,
			LocalEndpoint: &zipkinmodel.Endpoint{
				ServiceName: "search",
				IPv4:        net.IP{0x0A, 0x00, 0x00, 0x0D},
				Port:        8009,
			},
			RemoteEndpoint: &zipkinmodel.Endpoint{
				ServiceName: "redis",
				IPv6:        net.IP{0xFE, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x14, 0x53, 0xa7, 0x7c, 0xda, 0x4d, 0xd2, 0x1b},
				Port:        6379,
			},
			Annotations: []zipkinmodel.Annotation{
				{
					Timestamp: minus10hr5ms,
					Value:     "DB reset",
				},
				{
					Timestamp: minus10hr5ms,
					Value:     "GC Cycle 39",
				},
			},
		},
	}

	if g, w := len(got), len(want); g != w {
		t.Errorf("Number of spans doesn't match:: Got %d Want %d", g, w)
	}

	if !reflect.DeepEqual(got, want) {
		gj, _ := json.Marshal(got)
		wj, _ := json.Marshal(want)
		if len(gj) < 100 {
			t.Errorf("Unexpected found short output: %v", len(gj))
		}
		if !bytes.Equal(gj, wj) {
			t.Errorf("Failed to get roundtripped spans\nGot: %s\nWant:%s\n", gj, wj)
		}
	}
}

func TestParseSpans_failures(t *testing.T) {
	tests := []struct {
		spans   []*zipkin_proto3.Span
		wantErr string
	}{
		{
			spans: []*zipkin_proto3.Span{
				{TraceId: nil},
			},
			wantErr: "invalid TraceID: has length 0 yet wanted length 16",
		},
		{
			spans: []*zipkin_proto3.Span{
				{TraceId: []byte{0x01, 0x02}},
			},
			wantErr: "invalid TraceID: has length 2 yet wanted length 16",
		},
		{
			spans: []*zipkin_proto3.Span{
				{
					TraceId: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
					Id:      nil,
				},
			},
			wantErr: "expected a non-nil SpanID",
		},
		{
			spans: []*zipkin_proto3.Span{
				{
					TraceId: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
					Id:      []byte{0x01, 0x02},
				},
			},
			wantErr: "invalid SpanID: has length 2 yet wanted length 8",
		},
		{
			spans: []*zipkin_proto3.Span{
				{
					TraceId:  []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
					ParentId: []byte{0x01, 0x02},
				},
			},
			wantErr: "invalid ParentID: has length 2 yet wanted length 8",
		},
		{
			spans: []*zipkin_proto3.Span{
				{
					TraceId:  []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
					ParentId: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
				},
			},
			wantErr: "expected a non-nil SpanID",
		},
		{
			spans: []*zipkin_proto3.Span{
				{
					TraceId:  []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
					ParentId: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
					Id:       []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07},
				},
			},
			wantErr: "invalid SpanID: has length 7 yet wanted length 8",
		},
	}

	for i, tt := range tests {
		payload := &zipkin_proto3.ListOfSpans{Spans: tt.spans}
		protoBlob, err := proto.Marshal(payload)
		if err != nil {
			t.Errorf("Test #%d: Failed to serialize ProtoPayload: %v", i, err)
			continue
		}

		zms, err := zipkin_proto3.ParseSpans(protoBlob, true)
		if err == nil {
			t.Errorf("#%d: unexpectedly passed and got span\n%#v", i, zms)
			continue
		}
		if zms != nil {
			t.Errorf("#%d: inconsistency, ParseSpan is non-nil and so is the error", i)
		}
		if !strings.Contains(err.Error(), tt.wantErr) {
			t.Errorf("#%d: Mismatched errors\nGot: (%q)\nWant:(%q)", i, err, tt.wantErr)
		}
	}
}

func idPtr(id zipkinmodel.ID) *zipkinmodel.ID { return &id }

var (
	now          = time.Date(2018, 10, 31, 19, 43, 35, 789, time.UTC).Round(time.Microsecond)
	minus10hr5ms = now.Add(-(10*time.Hour + 5*time.Millisecond)).Round(time.Microsecond)
)

var payloadFromWild = &zipkin_proto3.ListOfSpans{
	Spans: []*zipkin_proto3.Span{
		{
			TraceId:   []byte{0x7F, 0x6F, 0x5F, 0x4F, 0x3F, 0x2F, 0x1F, 0x0F, 0xF7, 0xF6, 0xF5, 0xF4, 0xF3, 0xF2, 0xF1, 0xF0},
			Id:        []byte{0xF7, 0xF6, 0xF5, 0xF4, 0xF3, 0xF2, 0xF1, 0xF0},
			ParentId:  []byte{0xF7, 0xF6, 0xF5, 0xF4, 0xF3, 0xF2, 0xF1, 0xF0},
			Name:      "ProtoSpan1",
			Kind:      zipkin_proto3.Span_CONSUMER,
			Timestamp: uint64(now.UnixNano() / 1e3),
			Duration:  12e6,
			LocalEndpoint: &zipkin_proto3.Endpoint{
				ServiceName: "svc-1",
				Ipv4:        []byte{0xC0, 0xA8, 0x00, 0x01},
				Port:        8009,
			},
			RemoteEndpoint: &zipkin_proto3.Endpoint{
				ServiceName: "memcached",
				Ipv6:        []byte{0xFE, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x14, 0x53, 0xa7, 0x7c, 0xda, 0x4d, 0xd2, 0x1b},
				Port:        11211,
			},
		},
		{
			TraceId:   []byte{0x7A, 0x6A, 0x5A, 0x4A, 0x3A, 0x2A, 0x1A, 0x0A, 0xC7, 0xC6, 0xC5, 0xC4, 0xC3, 0xC2, 0xC1, 0xC0},
			Id:        []byte{0x67, 0x66, 0x65, 0x64, 0x63, 0x62, 0x61, 0x60},
			ParentId:  []byte{0x17, 0x16, 0x15, 0x14, 0x13, 0x12, 0x11, 0x10},
			Name:      "CacheWarmUp",
			Kind:      zipkin_proto3.Span_PRODUCER,
			Timestamp: uint64(minus10hr5ms.UnixNano() / 1e3),
			Duration:  7e6,
			LocalEndpoint: &zipkin_proto3.Endpoint{
				ServiceName: "search",
				Ipv4:        []byte{0x0A, 0x00, 0x00, 0x0D},
				Port:        8009,
			},
			RemoteEndpoint: &zipkin_proto3.Endpoint{
				ServiceName: "redis",
				Ipv6:        []byte{0xFE, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x14, 0x53, 0xa7, 0x7c, 0xda, 0x4d, 0xd2, 0x1b},
				Port:        6379,
			},
			Annotations: []*zipkin_proto3.Annotation{
				{
					Timestamp: uint64(minus10hr5ms.UnixNano() / 1e3),
					Value:     "DB reset",
				},
				{
					Timestamp: uint64(minus10hr5ms.UnixNano() / 1e3),
					Value:     "GC Cycle 39",
				},
			},
		},
	},
}
