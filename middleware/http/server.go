package http

import (
	"io"
	"net/http"
	"strconv"
	"sync/atomic"

	zipkin "github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/propagation/b3"
)

type handler struct {
	tracer          *zipkin.Tracer
	name            string
	next            http.Handler
	tagResponseSize bool
	defaultTags     map[string]string
	requestSampler  func(r *http.Request) bool
}

// ServerOption allows Middleware to be optionally configured.
type ServerOption func(*handler)

// ServerTags adds default Tags to inject into server spans.
func ServerTags(tags map[string]string) ServerOption {
	return func(h *handler) {
		h.defaultTags = tags
	}
}

// TagResponseSize will instruct the middleware to Tag the http response size
// in the server side span.
func TagResponseSize(enabled bool) ServerOption {
	return func(h *handler) {
		h.tagResponseSize = enabled
	}
}

// SpanName sets the name of the spans the middleware creates. Use this if
// wrapping each endpoint with its own Middleware.
// If omitting the SpanName option, the middleware will use the http request
// method as span name.
func SpanName(name string) ServerOption {
	return func(h *handler) {
		h.name = name
	}
}

// RequestSampler allows one to set the sampling decision based on the details
// found in the http.Request.
func RequestSampler(sampleFunc func(r *http.Request) bool) ServerOption {
	return func(h *handler) {
		h.requestSampler = sampleFunc
	}
}

// NewServerMiddleware returns a http.Handler middleware with Zipkin tracing.
func NewServerMiddleware(t *zipkin.Tracer, options ...ServerOption) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		h := &handler{
			tracer: t,
			next:   next,
		}
		for _, option := range options {
			option(h)
		}
		return h
	}
}

// ServeHTTP implements http.Handler.
func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var spanName string

	// try to extract B3 Headers from upstream
	sc := h.tracer.Extract(b3.ExtractHTTP(r))

	if h.requestSampler != nil && sc.Sampled == nil {
		sample := h.requestSampler(r)
		sc.Sampled = &sample
	}

	remoteEndpoint, _ := zipkin.NewEndpoint("", r.RemoteAddr)

	if len(h.name) == 0 {
		spanName = r.Method
	} else {
		spanName = h.name
	}

	// create Span using SpanContext if found
	sp := h.tracer.StartSpan(
		spanName,
		zipkin.Kind(model.Server),
		zipkin.Parent(sc),
		zipkin.RemoteEndpoint(remoteEndpoint),
	)

	for k, v := range h.defaultTags {
		sp.Tag(k, v)
	}

	// add our span to context
	ctx := zipkin.NewContext(r.Context(), sp)

	// tag typical HTTP request items
	zipkin.TagHTTPMethod.Set(sp, r.Method)
	zipkin.TagHTTPPath.Set(sp, r.URL.Path)
	zipkin.TagHTTPRequestSize.Set(sp, strconv.FormatInt(r.ContentLength, 10))

	// create http.ResponseWriter interceptor for tracking response size and
	// status code.
	ri := &rwInterceptor{w: w, statusCode: 200}

	// tag found response size and status code on exit
	defer func() {
		code := ri.getStatusCode()
		sCode := strconv.Itoa(code)
		if code > 399 {
			zipkin.TagError.Set(sp, sCode)
		}
		zipkin.TagHTTPStatusCode.Set(sp, sCode)
		if h.tagResponseSize {
			zipkin.TagHTTPResponseSize.Set(sp, ri.getResponseSize())
		}
		sp.Finish()
	}()

	// call next http Handler func using our updated context.
	h.next.ServeHTTP(ri.wrap(), r.WithContext(ctx))
}

// rwInterceptor intercepts the ResponseWriter so it can track response size
// and returned status code.
type rwInterceptor struct {
	w          http.ResponseWriter
	size       uint64
	statusCode int
}

func (r *rwInterceptor) Header() http.Header {
	return r.w.Header()
}

func (r *rwInterceptor) Write(b []byte) (n int, err error) {
	n, err = r.w.Write(b)
	atomic.AddUint64(&r.size, uint64(n))
	return
}

func (r *rwInterceptor) WriteHeader(i int) {
	r.statusCode = i
	r.w.WriteHeader(i)
}

func (r *rwInterceptor) getStatusCode() int {
	return r.statusCode
}

func (r *rwInterceptor) getResponseSize() string {
	return strconv.FormatUint(atomic.LoadUint64(&r.size), 10)
}

