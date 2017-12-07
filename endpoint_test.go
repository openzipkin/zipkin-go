package zipkin_test

import (
	"fmt"
	"net"
	"strings"
	"testing"

	zipkin "github.com/openzipkin/zipkin-go"
)

const (
	serviceName           = "service_name"
	port                  = 8081
	invalidNegativePort   = "localhost:-8081"
	invalidOutOfRangePort = "localhost:65536"
	unreachableHostPort   = "nosuchhost:8081"
)

var (
	hostPort        = "localhost:" + fmt.Sprintf("%d", port)
	ip4ForLocalhost = net.IPv4(127, 0, 0, 1)
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

func TestNewEndpointSuccess(t *testing.T) {
	endpoint, err := zipkin.NewEndpoint(serviceName, hostPort)

	if err != nil {
		t.Fatalf("not error expected, got %s", err.Error())
	}

	if serviceName != endpoint.ServiceName {
		t.Fatalf("wrong service name, expected %s and got %s", serviceName, endpoint.ServiceName)
	}

	if !ip4ForLocalhost.Equal(endpoint.IPv4) {
		t.Fatalf("wrong ip4, expected %s and got %s", ip4ForLocalhost.String(), endpoint.IPv4.String())
	}

	if port != endpoint.Port {
		t.Fatalf("wrong port, expected %s and got %s", ip4ForLocalhost.String(), endpoint.IPv4.String())
	}
}
