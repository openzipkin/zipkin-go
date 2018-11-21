package amqp_test

import (
	"encoding/json"
	"github.com/openzipkin/zipkin-go/model"
	zipkinamqp "github.com/openzipkin/zipkin-go/reporter/amqp"
	"github.com/streadway/amqp"
	"testing"
	"time"
)

var spans = []*model.SpanModel{
	makeNewSpan("avg", 123, 456, 0, true),
	makeNewSpan("sum", 123, 789, 456, true),
	makeNewSpan("div", 123, 101112, 456, true),
}

func TestRabbitProduce(t *testing.T) {
	address := "amqp://guest:guest@localhost:5672/"
	c, err := zipkinamqp.NewReporter(address)
	if err != nil {
		t.Fatal(err)
	}
	msgs, closeCh := setupRabbit(t, address)
	defer closeCh()

	for _, s := range spans {
		c.Send(*s)
	}

	for _, s := range spans {
		msg := <-msgs
		ds := decodeSpan(t, msg.Body)
		testEqual(t, s, ds)
	}
}

func failOnError(t *testing.T, err error, msg string) {
	if err != nil {
		t.Fatalf("%s: %s", msg, err)
	}
}

//func TestKafkaClose(t *testing.T) {
//	p := newStubProducer(false)
//	r, err := kafka.NewReporter(
//		[]string{"192.0.2.10:9092"}, kafka.Producer(p),
//	)
//	if err != nil {
//		t.Fatal(err)
//	}
//	if err = r.Close(); err != nil {
//		t.Fatal(err)
//	}
//	if !p.closed {
//		t.Fatal("producer not closed")
//	}
//}

//func TestKafkaCloseError(t *testing.T) {
//	p := newStubProducer(true)
//	c, err := kafka.NewReporter(
//		[]string{"192.0.2.10:9092"}, kafka.Producer(p),
//	)
//	if err != nil {
//		t.Fatal(err)
//	}
//	if err = c.Close(); err == nil {
//		t.Error("no error on close")
//	}
//}

//func TestKafkaErrors(t *testing.T) {
//	p := newStubProducer(true)
//	errs := make(chan []interface{}, len(spans))
//
//	NewReporter()
//
//	c, err := kafka.NewReporter(
//		[]string{"192.0.2.10:9092"},
//		kafka.Producer(p),
//		kafka.Logger(log.New(&chanWriter{errs}, "", log.LstdFlags)),
//	)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	var have []model.SpanModel
//	for _, want := range spans {
//		message := sendSpan(t, c, p, *want)
//		messageBody, err := message.Value.Encode()
//		if err != nil {
//			t.Errorf("unexpected error: %s", err.Error())
//		}
//
//		json.Unmarshal(messageBody, &have)
//		testEqual(t, want, &have[0])
//	}
//
//	for i := 0; i < len(spans); i++ {
//		select {
//		case <-errs:
//		case <-time.After(100 * time.Millisecond):
//			t.Fatalf("errors not logged. have %d, wanted %d", i, len(spans))
//		}
//	}
//}

func setupRabbit(t *testing.T, address string) (csm <-chan amqp.Delivery, close func()) {
	conn, err := amqp.Dial(address)
	failOnError(t, err, "Failed to connect to RabbitMQ")

	ch, err := conn.Channel()
	failOnError(t, err, "Failed to open a channel")

	close = func() {
		conn.Close()
		ch.Close()
	}

	csm, err = ch.Consume(
		"zipkin", // queue
		"",       // consumer
		true,     // auto-ack
		false,    // exclusive
		false,    // no-local
		false,    // no-wait
		nil,      // args
	)
	failOnError(t, err, "Failed to register a consumer")
	return
}

func decodeSpan(t *testing.T, data []byte) *model.SpanModel {
	var receivedSpans []model.SpanModel
	err := json.Unmarshal(data, &receivedSpans)
	if err != nil {
		t.Fatal(err)
	}
	return &receivedSpans[0]
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
	} else if have.ParentID != want.ParentID {
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
