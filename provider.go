package gotel

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// ---- Resource

func (sdk *SDK) newResource() (*resource.Resource, error) {
	return resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(sdk.config.ServiceName),
			semconv.ServiceVersion(sdk.config.ServiceVersion),
			semconv.DeploymentEnvironment(sdk.config.Environment),
		),
	)
}

// ---- Conexão gRPC

func (sdk *SDK) newGRPCConnection() (*grpc.ClientConn, error) {
	var transportCredentials credentials.TransportCredentials

	if sdk.config.Insecure {
		transportCredentials = insecure.NewCredentials()
	} else {
		transportCredentials = credentials.NewTLS(nil)
	}

	conn, err := grpc.NewClient(
		sdk.config.CollectorEndpoint,
		grpc.WithTransportCredentials(transportCredentials),
	)
	if err != nil {
		return nil, fmt.Errorf("falha ao conectar ao collector '%s': %w", sdk.config.CollectorEndpoint, err)
	}

	return conn, nil
}

// ---- Tracer Provider

func (sdk *SDK) initTracerProvider() error {
	conn, err := sdk.newGRPCConnection()
	if err != nil {
		return err
	}

	res, err := sdk.newResource()
	if err != nil {
		return fmt.Errorf("falha ao criar resource: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), sdk.config.Timeout)
	defer cancel()

	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return fmt.Errorf("falha ao criar trace exporter: %w", err)
	}

	provider := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
	)

	sdk.tracerProvider = provider
	sdk.shutdownFunctions = append(sdk.shutdownFunctions, provider.Shutdown)

	return nil
}

// ---- Metric Provider

func (sdk *SDK) initMetricProvider() error {
	conn, err := sdk.newGRPCConnection()
	if err != nil {
		return err
	}

	res, err := sdk.newResource()
	if err != nil {
		return fmt.Errorf("falha ao criar resource: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), sdk.config.Timeout)
	defer cancel()

	exporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithGRPCConn(conn))
	if err != nil {
		return fmt.Errorf("falha ao criar metric exporter: %w", err)
	}

	provider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(exporter)),
		metric.WithResource(res),
	)

	sdk.metricProvider = provider
	sdk.shutdownFunctions = append(sdk.shutdownFunctions, provider.Shutdown)

	return nil
}

// ---- Logger Provider

func (sdk *SDK) initLoggerProvider() error {
	conn, err := sdk.newGRPCConnection()
	if err != nil {
		return err
	}

	res, err := sdk.newResource()
	if err != nil {
		return fmt.Errorf("falha ao criar resource: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), sdk.config.Timeout)
	defer cancel()

	exporter, err := otlploggrpc.New(ctx, otlploggrpc.WithGRPCConn(conn))
	if err != nil {
		return fmt.Errorf("falha ao criar log exporter: %w", err)
	}

	provider := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(exporter)),
		log.WithResource(res),
	)

	sdk.loggerProvider = provider
	sdk.shutdownFunctions = append(sdk.shutdownFunctions, provider.Shutdown)

	return nil
}
