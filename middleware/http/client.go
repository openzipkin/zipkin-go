package http

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	zipkin "github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
)

// ErrValidTracerRequired error
var ErrValidTracerRequired = errors.New("valid tracer required")

// Client holds a Zipkin instrumented HTTP Client.
type Client struct {
	*http.Client
	tracer *zipkin.Tracer
}

// NewClient returns an HTTP Client adding Zipkin instrumentation around an
// embedded standard Go http.Client.
func NewClient(tracer *zipkin.Tracer, client *http.Client, options ...TransportOption) (*Client, error) {
	if tracer == nil {
		return nil, ErrValidTracerRequired
	}

	if client == nil {
		client = &http.Client{}
	}

	options = append(options, WithRoundTripper(client.Transport))
	transport, err := NewTransport(tracer, options...)
	if err != nil {
		return nil, err
	}
	client.Transport = transport

	return &Client{tracer: tracer, Client: client}, nil
}

// DoWithTrace wraps http.Client's Do with tracing using an application span.
func (c *Client) DoWithTrace(req *http.Request, name string) (res *http.Response, err error) {
	appSpan := c.tracer.StartSpan(name, zipkin.Kind(model.Client))

	zipkin.TagHTTPMethod.Set(appSpan, req.Method)
	zipkin.TagHTTPUrl.Set(appSpan, req.URL.String())
	zipkin.TagHTTPPath.Set(appSpan, req.URL.Path)

	res, err = c.Client.Do(
		req.WithContext(zipkin.NewContext(context.Background(), appSpan)),
	)
	if err != nil {
		zipkin.TagError.Set(appSpan, err.Error())
		appSpan.Finish()
		return
	}

	var traceEnabled bool
	if tr, ok := c.Transport.(*transport); ok {
		if tr.httptraceEnabled {
			traceEnabled = tr.httptraceEnabled
			appSpan.Annotate(time.Now(), "wr")
		}
	}

	statusCode := strconv.FormatInt(int64(res.StatusCode), 10)
	zipkin.TagHTTPStatusCode.Set(appSpan, statusCode)
	zipkin.TagHTTPResponseSize.Set(appSpan, strconv.FormatInt(res.ContentLength, 10))
	if res.StatusCode > 399 {
		zipkin.TagError.Set(appSpan, statusCode)
	}

	res.Body = &spanCloser{
		ReadCloser:   res.Body,
		sp:           appSpan,
		traceEnabled: traceEnabled,
	}
	return
}
