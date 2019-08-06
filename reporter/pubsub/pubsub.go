package pubsub

import (
	"cloud.google.com/go/pubsub"
	"context"
	"encoding/json"
	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/reporter"
	"log"
	"os"
)

const defaultPubSubTopic = "pubsub"

// Reporter implements Reporter by publishing spans to a GCP pubsub.
type Reporter struct {
	logger *log.Logger
	topic  string
	client *pubsub.Client
}

// ReporterOption sets a parameter for the PubSubReporter
type ReporterOption func(c *Reporter)

// Send send span to topic
func (r *Reporter) Send(s model.SpanModel) {
	// Zipkin expects the message to be wrapped in an array
	ss := []model.SpanModel{s}
	m, err := json.Marshal(ss)
	if err != nil {
		r.logger.Printf("failed when marshalling the span: %s\n", err.Error())
		return
	}
	err = r.publish(m)
	if err != nil {
		r.logger.Printf("Error publishing message to pubsub: %s msg: %s", err.Error(), string(m))
	}
}

// Close close span
func (r *Reporter) Close() error {
	return r.client.Close()
}

// Logger sets the logger used to report errors in the collection
// process.
func Logger(logger *log.Logger) ReporterOption {
	return func(c *Reporter) {
		c.logger = logger
	}
}

// Client sets the client used to produce to pubsub.
func Client(client *pubsub.Client) ReporterOption {
	return func(c *Reporter) {
		c.client = client
	}
}

// Topic sets the kafka topic to attach the reporter producer on.
func Topic(t string) ReporterOption {
	return func(c *Reporter) {
		c.topic = t
	}
}

// NewReporter returns a new Kafka-backed Reporter. address should be a slice of
// TCP endpoints of the form "host:port".
func NewReporter(options ...ReporterOption) (reporter.Reporter, error) {
	r := &Reporter{
		logger: log.New(os.Stderr, "", log.LstdFlags),
		topic:  defaultPubSubTopic,
	}

	for _, option := range options {
		option(r)
	}
	if r.client == nil {
		ctx := context.Background()
		proj := os.Getenv("GOOGLE_CLOUD_PROJECT")
		if proj == "" {
			log.Fatal("GOOGLE_CLOUD_PROJECT environment variable must be set. Traces wont be sent to pubsub")
		}
		client, err := pubsub.NewClient(ctx, proj)
		if err != nil {
			log.Fatalf("Could not create pubsub Client: %v", err)
		}
		r.client = client
	}

	return r, nil
}

func (r *Reporter) publish(msg []byte) error {
	ctx := context.Background()
	t := r.client.Topic(r.topic)

	result := t.Publish(ctx, &pubsub.Message{
		// data must be a ByteString
		Data: msg,
	})
	_, err := result.Get(ctx)
	return err
}
