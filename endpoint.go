package zipkin

import (
	"net"
	"strconv"
	"strings"
)

// Endpoint holds the network context of a node in the service graph.
type Endpoint struct {
	ServiceName string `json:"serviceName,omitempty"`
	IPv4        net.IP `json:"ipv4,omitempty"`
	IPv6        net.IP `json:"ipv6,omitempty"`
	Port        int    `json:"port,omitempty"`
}

// NewEndpoint creates a new endpoint given the provided serviceName and
// hostPort.
func NewEndpoint(serviceName string, hostPort string) (*Endpoint, error) {
	e := &Endpoint{
		ServiceName: serviceName,
	}

	if strings.IndexByte(hostPort, ':') < 0 {
		hostPort += ":0"
	}

	host, port, err := net.SplitHostPort(hostPort)
	if err != nil {
		return nil, err
	}

	p, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return nil, err
	}
	e.Port = int(p)

	addrs, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}

	for i := range addrs {
		addr := addrs[i].To4()
		if addr == nil {
			// IPv6 - 16 bytes
			if e.IPv6 == nil {
				e.IPv6 = addrs[i].To16()
			}
		} else {
			// IPv4 - 4 bytes
			if e.IPv4 == nil {
				e.IPv4 = addr
			}
		}
		if e.IPv4 != nil && e.IPv6 != nil {
			// Both IPv4 & IPv6 have been set, done...
			break
		}
	}

	// default to 0 filled 4 byte array for IPv4 if IPv6 only host was found
	if e.IPv4 == nil {
		e.IPv4 = make([]byte, 4)
	}

	return e, nil
}

// NewEndpointOrNil tries to create a new endpoint and returns it. On error
// nil will be returned.
func NewEndpointOrNil(serviceName string, hostPort string) *Endpoint {
	if endpoint, err := NewEndpoint(serviceName, hostPort); err == nil {
		return endpoint
	}
	return nil
}
