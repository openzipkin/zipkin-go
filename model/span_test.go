package model

import (
	"encoding/json"
	"errors"
	"net"
	"reflect"
	"testing"
	"time"
)

func TestJSON(t *testing.T) {
	var (
		span1    SpanModel
		span2    SpanModel
		parentID = ID(1003)
		sampled  = true
		tags     = make(map[string]string)
	)
	tags["myKey"] = "myValue"
	tags["another"] = "tag"

	span1 = SpanModel{
		SpanContext: SpanContext{
			TraceID: TraceID{
				High: 1001,
				Low:  1002,
			},
			ID:       ID(1004),
			ParentID: &parentID,
			Debug:    true,
			Sampled:  &sampled,
			Err:      errors.New("dummy"),
		},
		Name:      "myMethod",
		Kind:      Server,
		Timestamp: time.Now().Add(-100 * time.Millisecond),
		Duration:  50 * time.Millisecond,
		Shared:    true,
		LocalEndpoint: &Endpoint{
			ServiceName: "myService",
			IPv4:        net.IPv4(127, 0, 0, 1),
			IPv6:        net.IPv6loopback,
		},
		RemoteEndpoint: &Endpoint{},
		Annotations: []Annotation{
			{time.Now().Add(-90 * time.Millisecond), "myAnnotation"},
		},
		Tags: tags,
	}

	b, err := json.Marshal(&span1)
	if err != nil {
		t.Errorf("expected successful serialization to JSON, got error: %+v", err)
	}

	err = json.Unmarshal(b, &span2)
	if err != nil {
		t.Errorf("expected successful deserialization from JSON, got error: %+v", err)
	}

	/* remove items from span1 which should not have exported */
	span1.Sampled = nil
	span1.Err = nil

	// reset monotonic clock readings in time.Time values for comparison
	span1.Timestamp = span1.Timestamp.Round(0)
	for idx := range span1.Annotations {
		span1.Annotations[idx].Timestamp = span1.Annotations[idx].Timestamp.Round(0)
	}

	if !reflect.DeepEqual(span1, span2) {
		t.Errorf("want SpanModel: %+v, have: %+v", span1, span2)
	}
}
