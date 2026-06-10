package gotel

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/noop"
	"go.opentelemetry.io/otel/metric"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
)

// ---- SDK

type SDK struct {
	config            *Config
	tracerProvider    trace.TracerProvider
	metricProvider    metric.MeterProvider
	loggerProvider    log.LoggerProvider
	shutdownFunctions []shutdownFunc
}

type shutdownFunc func(context.Context) error

// ---- Construtor

func New(options ...Option) (*SDK, error) {
	config := NewConfig(options...)

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("gotel.New: %w", err)
	}

	sdk := &SDK{
		config:         config,
		tracerProvider: nooptrace.NewTracerProvider(),
		metricProvider: noopmetric.NewMeterProvider(),
		loggerProvider: noop.NewLoggerProvider(),
	}

	if err := sdk.initProviders(); err != nil {
		return nil, fmt.Errorf("gotel.New: falha ao inicializar providers: %w", err)
	}

	return sdk, nil
}

// ---- Shutdown

func (sdk *SDK) Shutdown(ctx context.Context) error {
	var errs []error

	for _, function := range sdk.shutdownFunctions {
		if err := function(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// --- Sinais

func (sdk *SDK) Tracer() trace.Tracer {
	return sdk.tracerProvider.Tracer(sdk.config.ServiceName)
}

func (sdk *SDK) Metric() metric.Meter {
	return sdk.metricProvider.Meter(sdk.config.ServiceName)
}

func (sdk *SDK) Logger() log.Logger {
	return sdk.loggerProvider.Logger(sdk.config.ServiceName)
}

// ---- Inicialização Interna

func (sdk *SDK) initProviders() error {
	if sdk.config.TracingEnabled {
		if err := sdk.initTracerProvider(); err != nil {
			return fmt.Errorf("tracer provider: %w", err)
		}
	}

	if sdk.config.MetricsEnabled {
		if err := sdk.initMetricProvider(); err != nil {
			return fmt.Errorf("metric provider: %w", err)
		}
	}

	if sdk.config.LoggingEnabled {
		if err := sdk.initLoggerProvider(); err != nil {
			return fmt.Errorf("logger provider: %w", err)
		}
	}

	return nil
}

// --- Providers

func (sdk *SDK) TracerProvider() trace.TracerProvider {
	return sdk.tracerProvider
}
