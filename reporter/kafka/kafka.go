// Copyright 2019 The OpenZipkin Authors
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
Package kafka implements a Kafka reporter to send spans to a Kafka server/cluster.
*/
package kafka

import (
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"

	"github.com/Shopify/sarama"
	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/reporter"
)

// defaultKafkaTopic sets the standard Kafka topic our Reporter will publish
// on. The default topic for zipkin-collector/kafka is "zipkin", see:
// https://github.com/openzipkin/zipkin/tree/master/zipkin-collector/kafka

// defaults
const (
	defaultBatchInterval = time.Second * 1 // BatchInterval in seconds
	defaultBatchSize     = 100
	defaultMaxBacklog    = 1000
	defaultKafkaTopic    = "zipkin"
)

// kafkaReporter implements Reporter by publishing spans to a Kafka
// broker.
type kafkaReporter struct {
	producer      sarama.AsyncProducer
	logger        *log.Logger
	topic         string
	serializer    reporter.SpanSerializer
	batchInterval time.Duration
	batchSize     int
	maxBacklog    int
	batchMtx      *sync.Mutex
	batch         []*model.SpanModel
	spanC         chan *model.SpanModel
	sendC         chan struct{}
	quit          chan struct{}
	shutdown      chan error
}

// ReporterOption sets a parameter for the kafkaReporter
type ReporterOption func(c *kafkaReporter)

// Logger sets the logger used to report errors in the collection
// process.
func Logger(logger *log.Logger) ReporterOption {
	return func(c *kafkaReporter) {
		c.logger = logger
	}
}

// Producer sets the producer used to produce to Kafka.
func Producer(p sarama.AsyncProducer) ReporterOption {
	return func(c *kafkaReporter) {
		c.producer = p
	}
}

// BatchSize sets the maximum batch size, after which a collect will be
// triggered. The default batch size is 100 traces.
func BatchSize(n int) ReporterOption {
	return func(r *kafkaReporter) { r.batchSize = n }
}

// BatchInterval sets the maximum duration we will buffer traces before
// emitting them to the collector. The default batch interval is 1 second.
func BatchInterval(d time.Duration) ReporterOption {
	return func(r *kafkaReporter) { r.batchInterval = d }
}

// MaxBacklog sets the maximum backlog size. When batch size reaches this
// threshold, spans from the beginning of the batch will be disposed.
func MaxBacklog(n int) ReporterOption {
	return func(r *kafkaReporter) { r.maxBacklog = n }
}

// Topic sets the kafka topic to attach the reporter producer on.
func Topic(t string) ReporterOption {
	return func(c *kafkaReporter) {
		c.topic = t
	}
}

// Serializer sets the serialization function to use for sending span data to
// Zipkin.
func Serializer(serializer reporter.SpanSerializer) ReporterOption {
	return func(c *kafkaReporter) {
		if serializer != nil {
			c.serializer = serializer
		}
	}
}

// NewReporter returns a new Kafka-backed Reporter. address should be a slice of
// TCP endpoints of the form "host:port".
func NewReporter(address []string, options ...ReporterOption) (reporter.Reporter, error) {
	r := &kafkaReporter{
		logger:        log.New(os.Stderr, "", log.LstdFlags),
		topic:         defaultKafkaTopic,
		serializer:    reporter.JSONSerializer{},
		batchInterval: defaultBatchInterval,
		batchSize:     defaultBatchSize,
		maxBacklog:    defaultMaxBacklog,
		batch:         []*model.SpanModel{},
		spanC:         make(chan *model.SpanModel),
		sendC:         make(chan struct{}, 1),
		quit:          make(chan struct{}, 1),
		shutdown:      make(chan error, 1),
		batchMtx:      &sync.Mutex{},
	}

	for _, option := range options {
		option(r)
	}
	if r.producer == nil {
		p, err := sarama.NewAsyncProducer(address, nil)
		if err != nil {
			return nil, err
		}
		r.producer = p
	}

	go r.loop()
	go r.sendLoop()
	go r.logErrors()

	return r, nil
}

func (r *kafkaReporter) logErrors() {
	for pe := range r.producer.Errors() {
		r.logger.Print("msg", pe.Msg, "err", pe.Err, "result", "failed to produce msg")
	}
}

func (r *kafkaReporter) Send(s model.SpanModel) {
	r.spanC <- &s
}

func (r *kafkaReporter) Close() error {
	close(r.quit)
	<-r.shutdown
	return r.producer.Close()
}

func (r *kafkaReporter) loop() {
	var (
		nextSend   = time.Now().Add(r.batchInterval)
		ticker     = time.NewTicker(r.batchInterval / 10)
		tickerChan = ticker.C
	)
	defer ticker.Stop()

	for {
		select {
		case span := <-r.spanC:
			currentBatchSize := r.append(span)
			if currentBatchSize >= r.batchSize {
				nextSend = time.Now().Add(r.batchInterval)
				r.enqueueSend()
			}
		case <-tickerChan:
			if time.Now().After(nextSend) {
				nextSend = time.Now().Add(r.batchInterval)
				r.enqueueSend()
			}
		case <-r.quit:
			close(r.sendC)
			return
		}
	}
}

func (r *kafkaReporter) sendLoop() {
	for range r.sendC {
		_ = r.sendBatch()
	}
	r.shutdown <- r.sendBatch()
}

func (r *kafkaReporter) enqueueSend() {
	select {
	case r.sendC <- struct{}{}:
	default:
		// Do nothing if there's a pending send request already
	}
}

func (r *kafkaReporter) sendBatch() error {
	// Zipkin expects the message to be wrapped in an array

	// Select all current spans in the batch to be sent
	r.batchMtx.Lock()
	sendBatch := r.batch[:]
	r.batchMtx.Unlock()

	if len(sendBatch) == 0 {
		return nil
	}
	m, err := json.Marshal(sendBatch)
	if err != nil {
		r.logger.Printf("failed when marshalling the span: %s\n", err.Error())
		return err
	}

	r.producer.Input() <- &sarama.ProducerMessage{
		Topic: r.topic,
		Key:   nil,
		Value: sarama.ByteEncoder(m),
	}

	// Remove sent spans from the batch even if they were not saved
	r.batchMtx.Lock()
	r.batch = r.batch[len(sendBatch):]
	r.batchMtx.Unlock()
	return nil
}

func (r *kafkaReporter) append(span *model.SpanModel) (newBatchSize int) {
	r.batchMtx.Lock()

	r.batch = append(r.batch, span)
	if len(r.batch) > r.maxBacklog {
		dispose := len(r.batch) - r.maxBacklog
		r.logger.Printf("backlog too long, disposing %d spans", dispose)
		r.batch = r.batch[dispose:]
	}
	newBatchSize = len(r.batch)

	r.batchMtx.Unlock()
	return
}
