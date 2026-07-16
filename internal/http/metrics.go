package http

import (
	"go.opentelemetry.io/otel/metric"
)

type Metrics struct {
	RequestCount    metric.Int64Counter
	RequestDuration metric.Float64Histogram
	InFlight        metric.Int64UpDownCounter
}

func NewMetrics(meter metric.Meter) (*Metrics, error) {
	reqCount, err := meter.Int64Counter(
		"http.server.request_count",
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, err
	}

	reqDuration, err := meter.Float64Histogram(
		"http.server.request_duration_seconds",
		metric.WithDescription("Duration of HTTP requests"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(
			0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10,
		),
	)
	if err != nil {
		return nil, err
	}

	inFlight, err := meter.Int64UpDownCounter(
		"http.server.requests_in_flight",
		metric.WithDescription("Number of HTTP requests currently in flight"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, err
	}

	return &Metrics{
		RequestCount:    reqCount,
		RequestDuration: reqDuration,
		InFlight:        inFlight,
	}, nil
}
