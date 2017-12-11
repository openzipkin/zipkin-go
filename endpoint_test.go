package zipkin_test

import (
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/openzipkin/zipkin-go"
)

const (
	serviceName           = "service_name"
	onlyHost              = "localhost"
	defaultPort           = 0
	port                  = 8081
	invalidNegativePort   = "localhost:-8081"
	invalidOutOfRangePort = "localhost:65536"
	unreachableHostPort   = "nosuchhost:8081"
)

var (
	ipv4HostPort    = "localhost:" + fmt.Sprintf("%d", port)
	ipv6HostPort    = "[2001:db8::68]:" + fmt.Sprintf("%d", port)
	ipv4ForHostPort = net.IPv4(127, 0, 0, 1)
	ipv6ForHostPort = net.ParseIP("2001:db8::68")
)

func TestNewEndpointFailsDueToOutOfRangePort(t *testing.T) {
	_, err := zipkin.NewEndpoint(serviceName, invalidOutOfRangePort)

	if err == nil {
		t.Fatalf("expected error")
	}

	if !strings.Contains(err.Error(), "value out of range") {
		t.Fatalf("expected out of range error, got: %s", err.Error())
	}
}

func TestNewEndpointFailsDueToNegativePort(t *testing.T) {
	_, err := zipkin.NewEndpoint(serviceName, invalidNegativePort)

	if err == nil {
		t.Fatalf("expected error")
	}

	if !strings.Contains(err.Error(), "invalid syntax") {
		t.Fatalf("expected invalid syntax error, got: %s", err.Error())
	}
}

func TestNewEndpointFailsDueToLookupIP(t *testing.T) {
	_, err := zipkin.NewEndpoint(serviceName, unreachableHostPort)

	if err == nil {
		t.Fatalf("expected error")
	}

	if !strings.Contains(err.Error(), "no such host") {
		t.Fatalf("expected no such host error, got: %s", err.Error())
	}
}

func TestNewEndpointDefaultsPortToZeroWhenMissing(t *testing.T) {
	endpoint, err := zipkin.NewEndpoint(serviceName, onlyHost)

	if err != nil {
		t.Fatalf("not error expected, got %s", err.Error())
	}

	if endpoint.Port != defaultPort {
		t.Fatalf("expected port %d, got %d", defaultPort, endpoint.Port)
	}
}

func TestNewEndpointIpv4Success(t *testing.T) {
	endpoint, err := zipkin.NewEndpoint(serviceName, ipv4HostPort)

	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	if serviceName != endpoint.ServiceName {
		t.Fatalf("expected service name %s, got %s", serviceName, endpoint.ServiceName)
	}

	if !ipv4ForHostPort.Equal(endpoint.IPv4) {
		t.Fatalf("expected IPv4 %s, got %s", ipv4ForHostPort.String(), endpoint.IPv4.String())
	}

	if port != endpoint.Port {
		t.Fatalf("expected port %d, got %d", port, endpoint.Port)
	}
}

func TestNewEndpointIpv6Success(t *testing.T) {
	endpoint, err := zipkin.NewEndpoint(serviceName, ipv6HostPort)

	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	if serviceName != endpoint.ServiceName {
		t.Fatalf("expected service name %s, got %s", serviceName, endpoint.ServiceName)
	}

	if !ipv6ForHostPort.Equal(endpoint.IPv6) {
		t.Fatalf("expected IPv6 %s, got %s", ipv6ForHostPort.String(), endpoint.IPv6.String())
	}

	if port != endpoint.Port {
		t.Fatalf("expected port %d, got %d", port, endpoint.Port)
	}
}
