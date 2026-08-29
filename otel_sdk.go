package gotel

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/noop"
	"go.opentelemetry.io/otel/metric"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc"
)

// ---- SDK

type SDK struct {
	config            *Config
	tracerProvider    trace.TracerProvider
	metricProvider    metric.MeterProvider
	loggerProvider    log.LoggerProvider
	conn              *grpc.ClientConn
	shutdownFunctions []shutdownFunc

	slogLoggerOnce sync.Once
	slogLogger     *slog.Logger
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
		shutdownCtx, cancel := context.WithTimeout(context.Background(), sdk.config.Timeout)
		defer cancel()

		if shutdownErr := sdk.Shutdown(shutdownCtx); shutdownErr != nil {
			return nil, fmt.Errorf("gotel.New: falha ao inicializar providers: %w (cleanup também falhou: %v)", err, shutdownErr)
		}

		return nil, fmt.Errorf("gotel.New: falha ao inicializar providers: %w", err)
	}

	return sdk, nil
}

// ---- Shutdown

func (sdk *SDK) Shutdown(ctx context.Context) error {
	var errs []error

	for _, fn := range sdk.shutdownFunctions {
		if err := fn(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if sdk.conn != nil {
		if err := sdk.conn.Close(); err != nil {
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

func (sdk *SDK) SlogLogger() *slog.Logger {
	sdk.slogLoggerOnce.Do(func() {
		sdk.slogLogger = otelslog.NewLogger(
			sdk.config.ServiceName,
			otelslog.WithLoggerProvider(sdk.loggerProvider),
			otelslog.WithVersion(sdk.config.ServiceVersion),
		)
	})

	return sdk.slogLogger
}

// ---- Inicialização Interna

func (sdk *SDK) initProviders() error {
	if !sdk.config.TracingEnabled && !sdk.config.MetricsEnabled && !sdk.config.LoggingEnabled {
		return nil
	}

	conn, err := sdk.newGRPCConnection()
	if err != nil {
		return err
	}
	sdk.conn = conn

	if sdk.config.TracingEnabled {
		if err := sdk.initTracerProvider(conn); err != nil {
			return fmt.Errorf("tracer provider: %w", err)
		}
	}

	if sdk.config.MetricsEnabled {
		if err := sdk.initMetricProvider(conn); err != nil {
			return fmt.Errorf("metric provider: %w", err)
		}
	}

	if sdk.config.LoggingEnabled {
		if err := sdk.initLoggerProvider(conn); err != nil {
			return fmt.Errorf("logger provider: %w", err)
		}
	}

	return nil
}

// --- Providers

func (sdk *SDK) TracerProvider() trace.TracerProvider {
	return sdk.tracerProvider
}

func (sdk *SDK) MeterProvider() metric.MeterProvider {
	return sdk.metricProvider
}

func (sdk *SDK) LoggerProvider() log.LoggerProvider {
	return sdk.loggerProvider
}
