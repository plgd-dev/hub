// Copyright The OpenTelemetry Authors
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

package otelhttp // import "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/felixge/httpsnoop"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
)

// Handler is http middleware that corresponds to the http.Handler interface and
// is designed to wrap a http.Mux (or equivalent), while individual routes on
// the mux are wrapped with WithRouteTag. A Handler will add various attributes
// to the span using the attribute.Keys defined in this package.
type Handler struct {
	operation string
	handler   http.Handler

	tracer            trace.Tracer
	meter             metric.Meter
	propagators       propagation.TextMapPropagator
	spanStartOptions  []trace.SpanStartOption
	readEvent         bool
	writeEvent        bool
	filters           []otelhttp.Filter
	spanNameFormatter func(string, *http.Request) string
	counters          map[string]metric.Int64Counter
	valueRecorders    map[string]metric.Float64Histogram
}

func defaultHandlerFormatter(operation string, _ *http.Request) string {
	return operation
}

// NewHandler wraps the passed handler, functioning like middleware, in a span
// named after the operation and with any provided Options.
func NewHandler(handler http.Handler, operation string, opts ...Option) http.Handler {
	h := Handler{
		handler:   handler,
		operation: operation,
	}

	defaultOpts := []Option{
		WithSpanOptions(trace.WithSpanKind(trace.SpanKindServer)),
		WithSpanNameFormatter(defaultHandlerFormatter),
	}

	c := newConfig(append(defaultOpts, opts...)...)
	h.configure(c)
	h.createMeasures()

	return &h
}

func (h *Handler) configure(c *config) {
	h.tracer = c.Tracer
	h.meter = c.Meter
	h.propagators = c.Propagators
	h.spanStartOptions = c.SpanStartOptions
	h.readEvent = c.ReadEvent
	h.writeEvent = c.WriteEvent
	h.filters = c.Filters
	h.spanNameFormatter = c.SpanNameFormatter
}

func handleErr(err error) {
	if err != nil {
		otel.Handle(err)
	}
}

func (h *Handler) createMeasures() {
	h.counters = make(map[string]metric.Int64Counter)
	h.valueRecorders = make(map[string]metric.Float64Histogram)

	requestBytesCounter, err := h.meter.Int64Counter(RequestContentLength)
	handleErr(err)

	responseBytesCounter, err := h.meter.Int64Counter(ResponseContentLength)
	handleErr(err)

	serverLatencyMeasure, err := h.meter.Float64Histogram(ServerLatency)
	handleErr(err)

	h.counters[RequestContentLength] = requestBytesCounter
	h.counters[ResponseContentLength] = responseBytesCounter
	h.valueRecorders[ServerLatency] = serverLatencyMeasure
}

