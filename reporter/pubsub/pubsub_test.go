package pubsub

import (
	"context"
	"fmt"
	"github.com/openzipkin/zipkin-go/model"
	"os"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/pubsub"
)

var topicID string

var once sync.Once // guards cleanup related operations in setup.

func setup(t *testing.T) *pubsub.Client {
	ctx := context.Background()
	proj := os.Getenv("GOOGLE_CLOUD_PROJECT")
	topicID = "test-topic"

	client, err := pubsub.NewClient(ctx, proj)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.CreateTopic(ctx, topicID)
	if err != nil {
		t.Fatalf("failed to create topic: %v", err)
	}
	fmt.Printf("Topic created: %s\n", t.Name())
	return client
}

func TestPublish(t *testing.T) {
	c := setup(t)
	reporter, err := NewReporter(Client(c), Topic(topicID))
	if err != nil {
		t.Errorf("failed creating reporter: %v", err)
	}
	span := makeNewSpan("avg", 123, 456, 0, true)
	reporter.Send(*span)

	// Cleanup resources from the previous failed tests.
	once.Do(func() {
		ctx := context.Background()
		topic := c.Topic(topicID)
		ok, err := topic.Exists(ctx)
		if err != nil {
			t.Fatalf("failed to check if topic exists: %v", err)
		}
		if !ok {
			return
		}
		if err := topic.Delete(ctx); err != nil {
			t.Fatalf("failed to cleanup the topic (%q): %v", topicID, err)
		}
	})
}

func makeNewSpan(methodName string, traceID, spanID, parentSpanID uint64, debug bool) *model.SpanModel {
	timestamp := time.Now()
	var parentID = new(model.ID)
	if parentSpanID != 0 {
		*parentID = model.ID(parentSpanID)
	}

	return &model.SpanModel{
		SpanContext: model.SpanContext{
			TraceID:  model.TraceID{Low: traceID},
			ID:       model.ID(spanID),
			ParentID: parentID,
			Debug:    debug,
		},
		Name:      methodName,
		Timestamp: timestamp,
	}
}

