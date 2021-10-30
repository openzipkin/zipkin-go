// Copyright 2021 The OpenZipkin Authors
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
	"github.com/openzipkin/zipkin-go/idgenerator"
	zipkinhttp "github.com/openzipkin/zipkin-go/middleware/http"
	"github.com/openzipkin/zipkin-go/propagation/baggage"
)

const ReqID = "X-Request-Id"

func TestHTTPBaggage(t *testing.T) {
	var (
		tracer, _ = zipkin.NewTracer(nil)
		tr, _     = zipkinhttp.NewTransport(tracer)
		srv       = newServer(&http.Client{Transport: tr})
	)

	// attach server middleware to http server
	srv.s.Handler = zipkinhttp.NewServerMiddleware(
		tracer,
		zipkinhttp.EnableBaggage(baggage.New(ReqID)),
	)(srv.s.Handler)

	// create listener
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("unable to create listener for http server: %+v", err)
	}
	defer func() {
		_ = ln.Close()
	}()
	srv.s.Addr = ln.Addr().String()

	// start http server
	go func() {
		_ = srv.s.Serve(ln)
	}()

	// generate request to handler1 with X-Request-Id set
	req, err := http.NewRequest("GET", "http://"+ln.Addr().String()+"/handler1", nil)
	if err != nil {
		t.Fatalf("unable to create initial http request: %+v", err)
	}
	reqID := idgenerator.NewRandom128().TraceID().String()
	req.Header.Add(ReqID, reqID)

	// send client request
	cli := &http.Client{}
	if _, err = cli.Do(req); err != nil {
		t.Errorf("unexpected http request error: %+v", err)
	}

	// check server inspection for request id propagation
	if srv.h1 != reqID {
		t.Errorf("h1 expected propagated %s: want %s, have: %s", ReqID, reqID, srv.h1)
	}
	if srv.h2 != reqID {
		t.Errorf("h2 expected propagated %s: want %s, have: %s", ReqID, reqID, srv.h2)
	}
}

type server struct {
	s      *http.Server
	c      *http.Client
	h1, h2 string
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
	s.h1 = h.Header.Get(ReqID)
	// call handler2
	req, _ := http.NewRequestWithContext(h.Context(), "GET", "http://"+s.s.Addr+"/handler2", nil)
	if _, err := s.c.Do(req); err != nil {
		http.Error(w, http.StatusText(500), 500)
		return
	}
	w.WriteHeader(201)
}

func (s *server) handler2(w http.ResponseWriter, h *http.Request) {
	// store received request id for inspection
	s.h2 = h.Header.Get(ReqID)
	w.WriteHeader(201)
}
