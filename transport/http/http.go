/*
Package http implements a transport to send spans to Zipkin V2 collectors.
*/
package http

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/openzipkin/zipkin-go"
)

// defaults
const (
	defaultTimeout       = time.Second * 5 // timeout for http request in seconds
	defaultBatchInterval = time.Second * 1 // BatchInterval in seconds
	defaultBatchSize     = 100
	defaultMaxBacklog    = 1000
)

type transport struct {
	url           string
	client        *http.Client
	logger        *log.Logger
	batchInterval time.Duration
	batchSize     int
	maxBacklog    int
	sendMtx       *sync.Mutex
	batchMtx      *sync.Mutex
	batch         []*zipkin.SpanModel
	spanc         chan *zipkin.SpanModel
	quit          chan struct{}
	shutdown      chan error
	reqCallback   RequestCallbackFn
}

// Send implements transporter
func (t *transport) Send(s zipkin.SpanModel) {
	t.spanc <- &s
}

// Close implements transporter
func (t *transport) Close() error {
	close(t.quit)
	return <-t.shutdown
}

func (t *transport) loop() {
	var (
		nextSend   = time.Now().Add(t.batchInterval)
		ticker     = time.NewTicker(t.batchInterval / 10)
		tickerChan = ticker.C
	)
	defer ticker.Stop()

	for {
		select {
		case span := <-t.spanc:
			currentBatchSize := t.append(span)
			if currentBatchSize >= t.batchSize {
				nextSend = time.Now().Add(t.batchInterval)
				go t.sendBatch()
			}
		case <-tickerChan:
			if time.Now().After(nextSend) {
				nextSend = time.Now().Add(t.batchInterval)
				go t.sendBatch()
			}
		case <-t.quit:
			t.shutdown <- t.sendBatch()
			return
		}
	}
}

func (t *transport) append(span *zipkin.SpanModel) (newBatchSize int) {
	t.batchMtx.Lock()
	defer t.batchMtx.Unlock()

	t.batch = append(t.batch, span)
	if len(t.batch) > t.maxBacklog {
		dispose := len(t.batch) - t.maxBacklog
		t.logger.Printf("backlog too long, disposing %d spans", dispose)
		t.batch = t.batch[dispose:]
	}
	newBatchSize = len(t.batch)
	return
}

func (t *transport) sendBatch() error {
	// in order to prevent sending the same batch twice
	t.sendMtx.Lock()
	defer t.sendMtx.Unlock()

	// Select all current spans in the batch to be sent
	t.batchMtx.Lock()
	sendBatch := t.batch[:]
	t.batchMtx.Unlock()

	if len(sendBatch) == 0 {
		return nil
	}

	body, err := json.Marshal(sendBatch)
	if err != nil {
		t.logger.Printf("failed when unmarshalling the spans batch: %s\n", err.Error())
		return err
	}

	req, err := http.NewRequest("POST", t.url, bytes.NewReader(body))
	if err != nil {
		t.logger.Printf("failed when creating the request: %s\n", err.Error())
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if t.reqCallback != nil {
		t.reqCallback(req)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		t.logger.Printf("failed to send the request: %s\n", err.Error())
		return err
	}
	resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		t.logger.Printf("failed the request with status code %d\n", resp.StatusCode)
	}

	// Remove sent spans from the batch even if they were not saved
	t.batchMtx.Lock()
	t.batch = t.batch[len(sendBatch):]
	t.batchMtx.Unlock()

	return nil
}

// RequestCallbackFn receives the initialized request from the Collector before
// sending it over the wire. This allows one to plug in additional headers or
// do other customization.
type RequestCallbackFn func(*http.Request)

// TransportOption sets a parameter for the HTTP Transporter
type TransportOption func(t *transport)

// Timeout sets maximum timeout for http request.
func Timeout(duration time.Duration) TransportOption {
	return func(t *transport) { t.client.Timeout = duration }
}

// BatchSize sets the maximum batch size, after which a collect will be
// triggered. The default batch size is 100 traces.
func BatchSize(n int) TransportOption {
	return func(t *transport) { t.batchSize = n }
}

// MaxBacklog sets the maximum backlog size,
// when batch size reaches this threshold, spans from the
// beginning of the batch will be disposed
func MaxBacklog(n int) TransportOption {
	return func(t *transport) { t.maxBacklog = n }
}

// BatchInterval sets the maximum duration we will buffer traces before
// emitting them to the collector. The default batch interval is 1 second.
func BatchInterval(d time.Duration) TransportOption {
	return func(t *transport) { t.batchInterval = d }
}

// Client sets a custom http client to use.
func Client(client *http.Client) TransportOption {
	return func(t *transport) { t.client = client }
}

// RequestCallback registers a callback function to adjust the collector
// *http.Request before it sends the request to Zipkin.
func RequestCallback(rc RequestCallbackFn) TransportOption {
	return func(t *transport) { t.reqCallback = rc }
}

// NewTransport returns a new HTTP Transporter.
func NewTransport(url string, opts ...TransportOption) zipkin.Transporter {
	t := transport{
		url:           url,
		logger:        &log.Logger{},
		client:        &http.Client{Timeout: defaultTimeout},
		batchInterval: defaultBatchInterval,
		batchSize:     defaultBatchSize,
		maxBacklog:    defaultMaxBacklog,
		batch:         []*zipkin.SpanModel{},
		spanc:         make(chan *zipkin.SpanModel),
		quit:          make(chan struct{}, 1),
		shutdown:      make(chan error, 1),
		sendMtx:       &sync.Mutex{},
		batchMtx:      &sync.Mutex{},
	}

	for _, opt := range opts {
		opt(&t)
	}

	go t.loop()

	return &t
}
