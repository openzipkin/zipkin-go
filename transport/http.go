package transport

import (
	"github.com/openzipkin/zipkin-go"
	"net/http"
	"sync"
	"encoding/json"
	"log"
	"bytes"
	"time"
)

// Default timeout for http request in seconds
const defaultHTTPTimeout = time.Second * 5

// defaultBatchInterval in seconds
const defaultHTTPBatchInterval = 1

const defaultHTTPBatchSize = 100

const defaultHTTPMaxBacklog = 1000

type HTTPTransport struct {
	url           string
	client        *http.Client
	logger        *log.Logger
	batchInterval time.Duration
	batchSize     int
	maxBacklog    int
	sendMutex     *sync.Mutex
	batchMutex    *sync.Mutex
	batch         []*zipkin.SpanModel
	spanChan      chan *zipkin.SpanModel
}

func (t *HTTPTransport) Send(s zipkin.SpanModel) {
	t.spanChan <- &s
}

func (t *HTTPTransport) loop() {
	var (
		nextSend   = time.Now().Add(t.batchInterval)
		ticker     = time.NewTicker(t.batchInterval / 10)
		tickerChan = ticker.C
	)

	// Sends
	defer t.sendBatch()

	defer ticker.Stop()

	for {
		select {
		case span := <-t.spanChan:
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
		}
	}
}

func (t *HTTPTransport) append(span *zipkin.SpanModel) (newBatchSize int) {
	t.batchMutex.Lock()
	defer t.batchMutex.Unlock()

	t.batch = append(t.batch, span)
	if len(t.batch) > t.maxBacklog {
		dispose := len(t.batch) - t.maxBacklog
		t.logger.Printf("backlog too long, disposing %d spans", dispose)
		t.batch = t.batch[dispose:]
	}
	newBatchSize = len(t.batch)
	return
}

func (t *HTTPTransport) sendBatch() error {
	// in order to prevent sending the same batch twice
	t.sendMutex.Lock()
	defer t.sendMutex.Unlock()

	// Select all current spans in the batch to be sent
	t.batchMutex.Lock()
	sendBatch := t.batch[:]
	t.batchMutex.Unlock()

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
	t.batchMutex.Lock()
	t.batch = t.batch[len(sendBatch):]
	t.batchMutex.Unlock()

	return nil
}

type HTTPTransportOpt func(t *HTTPTransport)

// HTTPTimeout sets maximum timeout for http request.
func HTTPTimeout(duration time.Duration) HTTPTransportOpt {
	return func(c *HTTPTransport) { c.client.Timeout = duration }
}

// HTTPBatchSize sets the maximum batch size, after which a collect will be
// triggered. The default batch size is 100 traces.
func HTTPBatchSize(n int) HTTPTransportOpt {
	return func(c *HTTPTransport) { c.batchSize = n }
}

// HTTPMaxBacklog sets the maximum backlog size,
// when batch size reaches this threshold, spans from the
// beginning of the batch will be disposed
func HTTPMaxBacklog(n int) HTTPTransportOpt {
	return func(c *HTTPTransport) { c.maxBacklog = n }
}

// HTTPBatchInterval sets the maximum duration we will buffer traces before
// emitting them to the collector. The default batch interval is 1 second.
func HTTPBatchInterval(d time.Duration) HTTPTransportOpt {
	return func(c *HTTPTransport) { c.batchInterval = d }
}

// HTTPClient sets a custom http client to use.
func HTTPClient(client *http.Client) HTTPTransportOpt {
	return func(c *HTTPTransport) { c.client = client }
}
func NewHTTPTransport (url string, opts ...HTTPTransportOpt) zipkin.Transporter {
	t := HTTPTransport{
		url:           url,
		logger:        &log.Logger{},
		client:        &http.Client{Timeout: defaultHTTPTimeout},
		batchInterval: defaultHTTPBatchInterval * time.Second,
		batchSize:     defaultHTTPBatchSize,
		maxBacklog:    defaultHTTPMaxBacklog,
		batch:         []*zipkin.SpanModel{},
		spanChan:      make(chan *zipkin.SpanModel),
		sendMutex:     &sync.Mutex{},
		batchMutex:    &sync.Mutex{},
	}

	for _, opt := range opts {
		opt(&t)
	}

	go t.loop()

	return &t
}
