package gcppubsub

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/reporter"
)

var topicID string

func setup(t *testing.T, topicID string) *pubsub.Client {
	ctx := context.Background()
	proj := os.Getenv("GOOGLE_CLOUD_PROJECT")
	fmt.Printf("GCP Project: %s\n", proj)

	client, err := pubsub.NewClient(ctx, proj)
	if err != nil {
		t.Fatalf("failed to create client: %s\n", topicID)
	}
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
			reporter, err := newStubReporter(Client(c), Topic(top))
			if err != nil {
				t.Fatalf("failed creating reporter: %v", err)
			}
			span := makeNewSpan("avg1", 124, 457, 0, true)
			reporter.Send(*span)
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

func TestClose(t *testing.T) {
	tcs := map[string]struct {
		willFail bool
	}{
		"will success": {true},
		"will fail":    {false},
	}

	for n, tc := range tcs {
		t.Run(n, func(t *testing.T) {
			reporter, err := newStubReporter()
			if err != nil {
				t.Fatalf("failed creating reporter: %v", err)
			}
			if err := reporter.Close(); err != nil && tc.willFail {
				t.Fatalf("failed to close reporter: %v", err)
			}
		})
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

type stubReporter struct {
	logger *log.Logger
	client *stubClient
	topic  *pubsub.Topic
}

type stubClient struct{}

func (c *stubClient) Topic(name string) *pubsub.Topic {
	return &pubsub.Topic{}
}

func (r *stubReporter) Send(span model.SpanModel) {}

func (r *stubReporter) Close() error {
	return nil
}

func newStubReporter(...ReporterOption) (reporter.Reporter, error) {
	r := &stubReporter{
		logger: log.New(os.Stderr, "", log.LstdFlags),
		client: &stubClient{},
	}

	if r.client == nil {
		return nil, errors.New("cannot create pubsub reporter without valid client")
	}
	if r.topic == nil {
		r.topic = r.client.Topic(defaultPubSubTopic)
	}
	return r, nil
}
