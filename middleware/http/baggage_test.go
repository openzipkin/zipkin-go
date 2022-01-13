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

package http_test

import (
	"net"
	"net/http"
	"testing"

	"github.com/openzipkin/zipkin-go"
	zipkinhttp "github.com/openzipkin/zipkin-go/middleware/http"
	"github.com/openzipkin/zipkin-go/propagation/baggage"
)

const (
	reqID            = "X-Request-Id"
	reqIDValue       = "5a3553a7-4088-4ae0-8845-8314ebd59ddb"
	customField      = "custom-field"
	customFieldValue = "custom-value"
)

func TestHTTPBaggage(t *testing.T) {
	var (
		tracer, _  = zipkin.NewTracer(nil)
		tr, _      = zipkinhttp.NewTransport(tracer)
		cli        = &http.Client{Transport: tr}
		srv        = newServer(cli)
		bagHandler = baggage.New(reqID, customField)
	)

	// attach server middleware to http server
	srv.s.Handler = zipkinhttp.NewServerMiddleware(
		tracer,
		zipkinhttp.EnableBaggage(bagHandler),
	)(srv.s.Handler)

	// create listener
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("unable to create listener for http server: %+v", err)
	}
	defer func() {
		_ = ln.Close()
	}()

	// start http server
	go func() {
		srv.s.Addr = ln.Addr().String()
		_ = srv.s.Serve(ln)
	}()

	// generate request to handler1 with X-Request-Id set
	req, err := http.NewRequest("GET", "http://"+ln.Addr().String()+"/handler1", nil)
	if err != nil {
		t.Fatalf("unable to create initial http request: %+v", err)
	}
	req.Header.Add(reqID, reqIDValue)

	// send client request
	if _, err = cli.Do(req); err != nil {
		t.Errorf("unexpected http request error: %+v", err)
	}

	// check server inspection variables for correct baggage field propagation
	if srv.resultHandler1 != reqIDValue {
		t.Errorf("resultHandler1 expected propagated %s: want %s, have: %s",
			reqID, reqIDValue, srv.resultHandler1)
	}
	if srv.result1Handler2 != reqIDValue {
		t.Errorf("result1Handler2 expected propagated %s: want %s, have: %s",
			reqID, reqIDValue, srv.result1Handler2)
	}
	if srv.result2Handler2 != customFieldValue {
		t.Errorf("result2Handler2 expected propagated %s: want %s, have: %s",
			customField, customFieldValue, srv.result2Handler2)
	}

}

type server struct {
	s               *http.Server
	c               *http.Client
	resultHandler1  string
	result1Handler2 string
	result2Handler2 string
}

func newServer(client *http.Client) *server {
	mux := http.NewServeMux()
	s := &server{
		c: client,
		s: &http.Server{Handler: mux},
	}
	mux.HandleFunc("/handler1", s.handler1)
	mux.HandleFunc("/handler2", s.handler2)
	return s
}

func (s *server) handler1(w http.ResponseWriter, h *http.Request) {
	// store received request id for inspection
	s.resultHandler1 = h.Header.Get(reqID)
	// add additional baggage field
	ctx := h.Context()
	if span := zipkin.SpanFromContext(ctx); span != nil {
		span.Context().Baggage.Add(customField, customFieldValue)
	}
	// call handler2
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://"+s.s.Addr+"/handler2", nil)
	if _, err := s.c.Do(req); err != nil {
		http.Error(w, http.StatusText(500), 500)
		return
	}
	w.WriteHeader(201)
}

func (s *server) handler2(w http.ResponseWriter, h *http.Request) {
	// store received request id for inspection
	s.result1Handler2 = h.Header.Get(reqID)
	s.result2Handler2 = h.Header.Get(customField)
	w.WriteHeader(201)
}
