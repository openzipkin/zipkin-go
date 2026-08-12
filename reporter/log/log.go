// Copyright 2022 The OpenZipkin Authors
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

/*
Package log implements a reporter to send spans in V2 JSON format to the Go
standard Logger.
*/
package log

import (
	"encoding/json"
	"log"
	"time"

	"github.com/openzipkin/zipkin-go/logging"
	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/reporter"
)

// logReporter will send spans to the default Go Logger.
type logReporter struct {
	logger *logging.Logger
}

// ReporterOption sets a parameter for the logReporter.
type ReporterOption func(r *logReporter)

// LogLevel sets the minimum severity level this reporter will log.
func LogLevel(level logging.Level) ReporterOption {
	return func(r *logReporter) { r.logger.SetLevel(level) }
}

// NewReporter returns a new log reporter. If l is nil, a default logger
// writing to os.Stderr with standard flags is used.
func NewReporter(l *log.Logger, options ...ReporterOption) reporter.Reporter {
	r := &logReporter{
		logger: logging.NewLogger(l),
	}
	for _, option := range options {
		option(r)
	}
	return r
}

// Send outputs a span to the Go logger.
func (r *logReporter) Send(s model.SpanModel) {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		r.logger.Errorf("failed when marshalling the span: %v", err)
		return
	}
	r.logger.Infof("%s:\n%s\n\n", time.Now(), string(b))
}

// Close closes the reporter
func (*logReporter) Close() error { return nil }
