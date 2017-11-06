package zipkin

import (
	"encoding/json"
	"time"
)

// Annotation associates an event that explains latency with a timestamp.
type Annotation struct {
	Timestamp time.Time
	Value     string
}

// MarshalJSON implements custom JSON encoding
func (a *Annotation) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Timestamp int64  `json:"timestamp"`
		Value     string `json:"value"`
	}{
		Timestamp: a.Timestamp.UnixNano() / 1e3,
		Value:     a.Value,
	})
}
