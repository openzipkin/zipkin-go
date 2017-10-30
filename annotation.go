package zipkin

import (
	"encoding/json"
	"time"
)

// annotation associates an event that explains latency with a timestamp.
type annotation struct {
	timestamp time.Time
	value     string
}

// MarshalJSON implements custom JSON encoding
func (a *annotation) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Timestamp int64  `json:"timestamp"`
		Value     string `json:"value"`
	}{
		Timestamp: a.timestamp.UnixNano() / 1e6,
		Value:     a.value,
	})
}
