package grpc

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"

	"google.golang.org/grpc/stats"

	"github.com/openzipkin/zipkin-go"
)

// A RPCHandler can be registered using WithClientRPCHandler to intercept calls to HandleRPC of a
// handler for additional span customization.
type RPCHandler func(span zipkin.Span, rpcStats stats.RPCStats)

func spanName(rti *stats.RPCTagInfo) string {
	name := strings.TrimPrefix(rti.FullMethodName, "/")
	name = strings.Replace(name, "/", ".", -1)
	return name
}

func handleRpc(span zipkin.Span, rs stats.RPCStats, handlers []RPCHandler) {
	for _, h := range handlers {
		h(span, rs)
	}

	switch rs := rs.(type) {
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
