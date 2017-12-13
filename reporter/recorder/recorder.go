/*
Package recorder implements a reporter to record spans in v2 format.
*/
package recorder

import (
	"sync"

	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/reporter"
)

// ReporterRecorder records Zipkin spans.
type ReporterRecorder struct {
	mtx   sync.Mutex
	spans []model.SpanModel
}

// NewReporter returns a new recording reporter.
func NewReporter() reporter.Reporter {
	return &ReporterRecorder{}
}

// Send adds the provided span to the span list held by the recorder.
func (r *ReporterRecorder) Send(span model.SpanModel) {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	r.spans = append(r.spans, span)
}

// Flush returns all recorded spans and clears its internal span storage
func (r *ReporterRecorder) Flush() []model.SpanModel {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	spans := r.spans
	r.spans = nil
	return spans
}

// Close flushes the reporter
func (r *ReporterRecorder) Close() error {
	_ = r.Flush
	return nil
}
