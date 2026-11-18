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
Package amqp implements a RabbitMq reporter to send spans to a Rabbit server/cluster.
*/
package amqp

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/reporter"
)

// defaultRmqRoutingKey/Exchange/Kind sets the standard RabbitMQ queue our Reporter will publish on.
const (
	defaultRmqRoutingKey = "zipkin"
	defaultRmqExchange   = "zipkin"
	defaultExchangeKind  = "direct"
)

// rmqReporter implements Reporter by publishing spans to a RabbitMQ exchange
type rmqReporter struct {
	e        chan error
	channel  *amqp.Channel
	conn     *amqp.Connection
	exchange string
	queue    string
	logger   *log.Logger
}

// ReporterOption sets a parameter for the rmqReporter
type ReporterOption func(c *rmqReporter)

// Logger sets the logger used to report errors in the collection
// process.
func Logger(logger *log.Logger) ReporterOption {
	return func(c *rmqReporter) {
		c.logger = logger
	}
}

// Exchange sets the Exchange used to send messages (
// see https://github.com/openzipkin/zipkin/tree/master/zipkin-collector/rabbitmq
// if want to change default routing key or exchange
func Exchange(exchange string) ReporterOption {
	return func(c *rmqReporter) {
		c.exchange = exchange
	}
}

// Queue sets the Queue used to send messages
func Queue(queue string) ReporterOption {
	return func(c *rmqReporter) {
		c.queue = queue
	}
}

// Channel sets the Channel used to send messages
func Channel(ch *amqp.Channel) ReporterOption {
	return func(c *rmqReporter) {
		c.channel = ch
	}
}

// Connection sets the Connection used to send messages
func Connection(conn *amqp.Connection) ReporterOption {
	return func(c *rmqReporter) {
		c.conn = conn
	}
}

// NewReporter returns a new RabbitMq-backed Reporter. address should be as described here: https://www.rabbitmq.com/uri-spec.html
func NewReporter(address string, options ...ReporterOption) (reporter.Reporter, error) {
	r := &rmqReporter{
		logger:   log.New(os.Stderr, "", log.LstdFlags),
		queue:    defaultRmqRoutingKey,
		exchange: defaultRmqExchange,
		e:        make(chan error),
	}

	for _, option := range options {
		option(r)
	}

	checks := []func() error{
		r.queueVerify,
		r.exchangeVerify,
		r.queueBindVerify,
	}

	var err error

	if r.conn == nil {
		r.conn, err = amqp.Dial(address)
		if err != nil {
			return nil, err
		}
	}

	if r.channel == nil {
		r.channel, err = r.conn.Channel()
		if err != nil {
			return nil, err
		}
	}

	for i := 0; i < len(checks); i++ {
		if err := checks[i](); err != nil {
			return nil, err
		}
	}

	go r.logErrors()

	return r, nil
}

func (r *rmqReporter) logErrors() {
	for err := range r.e {
		r.logger.Print("msg", err.Error())
	}
}

func (r *rmqReporter) Send(s model.SpanModel) {
	// Zipkin expects the message to be wrapped in an array
	ss := []model.SpanModel{s}
	m, err := json.Marshal(ss)
	if err != nil {
		r.e <- fmt.Errorf("failed when marshalling the span: %s", err.Error())
		return
	}

	msg := amqp.Publishing{
		Body: m,
	}

	err = r.channel.Publish(r.exchange, r.queue, false, false, msg)
	if err != nil {
		r.e <- fmt.Errorf("failed when publishing the span: %s", err.Error())
	}
}

func (r *rmqReporter) queueBindVerify() error {
	return r.channel.QueueBind(
		r.queue,
		r.queue,
		r.exchange,
		false,
		nil)
}

func (r *rmqReporter) exchangeVerify() error {
	err := r.channel.ExchangeDeclare(
		r.exchange,
		defaultExchangeKind,
		true,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		return err
	}

	return nil
}

func (r *rmqReporter) queueVerify() error {
	_, err := r.channel.QueueDeclare(
		r.queue,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	return nil
}

func (r *rmqReporter) Close() error {
	err := r.channel.Close()
	if err != nil {
		return err
	}

	err = r.conn.Close()
	if err != nil {
		return err
	}
	return nil
}
