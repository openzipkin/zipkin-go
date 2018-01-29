package zipkin_test

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"

	zipkin "github.com/openzipkin/zipkin-go"
	zipkinhttp "github.com/openzipkin/zipkin-go/middleware/http"
	logreporter "github.com/openzipkin/zipkin-go/reporter/log"
)

func Example() {
	// set up a span reporter
	reporter := logreporter.NewReporter(log.New(os.Stderr, "", log.LstdFlags))
	defer reporter.Close()

	// create our local service endpoint
	endpoint, err := zipkin.NewEndpoint("myService", "localhost:8123")
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
	client, err := zipkinhttp.NewClient(tracer, nil, zipkinhttp.ClientTrace(true))
	if err != nil {
		log.Fatalf("unable to create client: %+v\n", err)
	}

	// set-up router and endpoint handlers
	router := mux.NewRouter()
	router.Methods("GET").Path("/some_function").HandlerFunc(someFunc(client))
	router.Methods("POST").Path("/other_function").HandlerFunc(otherFunc(client))

	// start web service with zipkin http server middleware
	go func() {
		http.ListenAndServe("localhost:8123", serverMiddleware(router))
	}()

	// initiate a call to some_func
	req, err := http.NewRequest("GET", "http://localhost:8123/some_function", nil)
	if err != nil {
		log.Fatalf("unable to create http request: %+v\n", err)
	}

	res, err := client.DoWithTrace(req, "some_function")
	if err != nil {
		log.Fatalf("unable to do http request: %+v\n", err)
	}
	res.Body.Close()

	// Output:
}

func someFunc(client *zipkinhttp.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("some_function called with method: %s\n", r.Method)

		// retrieve span from context (created by server middleware)
		span := zipkin.SpanFromContext(r.Context())
		span.Tag("custom_key", "some value")

		// doing some expensive calculations....
		time.Sleep(25 * time.Millisecond)
		span.Annotate(time.Now(), "expensive_calc_done")

		newRequest, err := http.NewRequest(
			"POST", "http://localhost:8123/other_function", nil,
		)
		if err != nil {
			log.Printf("unable to create client: %+v\n", err)
			http.Error(w, err.Error(), 500)
			return
		}

		ctx := zipkin.NewContext(newRequest.Context(), span)

		newRequest = newRequest.WithContext(ctx)

		res, err := client.DoWithTrace(newRequest, "other_function")
		if err != nil {
			log.Printf("call to other_function returned error: %+v\n", err)
			http.Error(w, err.Error(), 500)
			return
		}
		res.Body.Close()
	}
}

func otherFunc(client *zipkinhttp.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("other_function called with method: %s\n", r.Method)
		time.Sleep(50 * time.Millisecond)
	}
}
