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

package pulsar_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"

	"github.com/openzipkin/zipkin-go/model"
	zp3 "github.com/openzipkin/zipkin-go/proto/zipkin_proto3"
	zipkinpulsar "github.com/openzipkin/zipkin-go/reporter/pulsar"
)

var spans = []*model.SpanModel{
	makeNewSpan("avg", 123, 456, 0, true),
	makeNewSpan("sum", 123, 789, 456, true),
	makeNewSpan("div", 123, 101112, 456, true),
}

func TestPulsarProduce(t *testing.T) {
	address := os.Getenv("PULSAR_ADDR")
	if address == "" {
		t.Skip("PULSAR_ADDR not set, skipping test...")
	}
	client, producer, closeFunc := setupPulsar(t, address)
	defer closeFunc()

	reporter, err := zipkinpulsar.NewReporter(address, zipkinpulsar.Producer(producer))
	if err != nil {
		t.Fatal(err)
	}

	consume := setupConsume(t, client)
	defer consume.Close()

	for _, s := range spans {
		reporter.Send(*s)
	}

	for _, s := range spans {
		msg := <-consume.Chan()
		ds := deserializeSpan(t, msg.Payload())
		testEqual(t, s, ds)
	}
}

func TestPulsarProduceProto(t *testing.T) {
	address := os.Getenv("PULSAR_ADDR")
	if address == "" {
		t.Skip("PULSAR_ADDR not set, skipping test...")
	}
	client, producer, closeFunc := setupPulsar(t, address)
	defer closeFunc()

	reporter, err := zipkinpulsar.NewReporter(
		address,
		zipkinpulsar.Client(client),
		zipkinpulsar.Producer(producer),
		zipkinpulsar.Serializer(zp3.SpanSerializer{}),
	)
	if err != nil {
		t.Fatal(err)
	}

	consume := setupConsume(t, client)
	defer consume.Close()

	for _, s := range spans {
		reporter.Send(*s)
	}

	for _, s := range spans {
		msg := <-consume.Chan()
		ds := deserializeSpan(t, msg.Payload())
		testEqual(t, s, ds)
	}
}

func setupConsume(t *testing.T, client pulsar.Client) pulsar.Consumer {
	consumer, err := client.Subscribe(pulsar.ConsumerOptions{
		Type:             pulsar.Failover,
		Topic:            "zipkin_test",
		SubscriptionName: "zipkin_test_sub",
	})
	failOnError(t, err, "Failed to subscribe to Pulsar")
	return consumer
}

func setupPulsar(t *testing.T, address string) (pulsar.Client, pulsar.Producer, func()) {
	var err error
	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL: address,
	})
	failOnError(t, err, "Failed to connect to Pulsar")

	producer, err := client.CreateProducer(pulsar.ProducerOptions{
		Topic: "zipkin_test",
	})
	failOnError(t, err, "Failed to create Pulsar producer")

	return client, producer, func() {
		producer.Close()
		client.Close()
	}
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

func deserializeSpan(t *testing.T, data []byte) *model.SpanModel {
	var receivedSpans []model.SpanModel
	err := json.Unmarshal(data, &receivedSpans)
	if err != nil {
		t.Fatal(err)
	}
	return &receivedSpans[0]
}

func failOnError(t *testing.T, err error, msg string) {
	if err != nil {
		t.Fatalf("%s: %s", msg, err)
	}
}

func makeNewSpan(methodName string, traceID, spanID, parentSpanID uint64, debug bool) *model.SpanModel {
	timestamp := time.Now()
	parentID := new(model.ID)
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
