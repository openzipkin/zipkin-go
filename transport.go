package zipkin

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

// Transporter interface
type Transporter interface {
	Send(SpanModel)
}

// NoopTransport will silently drop spans.
type NoopTransport struct{}

// Send drops a span
func (t *NoopTransport) Send(_ SpanModel) {}

// loggerTransport will send spans to the default Go Logger.
type loggerTransport struct {
	logger *log.Logger
}

// NewLoggerTransporter returns a new logger transporter.
func NewLoggerTransporter(l *log.Logger) *loggerTransport {
	if l == nil {
		// use standard type of logger setup
		l = log.New(os.Stderr, "", log.LstdFlags)
	}
	return &loggerTransport{
		logger: l,
	}
}

// Send outputs a span to the Go logger.
func (t *loggerTransport) Send(s SpanModel) {
	if b, err := json.MarshalIndent(s, "", "  "); err == nil {
		t.logger.Printf("%s:\n%s\n\n", time.Now(), string(b))
	}
}
