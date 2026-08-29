package gotel_test

import (
	"os"
	"testing"

	gotel "github.com/DSanches92/go-otel"
	"go.opentelemetry.io/otel/sdk/trace"
)

func TestConfig_WithEnvConfig(test *testing.T) {
	test.Run("deve ler OTEL_SERVICE_NAME da env", func(test *testing.T) {
		os.Setenv("OTEL_SERVICE_NAME", "env-service")
		defer os.Unsetenv("OTEL_SERVICE_NAME")

		config := gotel.NewConfig(
			gotel.WithEnvConfig(),
			gotel.WithCollectorEndpoint("localhost:4317"),
			gotel.WithTracing(),
		)

		if config.ServiceName != "env-service" {
			test.Errorf("esperado 'env-service', obtido '%s'", config.ServiceName)
		}
	})

	test.Run("deve ler OTEL_EXPORTER_OTLP_ENDPOINT da env", func(test *testing.T) {
		os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "collector:4317")
		defer os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")

		config := gotel.NewConfig(
			gotel.WithEnvConfig(),
			gotel.WithServiceName("test"),
			gotel.WithTracing(),
		)

		if config.CollectorEndpoint != "collector:4317" {
			test.Errorf("esperado 'collector:4317', obtido '%s'", config.CollectorEndpoint)
		}
	})

	test.Run("não deve sobrescrever valor explícito com env", func(test *testing.T) {
		os.Setenv("OTEL_SERVICE_NAME", "env-service")
		defer os.Unsetenv("OTEL_SERVICE_NAME")

		config := gotel.NewConfig(
			gotel.WithServiceName("explicit-service"),
			gotel.WithEnvConfig(),
			gotel.WithCollectorEndpoint("localhost:4317"),
			gotel.WithTracing(),
		)

		if config.ServiceName != "explicit-service" {
			test.Errorf("esperado 'explicit-service', obtido '%s'", config.ServiceName)
		}
	})

	test.Run("deve ler APP_ENV da env", func(test *testing.T) {
		os.Setenv("APP_ENV", "production")
		defer os.Unsetenv("APP_ENV")

		config := gotel.NewConfig(
			gotel.WithEnvConfig(),
			gotel.WithServiceName("test"),
			gotel.WithCollectorEndpoint("localhost:4317"),
			gotel.WithTracing(),
		)

		if config.Environment != "production" {
			test.Errorf("esperado 'production', obtido '%s'", config.Environment)
		}
	})

	test.Run("deve ler OTEL_INSECURE da env", func(test *testing.T) {
		os.Setenv("OTEL_INSECURE", "true")
		defer os.Unsetenv("OTEL_INSECURE")

		config := gotel.NewConfig(
			gotel.WithEnvConfig(),
			gotel.WithServiceName("test"),
			gotel.WithCollectorEndpoint("localhost:4317"),
			gotel.WithTracing(),
		)

		if !config.Insecure {
			test.Error("esperado Insecure=true com OTEL_INSECURE=true")
		}
	})
}

func TestConfig_WithSampler(test *testing.T) {
	test.Run("deve aceitar sampler customizado", func(test *testing.T) {
		sampler := trace.AlwaysSample()

		config := gotel.NewConfig(
			gotel.WithServiceName("test"),
			gotel.WithCollectorEndpoint("localhost:4317"),
			gotel.WithTracing(),
			gotel.WithSampler(sampler),
		)

		if config.Sampler == nil {
			test.Error("esperado sampler não-nil")
		}
	})
}
