package gcppubsub

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/openzipkin/zipkin-go/model"
)

var topicID string

var once sync.Once // guards cleanup related operations in setup.

func setup(t *testing.T, topicID string) *pubsub.Client {
	ctx := context.Background()
	proj := os.Getenv("GOOGLE_CLOUD_PROJECT")
	fmt.Printf("GCP Project: %s\n", proj)

	client, err := pubsub.NewClient(ctx, proj)
	if err != nil {
		t.Fatalf("failed to create client: %s\n", topicID)
	}

	_, err = client.CreateTopic(ctx, topicID)
	if err != nil {
		t.Fatalf("failed to create topic: %v", err)
	}
	fmt.Printf("Topic created: %s\n", topicID)
	return client
}

func TestPublish(t *testing.T) {
	tcs := map[string]struct {
		topicID string
	}{
		"with test-topic": {
			topicID: "test-topic",
		},
		"with default topic": {
			topicID: defaultPubSubTopic,
		},
	}

	for n, tc := range tcs {
		t.Run(n, func(t *testing.T) {
			c := setup(t, tc.topicID)
			top := c.Topic(topicID)
			reporter, err := NewReporter(Client(c), Topic(top))
			if err != nil {
				t.Fatalf("failed creating reporter: %v", err)
			}
			span := makeNewSpan("avg1", 124, 457, 0, true)
			reporter.Send(*span)

			// Cleanup resources from the previous failed tests.
			once.Do(func() {
				ctx := context.Background()
				topic := c.Topic(topicID)
				_, err := topic.Exists(ctx)
				if err != nil {
					t.Fatalf("failed to check if topic exists: %v", err)
				}

				if err := topic.Delete(ctx); err != nil {
					t.Fatalf("failed to cleanup the topic (%q): %v", topicID, err)
				}
			})
		})
	}
}

func TestErrorNotProjEnv(t *testing.T) {
	reporter, err := NewReporter()
	if reporter != nil {
		t.Fatal("Reporter should be null when initiated without client")
	}
	if err == nil {
		t.Fatal("NewReporter should return an error when initiated without client")
	}
	if err.Error() != "cannot create pubsub reporter without valid client" {
		t.Fatal("NewReporter should return cannot create pubsub reporter without valid client error")
	}
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

func TestLogger(t *testing.T) {
	tcs := map[string]struct {
		logger *log.Logger
	}{
		"with no logger": {
			logger: nil,
		},
		"with default logger": {
			logger: log.New(nil, "", 0),
		},
	}

	for n, tc := range tcs {
		t.Run(n, func(t *testing.T) {
			c := setup(t, defaultPubSubTopic)
			_, err := NewReporter(Client(c), Logger(tc.logger))
			if err != nil {
				t.Fatalf("failed creating reporter with logger: %v", err)
			}
		})
	}
}
