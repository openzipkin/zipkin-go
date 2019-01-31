package reporter

import (
	"encoding/json"

	"github.com/openzipkin/zipkin-go/model"
)

// SpanSerializer describes the methods needed for allowing to set Span encoding
// type for the various Zipkin transports.
type SpanSerializer interface {
	Serialize([]*model.SpanModel) ([]byte, error)
	ContentType() string
}

// JSONSerializer implements the default JSON encoding SpanSerializer.
type JSONSerializer struct{}

// Serialize takes an array of Zipkin SpanModel objects and returns a JSON
// encoding of it.
func (JSONSerializer) Serialize(spans []*model.SpanModel) ([]byte, error) {
	return json.Marshal(spans)
}

// ContentType returns the ContentType needed for this encoding.
func (JSONSerializer) ContentType() string {
	return "application/json"
}
