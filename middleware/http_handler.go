package middleware

import (
	"net/http"
	"strconv"
	"sync/atomic"

	zipkin "github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/kind"
	"github.com/openzipkin/zipkin-go/propagation/b3"
	"github.com/openzipkin/zipkin-go/propagation/context/span"
)

type httpHandler struct {
	tracer *zipkin.Tracer
	name   string
	next   http.Handler
}

// WrapHTTPHandler wraps a standard http.Handler with Zipkin tracing.
func WrapHTTPHandler(t *zipkin.Tracer, op string, h http.Handler) http.Handler {
	return &httpHandler{
		tracer: t,
		next:   h,
		name:   op,
	}
}

// ServeHTTP implements http.Handler.
func (h httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// try to extract B3 Headers from upstream
	sc := h.tracer.Extract(b3.ExtractHTTP(r))

	// create Span using SpanContext if found
	sp := h.tracer.StartSpan(
		h.name,
		zipkin.Kind(kind.Server),
		zipkin.Parent(sc),
		zipkin.RemoteEndpoint(zipkin.NewEndpointOrNil("", r.RemoteAddr)),
	)
	defer sp.Finish()

	// add our span to context
	ctx := span.NewContext(r.Context(), sp)

	// tag typical HTTP request items
	zipkin.TagHTTPMethod.Set(sp, r.Method)
	zipkin.TagHTTPUrl.Set(sp, r.URL.String())
	zipkin.TagHTTPRequestSize.Set(sp, strconv.FormatInt(r.ContentLength, 10))

	// create http.ResponseWriter interceptor for tracking response size and
	// status code.
	ri := &rwInterceptor{w: w, statusCode: 200}

	// tag found response size and status code on exit
	defer func() {
		zipkin.TagHTTPStatusCode.Set(sp, ri.getStatusCode())
		zipkin.TagHTTPResponseSize.Set(sp, ri.getResponseSize())
	}()

	// call next http Handler func using our updated context.
	h.next.ServeHTTP(ri, r.WithContext(ctx))
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

func (r *rwInterceptor) getStatusCode() string {
	return strconv.Itoa(r.statusCode)
}

func (r *rwInterceptor) getResponseSize() string {
	return strconv.FormatUint(atomic.LoadUint64(&r.size), 10)
}
