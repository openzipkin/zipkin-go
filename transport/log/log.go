/*
Package log implements a transport to send spans in V2 JSON format to the Go
// standard Logger.
*/
package log

import (
	"encoding/json"
	"log"
	"os"
	"time"

	zipkin "github.com/openzipkin/zipkin-go"
)

// Transport will send spans to the default Go Logger.
type Transport struct {
	logger *log.Logger
}

// NewTransporter returns a new log transporter.
func NewTransporter(l *log.Logger) *Transport {
	if l == nil {
		// use standard type of log setup
		l = log.New(os.Stderr, "", log.LstdFlags)
	}
	return &Transport{
		logger: l,
	}
}

// Send outputs a span to the Go logger.
func (t *Transport) Send(s zipkin.SpanModel) {
	if b, err := json.MarshalIndent(s, "", "  "); err == nil {
		t.logger.Printf("%s:\n%s\n\n", time.Now(), string(b))
	}
}

// Close closes the transporter
func (t *Transport) Close() error { return nil }
