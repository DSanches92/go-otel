package http

import (
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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

// ---- Middleware

func NewMiddleware(provider trace.TracerProvider) func(http.Handler) http.Handler {
	tracer := provider.Tracer("gotel/http")
	propagator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
			ctx := propagator.Extract(req.Context(), propagation.HeaderCarrier(req.Header))

			spanName := fmt.Sprintf("%s %s", req.Method, req.URL.Path)
			ctx, span := tracer.Start(ctx, spanName)
			defer span.End()

			span.SetAttributes(
				attrHTTPMethod.String(req.Method),
				attrURLPath.String(req.URL.Path),
			)

			resWriter := newResponseWriter(writer)
			next.ServeHTTP(resWriter, req.WithContext(ctx))

			span.SetAttributes(
				attrHTTPStatusCode.Int(resWriter.status),
			)

			if resWriter.status >= statusErroBoundary {
				span.SetStatus(codes.Error, http.StatusText(resWriter.status))
			}
		})
	}
}

// ---- ResponseWriter

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
