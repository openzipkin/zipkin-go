/*
Package log implements a reporter to send spans in V2 JSON format to the Go
standard Logger.
*/
package log

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/openzipkin/zipkin-go/model"
)

// LogReporter will send spans to the default Go Logger.
type LogReporter struct {
	logger *log.Logger
}

// NewReporter returns a new log reporter.
func NewReporter(l *log.Logger) *LogReporter {
	if l == nil {
		// use standard type of log setup
		l = log.New(os.Stderr, "", log.LstdFlags)
	}
	return &LogReporter{
		logger: l,
	}
}

// Send outputs a span to the Go logger.
func (r *LogReporter) Send(s model.SpanModel) {
	if b, err := json.MarshalIndent(s, "", "  "); err == nil {
		r.logger.Printf("%s:\n%s\n\n", time.Now(), string(b))
	}
}

// Close closes the reporter
func (*LogReporter) Close() error { return nil }
