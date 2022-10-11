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

package zipkin_test

import (
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/openzipkin/zipkin-go"
	zipkinhttp "github.com/openzipkin/zipkin-go/middleware/http"
	logreporter "github.com/openzipkin/zipkin-go/reporter/log"
)

func Example() {
	// set up a span reporter
	reporter := logreporter.NewReporter(log.New(os.Stderr, "", log.LstdFlags))
	defer func() {
		_ = reporter.Close()
	}()

	// create our local service endpoint
	endpoint, err := zipkin.NewEndpoint("myService", "localhost:0")
	if err != nil {
		log.Fatalf("unable to create local endpoint: %+v\n", err)
	}

	// initialize our tracer
	tracer, err := zipkin.NewTracer(reporter, zipkin.WithLocalEndpoint(endpoint))
	if err != nil {
		log.Fatalf("unable to create tracer: %+v\n", err)
	}

	// create global zipkin http server middleware
	serverMiddleware := zipkinhttp.NewServerMiddleware(
		tracer, zipkinhttp.TagResponseSize(true),
	)

	// create global zipkin traced http client
	client, err := zipkinhttp.NewClient(tracer, zipkinhttp.ClientTrace(true))
	if err != nil {
		log.Fatalf("unable to create client: %+v\n", err)
	}

	// initialize router
	router := http.NewServeMux()

	// start web service with zipkin http server middleware
	ts := httptest.NewServer(serverMiddleware(router))
	defer ts.Close()

	// set-up handlers
	router.HandleFunc("/some_function", someFunc(client, ts.URL))
	router.HandleFunc("/other_function", otherFunc(client))

	// initiate a call to some_func
	var req *http.Request
	req, err = http.NewRequest("GET", ts.URL+"/some_function", nil)
	if err != nil {
		log.Fatalf("unable to create http request: %+v\n", err)
	}

	var res *http.Response
	res, err = client.DoWithAppSpan(req, "some_function")
	if err != nil {
		log.Fatalf("unable to do http request: %+v\n", err)
	}
	_ = res.Body.Close()

	// Output:
}

func someFunc(client *zipkinhttp.Client, url string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("some_function called with method: %s\n", r.Method)

		// retrieve span from context (created by server middleware)
		span := zipkin.SpanFromContext(r.Context())
		span.Tag("custom_key", "some value")

		// doing some expensive calculations....
		time.Sleep(25 * time.Millisecond)
		span.Annotate(time.Now(), "expensive_calc_done")

		newRequest, err := http.NewRequest("POST", url+"/other_function", nil)
		if err != nil {
			log.Printf("unable to create client: %+v\n", err)
			http.Error(w, err.Error(), 500)
			return
		}

		ctx := zipkin.NewContext(newRequest.Context(), span)

		newRequest = newRequest.WithContext(ctx)

		var res *http.Response
		res, err = client.DoWithAppSpan(newRequest, "other_function")
		if err != nil {
			log.Printf("call to other_function returned error: %+v\n", err)
			http.Error(w, err.Error(), 500)
			return
		}
		_ = res.Body.Close()
	}
}

func otherFunc(_ *zipkinhttp.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("other_function called with method: %s\n", r.Method)
		time.Sleep(50 * time.Millisecond)
	}
}
