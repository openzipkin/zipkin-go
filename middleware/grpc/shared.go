// Copyright 2020 The OpenZipkin Authors
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

package grpc

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"

	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
)

type handleRPCParser struct {
	inPayload  func(*stats.InPayload, zipkin.SpanCustomizer)
	inTrailer  func(*stats.InTrailer, zipkin.SpanCustomizer)
	inHeader   func(*stats.InHeader, zipkin.SpanCustomizer)
	outPayload func(*stats.OutPayload, zipkin.SpanCustomizer)
	outTrailer func(*stats.OutTrailer, zipkin.SpanCustomizer)
	outHeader  func(*stats.OutHeader, zipkin.SpanCustomizer)
}

func (h handleRPCParser) IsZero() bool {
	return h.inPayload == nil &&
		h.inTrailer == nil &&
		h.inHeader == nil &&
		h.outPayload == nil &&
		h.outTrailer == nil &&
		h.outHeader == nil
}

// A RPCHandler can be registered using WithClientRPCHandler or WithServerRPCHandler to intercept calls to HandleRPC of
// a handler for additional span customization.
type RPCHandler func(span zipkin.Span, rpcStats stats.RPCStats)

func spanName(rti *stats.RPCTagInfo) string {
	name := strings.TrimPrefix(rti.FullMethodName, "/")
	name = strings.Replace(name, "/", ".", -1)
	return name
}

func handleRPC(ctx context.Context, rs stats.RPCStats, h handleRPCParser) {
	span := zipkin.SpanFromContext(ctx)
	if span.IsNoop() {
		return
	}

	var spanCustomizer zipkin.SpanCustomizer
	if !h.IsZero() {
		spanCustomizer = zipkin.WrapWithSpanCustomizerShield(span)
	}

	switch rs := rs.(type) {
	case *stats.InPayload:
		if h.inPayload != nil {
			h.inPayload(rs, spanCustomizer)
		}
	case *stats.InHeader:
		if h.inHeader != nil {
			h.inHeader(rs, spanCustomizer)
		}
	case *stats.InTrailer:
		if h.inTrailer != nil {
			h.inTrailer(rs, spanCustomizer)
		}
	case *stats.OutPayload:
		if h.outPayload != nil {
			h.outPayload(rs, spanCustomizer)
		}
	case *stats.OutHeader:
		if h.outHeader != nil {
			h.outHeader(rs, spanCustomizer)
		}
	case *stats.OutTrailer:
		if h.outTrailer != nil {
			h.outTrailer(rs, spanCustomizer)
		}
	case *stats.End:
		s, ok := status.FromError(rs.Error)
		// rs.Error should always be convertable to a status, this is just a defensive check.
		if ok {
			if s.Code() != codes.OK {
				// Uppercase for consistency with Brave
				c := strings.ToUpper(s.Code().String())
				span.Tag("grpc.status_code", c)
				zipkin.TagError.Set(span, c)
			}
		} else {
			zipkin.TagError.Set(span, rs.Error.Error())
		}
		span.Finish()
	}
}

func remoteEndpointFromContext(ctx context.Context, name string) *model.Endpoint {
	remoteAddr := ""

	p, ok := peer.FromContext(ctx)
	if ok {
		remoteAddr = p.Addr.String()
	}

	ep, _ := zipkin.NewEndpoint(name, remoteAddr)
	return ep
}
