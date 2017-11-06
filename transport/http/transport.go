package http

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/davecgh/go-spew/spew"
	zipkin "github.com/openzipkin/zipkin-go"
)

type HTTPTransport struct {
}

// Send will batch and transport zipkin spans to v2 API
func (h *HTTPTransport) Send(s zipkin.SpanModel) {
	client := http.Client{}
	var spans []*spanImpl
	spans = append(spans, s)
	b, err := json.Marshal(spans)
	if err != nil {
		panic(err)
	}
	buf := bytes.NewBuffer(b)
	request, err := http.NewRequest("POST", "http://localhost:9411/api/v2/spans", buf)
	if err != nil {
		panic(err)
	}
	if res, err := client.Do(request); err != nil {
		panic(err)
	} else {
		spew.Dump(res)
	}

}
