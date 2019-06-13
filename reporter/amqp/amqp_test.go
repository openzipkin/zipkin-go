package amqp_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/openzipkin/zipkin-go/model"
	zipkinamqp "github.com/openzipkin/zipkin-go/reporter/amqp"
	"github.com/streadway/amqp"
)

var spans = []*model.SpanModel{
	makeNewSpan("avg", 123, 456, 0, true),
	makeNewSpan("sum", 123, 789, 456, true),
	makeNewSpan("div", 123, 101112, 456, true),
}

func TestRabbitProduce(t *testing.T) {
	address := "amqp://guest:guest@127.0.0.1:5672/"
	_, ch, closeFunc := setupRabbit(t, address)
	defer closeFunc()

	c, err := zipkinamqp.NewReporter(address, zipkinamqp.Channel(ch))
	if err != nil {
		t.Fatal(err)
	}

	msgs := setupConsume(t, ch)

	for _, s := range spans {
		c.Send(*s)
	}

	for _, s := range spans {
		msg := <-msgs
		ds := deserializeSpan(t, msg.Body)
		testEqual(t, s, ds)
	}
}

func TestRabbitClose(t *testing.T) {
	address := "amqp://guest:guest@127.0.0.1:5672/"
	conn, ch, closeFunc := setupRabbit(t, address)
	defer closeFunc()

	r, err := zipkinamqp.NewReporter(address, zipkinamqp.Channel(ch), zipkinamqp.Connection(conn))
	if err != nil {
		t.Fatal(err)
	}
	if err = r.Close(); err != nil {
		t.Fatal(err)
	}
}

func setupRabbit(t *testing.T, address string) (conn *amqp.Connection, ch *amqp.Channel, close func()) {
	var err error
	conn, err = amqp.Dial(address)
	failOnError(t, err, "Failed to connect to RabbitMQ")

	ch, err = conn.Channel()
	failOnError(t, err, "Failed to open a channel")

	close = func() {
		conn.Close()
		ch.Close()
	}
	return
}

func setupConsume(t *testing.T, ch *amqp.Channel) <-chan amqp.Delivery {
	csm, err := ch.Consume(
		"zipkin",
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(t, err, "Failed to register a consumer")
	return csm
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
