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

package kafka_test

import (
	"encoding/json"
	"errors"
	"log"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/openzipkin/zipkin-go/model"
	zipkin_proto3 "github.com/openzipkin/zipkin-go/proto/zipkin_proto3"
	"github.com/openzipkin/zipkin-go/reporter"
	"github.com/openzipkin/zipkin-go/reporter/kafka"
)

type stubProducer struct {
	in        chan *sarama.ProducerMessage
	err       chan *sarama.ProducerError
	kafkaDown bool
	closed    bool
}

func (p *stubProducer) AsyncClose() {}
func (p *stubProducer) Close() error {
	if p.kafkaDown {
		return errors.New("kafka is down")
	}
	p.closed = true
	return nil
}
func (p *stubProducer) Input() chan<- *sarama.ProducerMessage     { return p.in }
func (p *stubProducer) Successes() <-chan *sarama.ProducerMessage { return nil }
func (p *stubProducer) Errors() <-chan *sarama.ProducerError      { return p.err }

func newStubProducer(kafkaDown bool) *stubProducer {
	return &stubProducer{
		make(chan *sarama.ProducerMessage),
		make(chan *sarama.ProducerError),
		kafkaDown,
		false,
	}
}

var spans = []*model.SpanModel{
	makeNewSpan("avg", 123, 456, 0, true),
	makeNewSpan("sum", 123, 789, 456, true),
	makeNewSpan("div", 123, 101112, 456, true),
}

func jsonDeserializer(body []byte) ([]*model.SpanModel, error) {
	spans := []*model.SpanModel{}
	err := json.Unmarshal(body, &spans)
	return spans, err
}

func protoDeserializer(body []byte) ([]*model.SpanModel, error) {
	spans, err := zipkin_proto3.ParseSpans(body, true)
	return spans, err
}

func TestKafkaProduce(t *testing.T) {
	p := newStubProducer(false)
	c, err := kafka.NewReporter(
		[]string{"192.0.2.10:9092"},
		kafka.Producer(p),
	)
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range spans {
		m := sendSpan(t, c, p, *want)
		testMetadata(t, m)
		have := deserializeSpan(t, m.Value, jsonDeserializer)
		testEqual(t, want, have)
	}
}

func TestKafkaProduceProto(t *testing.T) {
	p := newStubProducer(false)
	c, err := kafka.NewReporter(
		[]string{"192.0.2.10:9092"},
		kafka.Producer(p),
		kafka.Serializer(zipkin_proto3.SpanSerializer{}),
	)
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range spans {
		m := sendSpan(t, c, p, *want)
		testMetadata(t, m)
		have := deserializeSpan(t, m.Value, protoDeserializer)
		testEqual(t, want, have)
	}
}

func TestKafkaClose(t *testing.T) {
	p := newStubProducer(false)
	r, err := kafka.NewReporter(
		[]string{"192.0.2.10:9092"}, kafka.Producer(p),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err = r.Close(); err != nil {
		t.Fatal(err)
	}
	if !p.closed {
		t.Fatal("producer not closed")
	}
}

func TestKafkaCloseError(t *testing.T) {
	p := newStubProducer(true)
	c, err := kafka.NewReporter(
		[]string{"192.0.2.10:9092"}, kafka.Producer(p),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err = c.Close(); err == nil {
		t.Error("no error on close")
	}
}

type chanWriter struct {
	errs chan []interface{}
}

func (cw *chanWriter) Write(p []byte) (n int, err error) {
	cw.errs <- []interface{}{p}

	return 1, nil
}

func TestKafkaErrors(t *testing.T) {
	p := newStubProducer(true)
	errs := make(chan []interface{}, len(spans))

	c, err := kafka.NewReporter(
		[]string{"192.0.2.10:9092"},
		kafka.Producer(p),
		kafka.Logger(log.New(&chanWriter{errs}, "", log.LstdFlags)),
	)
	if err != nil {
		t.Fatal(err)
	}

	var have []model.SpanModel
	for _, want := range spans {
		message := sendSpan(t, c, p, *want)
		messageBody, err := message.Value.Encode()
		if err != nil {
			t.Errorf("unexpected error: %s", err.Error())
		}

		json.Unmarshal(messageBody, &have)
		testEqual(t, want, &have[0])
	}

	for i := 0; i < len(spans); i++ {
		select {
		case <-errs:
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("errors not logged. have %d, wanted %d", i, len(spans))
		}
	}
}

func sendSpan(t *testing.T, r reporter.Reporter, p *stubProducer, s model.SpanModel) *sarama.ProducerMessage {
	var m *sarama.ProducerMessage
	received := make(chan bool, 1)
	go func() {
		select {
		case m = <-p.in:
			received <- true
			if p.kafkaDown {
				p.err <- &sarama.ProducerError{
					Msg: m,
					Err: errors.New("kafka is down"),
				}
			}
		case <-time.After(100 * time.Millisecond):
			received <- false
		}
	}()

	r.Send(s)

	if !<-received {
		t.Fatal("expected message to be received")
	}
	return m
}

func testMetadata(t *testing.T, m *sarama.ProducerMessage) {
	if m.Topic != "zipkin" {
		t.Errorf("unexpected topic. have %q, want %q", m.Topic, "zipkin")
	}
	if m.Key != nil {
		t.Errorf("unexpected key. have %q, want nil", m.Key)
	}
}

func deserializeSpan(t *testing.T, e sarama.Encoder, deserializer func([]byte) ([]*model.SpanModel, error)) *model.SpanModel {
	bytes, err := e.Encode()
	if err != nil {
		t.Errorf("unexpected error in encoding: %v", err)
	}

	s, err := deserializer(bytes)
	if err != nil {
		t.Errorf("unexpected error in decoding: %v", err)
		return nil
	}

	return s[0]
}

func testEqual(t *testing.T, want *model.SpanModel, have *model.SpanModel) {
	if have.TraceID != want.TraceID {
		t.Errorf("incorrect trace_id. have %d, want %d", have.TraceID, want.TraceID)
	}
	if have.ID != want.ID {
		t.Errorf("incorrect id. have %d, want %d", have.ID, want.ID)
	}
	if have.ParentID == nil {
		if want.ParentID != nil {
			t.Errorf("incorrect parent_id. have %d, want %d", have.ParentID, want.ParentID)
		}
	} else if *have.ParentID != *want.ParentID {
		t.Errorf("incorrect parent_id. have %d, want %d", have.ParentID, want.ParentID)
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
