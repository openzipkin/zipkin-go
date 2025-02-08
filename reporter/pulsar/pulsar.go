// Copyright 2022 The OpenZipkin Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*
Package pulsar implements a Pulsar reporter to send spans to a Pulsar server/cluster.
*/
package pulsar

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/apache/pulsar-client-go/pulsar"

	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/reporter"
)

// defaultPulsarTopic sets the standard Pulsar topic our Reporter will publish
// on. The default topic for zipkin-collector-pulsar is "zipkin", see:
// https://github.com/openzipkin/zipkin/tree/master/zipkin-collector/pulsar
const defaultPulsarTopic = "zipkin"

// pulsarReporter implements Reporter by publishing spans to a Pulsar broker.
type pulsarReporter struct {
	e          chan error
	client     pulsar.Client
	producer   pulsar.Producer
	logger     *log.Logger
	topic      string
	serializer reporter.SpanSerializer
}

// ReporterOption sets a parameter for the pulsarReporter
type ReporterOption func(c *pulsarReporter)

// Logger sets the logger used to report errors in the collection
// process.
func Logger(logger *log.Logger) ReporterOption {
	return func(c *pulsarReporter) {
		c.logger = logger
	}
}

// Topic sets the pulsar topic to attach the reporter producer on.
func Topic(t string) ReporterOption {
	return func(c *pulsarReporter) {
		c.topic = t
	}
}

// Serializer sets the serialization function to use for sending span data to
// Zipkin.
func Serializer(serializer reporter.SpanSerializer) ReporterOption {
	return func(c *pulsarReporter) {
		if serializer != nil {
			c.serializer = serializer
		}
	}
}

// Client sets the Pulsar client to use for the reporter.
func Client(p pulsar.Client) ReporterOption {
	return func(c *pulsarReporter) {
		c.client = p
	}
}

// Producer sets the Pulsar producer to use for the reporter.
func Producer(p pulsar.Producer) ReporterOption {
	return func(c *pulsarReporter) {
		c.producer = p
	}
}

func (p *pulsarReporter) logErrors() {
	for err := range p.e {
		p.logger.Print("msg", err.Error())
	}
}

func NewReporter(address string, options ...ReporterOption) (reporter.Reporter, error) {
	p := &pulsarReporter{
		logger:     log.New(os.Stderr, "", log.LstdFlags),
		topic:      defaultPulsarTopic,
		serializer: reporter.JSONSerializer{},
	}

	for _, option := range options {
		option(p)
	}

	var err error
	if p.client == nil {
		p.client, err = pulsar.NewClient(pulsar.ClientOptions{
			URL: address,
		})
		if err != nil {
			return nil, err
		}
	}
	if p.producer == nil {
		p.producer, err = p.client.CreateProducer(pulsar.ProducerOptions{
			Topic: p.topic,
		})
		if err != nil {
			return nil, err
		}
	}

	go p.logErrors()

	return p, nil
}

func (p *pulsarReporter) Send(s model.SpanModel) {
	// Zipkin expects the message to be wrapped in an array
	ss := []*model.SpanModel{&s}
	m, err := p.serializer.Serialize(ss)
	if err != nil {
		p.e <- fmt.Errorf("failed when marshalling the span: %s\n", err.Error())
		return
	}

	message := &pulsar.ProducerMessage{
		Payload: m,
	}
	p.producer.SendAsync(context.Background(), message, func(_ pulsar.MessageID, _ *pulsar.ProducerMessage, err error) {
		if err != nil {
			p.e <- fmt.Errorf("failed to produce msg: %s\n", err.Error())
		}
	})
}

func (p *pulsarReporter) Close() error {
	p.producer.Close()
	p.client.Close()
	return nil
}
