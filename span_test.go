package zipkin

import (
	"reflect"
	"testing"

	"github.com/openzipkin/zipkin-go/reporter/recorder"
)

func TestSpanNameUpdate(t *testing.T) {
	var (
		oldName = "oldName"
		newName = "newName"
	)
	reporter := recorder.NewReporter()
	defer reporter.Close()

	tracer, _ := NewTracer(reporter)

	span := tracer.StartSpan(oldName)

	if want, have := oldName, span.(*spanImpl).Name; want != have {
		t.Errorf("Name want %q, have %q", want, have)
	}

	span.SetName(newName)

	if want, have := newName, span.(*spanImpl).Name; want != have {
		t.Errorf("Name want %q, have %q", want, have)
	}
}

func TestRemoteEndpoint(t *testing.T) {
	rec := recorder.NewReporter()
	defer rec.Close()

	tracer, err := NewTracer(rec)
	if err != nil {
		t.Fatalf("expected valid tracer, got error: %+v", err)
	}

	ep1, err := NewEndpoint("myService", "www.google.com:80")

	if err != nil {
		t.Fatalf("expected valid endpoint, got error: %+v", err)
	}

	span := tracer.StartSpan("test", RemoteEndpoint(ep1))

	if !reflect.DeepEqual(span.(*spanImpl).RemoteEndpoint, ep1) {
		t.Errorf("RemoteEndpoint want %+v, have %+v", ep1, span.(*spanImpl).RemoteEndpoint)
	}

	ep2, err := NewEndpoint("otherService", "www.microsoft.com:443")

	if err != nil {
		t.Fatalf("expected valid endpoint, got error: %+v", err)
	}

	span.SetRemoteEndpoint(ep2)

	if !reflect.DeepEqual(span.(*spanImpl).RemoteEndpoint, ep2) {
		t.Errorf("RemoteEndpoint want %+v, have %+v", ep1, span.(*spanImpl).RemoteEndpoint)
	}

	span.SetRemoteEndpoint(nil)

	if have := span.(*spanImpl).RemoteEndpoint; have != nil {
		t.Errorf("RemoteEndpoint want nil, have %+v", have)
	}
}
