package amqp

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/streadway/amqp"

	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/reporter"
)

const (
	defaultRmqRoutingKey = "zipkin"
	defaultRmqExchange   = "zipkin"
	defaultExchangeKind  = "direct"
)

type rmqReporter struct {
	e        chan error
	channel  *amqp.Channel
	conn     *amqp.Connection
	Exchange string
	Queue    string
	logger   *log.Logger
}

type ReporterOption func(c *rmqReporter)

func Logger(logger *log.Logger) ReporterOption {
	return func(c *rmqReporter) {
		c.logger = logger
	}
}

func Exchange(e string) ReporterOption {
	return func(c *rmqReporter) {
		c.Exchange = e
	}
}

func Queue(t string) ReporterOption {
	return func(c *rmqReporter) {
		c.Queue = t
	}
}

func Channel(ch *amqp.Channel) ReporterOption {
	return func(c *rmqReporter) {
		c.channel = ch
	}
}

func Connection(conn *amqp.Connection) ReporterOption {
	return func(c *rmqReporter) {
		c.conn = conn
	}
}

func NewReporter(address string, options ...ReporterOption) (reporter.Reporter, error) {
	r := &rmqReporter{
		logger:   log.New(os.Stderr, "", log.LstdFlags),
		Queue:    defaultRmqRoutingKey,
		Exchange: defaultRmqExchange,
		// add severity ?
		e: make(chan error),
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
		r.e <- fmt.Errorf("failed when marshalling the span: %s\n", err.Error())
		return
	}

	msg := amqp.Publishing{
		Body: m,
	}

	err = r.channel.Publish(defaultRmqExchange, defaultRmqRoutingKey, false, false, msg)
	if err != nil {
		r.e <- fmt.Errorf("failed when publishing the span: %s\n", err.Error())
	}
}

func (r *rmqReporter) queueBindVerify() error {
	return r.channel.QueueBind(
		defaultRmqRoutingKey,
		defaultRmqRoutingKey,
		defaultRmqExchange,
		false,
		nil)
}

func (r *rmqReporter) exchangeVerify() error {
	err := r.channel.ExchangeDeclare(
		defaultRmqExchange,
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
		defaultRmqExchange,
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
