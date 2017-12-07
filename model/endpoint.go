package model

import "net"

// Endpoint holds the network context of a node in the service graph.
type Endpoint struct {
	ServiceName string `json:"serviceName,omitempty"`
	IPv4        net.IP `json:"ipv4,omitempty"`
	IPv6        net.IP `json:"ipv6,omitempty"`
	Port        uint16 `json:"port,omitempty"`
}
