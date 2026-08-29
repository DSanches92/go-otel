package otelhttp

import (
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const attrURLFull = attribute.Key("url.full")

type Transport struct {
	inner      http.RoundTripper
	tracer     trace.Tracer
	propagator propagation.TextMapPropagator
}

type TransportOption func(*Transport)

func WithRoundTripper(rt http.RoundTripper) TransportOption {
	return func(t *Transport) {
		t.inner = rt
	}
}

func NewTransport(provider trace.TracerProvider, opts ...TransportOption) *Transport {
	t := &Transport{
		inner:      http.DefaultTransport,
		tracer:     provider.Tracer("gotel/http/client"),
		propagator: defaultPropagator(),
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	ctx, span := t.tracer.Start(ctx, fmt.Sprintf("%s %s", req.Method, req.URL.Host))
	defer span.End()

	span.SetAttributes(
		attrHTTPMethod.String(req.Method),
		attrURLFull.String(req.URL.String()),
	)

	t.propagator.Inject(ctx, propagation.HeaderCarrier(req.Header))

	resp, err := t.inner.RoundTrip(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(
		attrHTTPStatusCode.Int(resp.StatusCode),
	)

	if resp.StatusCode >= http.StatusBadRequest {
		span.SetStatus(codes.Error, http.StatusText(resp.StatusCode))
	}

	return resp, nil
}