func (t *rwInterceptor) wrap() http.ResponseWriter {
	var (
		hj, i0 = t.w.(http.Hijacker)
		cn, i1 = t.w.(http.CloseNotifier)
		pu, i2 = t.w.(http.Pusher)
		fl, i3 = t.w.(http.Flusher)
		rf, i4 = t.w.(io.ReaderFrom)
	)

	switch {
	case !i0 && !i1 && !i2 && !i3 && !i4:
		return struct {
			http.ResponseWriter
		}{t}
	case !i0 && !i1 && !i2 && !i3 && i4:
		return struct {
			http.ResponseWriter
			io.ReaderFrom
		}{t, rf}
	case !i0 && !i1 && !i2 && i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Flusher
		}{t, fl}
	case !i0 && !i1 && !i2 && i3 && i4:
		return struct {
			http.ResponseWriter
			http.Flusher
			io.ReaderFrom
		}{t, fl, rf}
	case !i0 && !i1 && i2 && !i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Pusher
		}{t, pu}
	case !i0 && !i1 && i2 && !i3 && i4:
		return struct {
			http.ResponseWriter
			http.Pusher
			io.ReaderFrom
		}{t, pu, rf}
	case !i0 && !i1 && i2 && i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Pusher
			http.Flusher
		}{t, pu, fl}
	case !i0 && !i1 && i2 && i3 && i4:
		return struct {
			http.ResponseWriter
			http.Pusher
			http.Flusher
			io.ReaderFrom
		}{t, pu, fl, rf}
	case !i0 && i1 && !i2 && !i3 && !i4:
		return struct {
			http.ResponseWriter
			http.CloseNotifier
		}{t, cn}
	case !i0 && i1 && !i2 && !i3 && i4:
		return struct {
			http.ResponseWriter
			http.CloseNotifier
			io.ReaderFrom
		}{t, cn, rf}
	case !i0 && i1 && !i2 && i3 && !i4:
		return struct {
			http.ResponseWriter
			http.CloseNotifier
			http.Flusher
		}{t, cn, fl}
	case !i0 && i1 && !i2 && i3 && i4:
		return struct {
			http.ResponseWriter
			http.CloseNotifier
			http.Flusher
			io.ReaderFrom
		}{t, cn, fl, rf}
	case !i0 && i1 && i2 && !i3 && !i4:
		return struct {
			http.ResponseWriter
			http.CloseNotifier
			http.Pusher
		}{t, cn, pu}
	case !i0 && i1 && i2 && !i3 && i4:
		return struct {
			http.ResponseWriter
			http.CloseNotifier
			http.Pusher
			io.ReaderFrom
		}{t, cn, pu, rf}
	case !i0 && i1 && i2 && i3 && !i4:
		return struct {
			http.ResponseWriter
			http.CloseNotifier
			http.Pusher
			http.Flusher
		}{t, cn, pu, fl}
	case !i0 && i1 && i2 && i3 && i4:
		return struct {
			http.ResponseWriter
			http.CloseNotifier
			http.Pusher
			http.Flusher
			io.ReaderFrom
		}{t, cn, pu, fl, rf}
	case i0 && !i1 && !i2 && !i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
		}{t, hj}
	case i0 && !i1 && !i2 && !i3 && i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			io.ReaderFrom
		}{t, hj, rf}
	case i0 && !i1 && !i2 && i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.Flusher
		}{t, hj, fl}
	case i0 && !i1 && !i2 && i3 && i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.Flusher
			io.ReaderFrom
		}{t, hj, fl, rf}
	case i0 && !i1 && i2 && !i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.Pusher
		}{t, hj, pu}
	case i0 && !i1 && i2 && !i3 && i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.Pusher
			io.ReaderFrom
		}{t, hj, pu, rf}
	case i0 && !i1 && i2 && i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.Pusher
			http.Flusher
		}{t, hj, pu, fl}
	case i0 && !i1 && i2 && i3 && i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.Pusher
			http.Flusher
			io.ReaderFrom
		}{t, hj, pu, fl, rf}
	case i0 && i1 && !i2 && !i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.CloseNotifier
		}{t, hj, cn}
	case i0 && i1 && !i2 && !i3 && i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.CloseNotifier
			io.ReaderFrom
		}{t, hj, cn, rf}
	case i0 && i1 && !i2 && i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.CloseNotifier
			http.Flusher
		}{t, hj, cn, fl}
	case i0 && i1 && !i2 && i3 && i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.CloseNotifier
			http.Flusher
			io.ReaderFrom
		}{t, hj, cn, fl, rf}
	case i0 && i1 && i2 && !i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.CloseNotifier
			http.Pusher
		}{t, hj, cn, pu}
	case i0 && i1 && i2 && !i3 && i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.CloseNotifier
			http.Pusher
			io.ReaderFrom
		}{t, hj, cn, pu, rf}
	case i0 && i1 && i2 && i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.CloseNotifier
			http.Pusher
			http.Flusher
		}{t, hj, cn, pu, fl}
	case i0 && i1 && i2 && i3 && i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.CloseNotifier
			http.Pusher
			http.Flusher
			io.ReaderFrom
		}{t, hj, cn, pu, fl, rf}
	default:
		return struct {
			http.ResponseWriter
		}{t}
	}
}
