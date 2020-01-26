package gcppubsub

import (
	"cloud.google.com/go/pubsub"
	"context"
	"encoding/json"
	"errors"
	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/reporter"
	"log"
	"os"
)

const DefaultPubSubTopic = "defaultTopic"

var resultMsg = make(chan reporterResult)

// Reporter implements Reporter by publishing spans to a GCP gcppubsub.
type Reporter struct {
	logger *log.Logger
	topic  *pubsub.Topic
	client *pubsub.Client
}

type reporterResult struct {
	ctx    context.Context
	result pubsub.PublishResult
}

// ReporterOption sets a parameter for the reporter
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
	r.publish(m)
}

// Close releases any resources held by the client (pubsub client publisher and subscriber connections).
func (r *Reporter) Close() error {
	close(resultMsg)
	return r.client.Close()
}

// Logger sets the logger used to report errors in the collection
// process.
func Logger(logger *log.Logger) ReporterOption {
	return func(c *Reporter) {
		c.logger = logger
	}
}

// Client sets the client used to produce to gcppubsub.
func Client(client *pubsub.Client) ReporterOption {
	return func(c *Reporter) {
		c.client = client
	}
}

// Topic sets the gcppubsub topic to attach the reporter producer on.
func Topic(t *pubsub.Topic) ReporterOption {
	return func(c *Reporter) {
		c.topic = t
	}
}

// NewReporter returns a new gcppubsub-backed Reporter. address should be a slice of
// TCP endpoints of the form "host:port".
func NewReporter(options ...ReporterOption) (reporter.Reporter, error) {
	r := &Reporter{
		logger: log.New(os.Stderr, "", log.LstdFlags),
	}

	for _, option := range options {
		option(r)
	}

	if r.client == nil {
		err := errors.New("cannot create pubsub reporter without valid client")
		return nil, err
	}
	if r.topic == nil {
		t := r.client.Topic(DefaultPubSubTopic)
		r.topic = t
	}
	go r.checkResult()
	return r, nil
}

func (r *Reporter) publish(msg []byte) {
	ctx := context.Background()

	result := r.topic.Publish(ctx, &pubsub.Message{
		Data: msg,
	})
	resultMsg <- reporterResult{ctx, *result}
}

func (r *Reporter) checkResult() {
	for n := range resultMsg {
		_, err := n.result.Get(n.ctx)
		if err != nil {
			r.logger.Printf("Error sending message: %s\n", err.Error())
		}
	}
}
