// Copyright 2019 The OpenZipkin Authors
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
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	zipkinmodel "github.com/openzipkin/zipkin-go/model"
	zipkin_proto3 "github.com/openzipkin/zipkin-go/proto/v2"
)

func TestExportSpans(t *testing.T) {
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

	protoBlob, err := zipkin_proto3.SpanSerializer{}.Serialize(want)
	if err != nil {
		t.Fatalf("Failed to parse spans from protobuf blob: %v", err)
	}

	if got, _ := zipkin_proto3.ParseSpans(protoBlob, true); !reflect.DeepEqual(want, got) {
		t.Errorf("conversion error!\nWANT:\n%s\n\nGOT:\n%s\n", spew.Sdump(want), spew.Sdump(got))
	}
}