// ServeHTTP serves HTTP requests (http.Handler)
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestStartTime := time.Now()
	for _, f := range h.filters {
		if !f(r) {
			// Simply pass through to the handler if a filter rejects the request
			h.handler.ServeHTTP(w, r)
			return
		}
	}

	opts := append([]trace.SpanStartOption{
		trace.WithAttributes(semconv.NetAttributesFromHTTPRequest("tcp", r)...),
		trace.WithAttributes(semconv.EndUserAttributesFromHTTPRequest(r)...),
		trace.WithAttributes(semconv.HTTPServerAttributesFromHTTPRequest(h.operation, "", r)...),
	}, h.spanStartOptions...) // start with the configured options

	tracer := h.tracer

	if tracer == nil {
		if span := trace.SpanFromContext(r.Context()); span.SpanContext().IsValid() {
			tracer = newTracer(span.TracerProvider())
		} else {
			tracer = newTracer(otel.GetTracerProvider())
		}
	}

	ctx := h.propagators.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
	ctx, span := tracer.Start(ctx, h.spanNameFormatter(h.operation, r), opts...)
	defer span.End()

	readRecordFunc := func(int64) {
		// by default do nothing
	}
	if h.readEvent {
		readRecordFunc = func(n int64) {
			span.AddEvent("read", trace.WithAttributes(ReadBytesKey.Int64(n)))
		}
	}

	var bw bodyWrapper
	// if request body is nil we don't want to mutate the body as it will affect
	// the identity of it in a unforeseeable way because we assert ReadCloser
	// fullfills a certain interface and it is indeed nil.
	if r.Body != nil {
		bw.ReadCloser = r.Body
		bw.record = readRecordFunc
		r.Body = &bw
	}

	writeRecordFunc := func(int64) {
		// by default do nothing
	}
	if h.writeEvent {
		writeRecordFunc = func(n int64) {
			span.AddEvent("write", trace.WithAttributes(WroteBytesKey.Int64(n)))
		}
	}

	rww := &respWriterWrapper{ResponseWriter: w, record: writeRecordFunc, ctx: ctx, props: h.propagators}

	// Wrap w to use our ResponseWriter methods while also exposing
	// other interfaces that w may implement (http.CloseNotifier,
	// http.Flusher, http.Hijacker, http.Pusher, io.ReaderFrom).

	hooks := httpsnoop.Hooks{
		Header: func(httpsnoop.HeaderFunc) httpsnoop.HeaderFunc {
			return rww.Header
		},
		Write: func(httpsnoop.WriteFunc) httpsnoop.WriteFunc {
			return rww.Write
		},
		WriteHeader: func(httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
			return rww.WriteHeader
		},
	}
	if _, ok := w.(http.Flusher); ok {
		hooks.Flush = func(httpsnoop.FlushFunc) httpsnoop.FlushFunc {
			return rww.Flush
		}
	}
	if _, ok := w.(http.Hijacker); ok {
		hooks.Hijack = func(httpsnoop.HijackFunc) httpsnoop.HijackFunc {
			return rww.Hijack
		}
	}
	if _, ok := w.(http.Pusher); ok {
		hooks.Push = func(httpsnoop.PushFunc) httpsnoop.PushFunc {
			return rww.Push
		}
	}
	if _, ok := w.(io.ReaderFrom); ok {
		hooks.ReadFrom = func(httpsnoop.ReadFromFunc) httpsnoop.ReadFromFunc {
			return rww.ReadFrom
		}
	}

	w = httpsnoop.Wrap(w, hooks)

	labeler := &Labeler{}
	ctx = injectLabeler(ctx, labeler)

	h.handler.ServeHTTP(w, r.WithContext(ctx))

	setAfterServeAttributes(span, bw.read, rww.written, rww.statusCode, bw.err, rww.err)

	// Add metrics
	attributes := append(labeler.Get(), semconv.HTTPServerMetricAttributesFromHTTPRequest(h.operation, r)...)
	counterOpts := metric.WithAttributes(attributes...)
	h.counters[RequestContentLength].Add(ctx, bw.read, counterOpts)
	h.counters[ResponseContentLength].Add(ctx, rww.written, counterOpts)

	// Use floating point division here for higher precision (instead of Millisecond method).
	elapsedTime := float64(time.Since(requestStartTime)) / float64(time.Millisecond)

	h.valueRecorders[ServerLatency].Record(ctx, elapsedTime, counterOpts)
}

func setAfterServeAttributes(span trace.Span, read, wrote int64, statusCode int, rerr, werr error) {
	attributes := []attribute.KeyValue{}

	if read > 0 {
		attributes = append(attributes, ReadBytesKey.Int64(read))
	}
	if rerr != nil && !errors.Is(rerr, io.EOF) {
		attributes = append(attributes, ReadErrorKey.String(rerr.Error()))
	}
	if wrote > 0 {
		attributes = append(attributes, WroteBytesKey.Int64(wrote))
	}
	if statusCode > 0 {
		attributes = append(attributes, semconv.HTTPAttributesFromHTTPStatusCode(statusCode)...)
		span.SetStatus(semconv.SpanStatusFromHTTPStatusCode(statusCode))
	}
	if werr != nil && !errors.Is(werr, io.EOF) {
		attributes = append(attributes, WriteErrorKey.String(werr.Error()))
	}
	span.SetAttributes(attributes...)
}

// WithRouteTag annotates a span with the provided route name using the
// RouteKey Tag.
func WithRouteTag(route string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := trace.SpanFromContext(r.Context())
		span.SetAttributes(semconv.HTTPRouteKey.String(route))
		h.ServeHTTP(w, r)
	})
}
