// +build go1.9

package grpc

import (
	"strings"

	"google.golang.org/grpc/stats"
)

func spanName(rti *stats.RPCTagInfo) string {
	name := strings.TrimPrefix(rti.FullMethodName, "/")
	name = strings.Replace(name, "/", ".", -1)
	return name
}
