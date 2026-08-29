package otelhttp

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	attrHTTPMethod     = attribute.Key("http.request.method")
	attrURLPath        = attribute.Key("url.path")
	attrHTTPStatusCode = attribute.Key("http.response.status_code")
	attrHTTPRoute      = attribute.Key("http.route")
)

const statusErroBoundary = http.StatusBadRequest

var defaultPropagator = sync.OnceValue(func() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
})

type config struct {
	routeResolver func(*http.Request) string
	meter         metric.Meter
}

type Option func(*config)

func WithRouteResolver(fn func(*http.Request) string) Option {
	return func(c *config) {
		c.routeResolver = fn
	}
}

func WithMeter(meter metric.Meter) Option {
	return func(c *config) {
		c.meter = meter
	}
}

func NewMiddleware(provider trace.TracerProvider, opts ...Option) func(http.Handler) http.Handler {
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}

	tracer := provider.Tracer("gotel/http")
	propagator := defaultPropagator()

	var metrics *Metrics
	if cfg.meter != nil {
		var err error
		metrics, err = NewMetrics(cfg.meter)
		if err != nil {
			metrics = nil
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
			ctx := propagator.Extract(req.Context(), propagation.HeaderCarrier(req.Header))

			route := req.URL.Path
			if cfg.routeResolver != nil {
				route = cfg.routeResolver(req)
			}

			spanName := fmt.Sprintf("%s %s", req.Method, route)
			ctx, span := tracer.Start(ctx, spanName)
			defer span.End()

			span.SetAttributes(
				attrHTTPMethod.String(req.Method),
				attrURLPath.String(req.URL.Path),
				attrHTTPRoute.String(route),
			)

			resWriter := newResponseWriter(writer)

			var start time.Time
			if metrics != nil {
				metrics.InFlight.Add(ctx, 1)
				start = time.Now()
			}

			next.ServeHTTP(resWriter, req.WithContext(ctx))

			if metrics != nil {
				metrics.InFlight.Add(ctx, -1)
				duration := time.Since(start).Seconds()

				attrs := metric.WithAttributes(
					attrHTTPMethod.String(req.Method),
					attrHTTPStatusCode.Int(resWriter.status),
					attrHTTPRoute.String(route),
				)

				metrics.RequestCount.Add(ctx, 1, attrs)
				metrics.RequestDuration.Record(ctx, duration, attrs)
			}

			span.SetAttributes(
				attrHTTPStatusCode.Int(resWriter.status),
			)

			if resWriter.status >= statusErroBoundary {
				span.SetStatus(codes.Error, http.StatusText(resWriter.status))
			}
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func newResponseWriter(writer http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: writer,
		status:         http.StatusOK,
	}
}

func (resWriter *responseWriter) WriteHeader(status int) {
	resWriter.status = status
	resWriter.ResponseWriter.WriteHeader(status)
}
