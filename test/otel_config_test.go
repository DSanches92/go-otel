package gotel_test

import (
	"testing"
	"time"

	gotel "github.com/DSanches92/go-otel"
)

func TestConfig_Defaults(test *testing.T) {
	config := gotel.NewConfig(
		gotel.WithServiceName("my-service"),
		gotel.WithCollectorEndpoint("localhost:4317"),
	)

	test.Run("ServiceVersion deve ser 0.0.0", func(test *testing.T) {
		if config.ServiceVersion != "0.0.0" {
			test.Errorf("esperado '0.0.0', obtido '%s'", config.ServiceVersion)
		}
	})

	test.Run("Environment deve ser development", func(test *testing.T) {
		if config.Environment != "development" {
			test.Errorf("esperado 'development', obtido '%s'", config.Environment)
		}
	})

	test.Run("Timeout deve ser 5s", func(test *testing.T) {
		if config.Timeout != 5*time.Second {
			test.Errorf("esperado '5s', obtido '%s'", config.Timeout)
		}
	})

	test.Run("Insecure deve ser false por default", func(test *testing.T) {
		if config.Insecure {
			test.Error("esperado 'false' — secure by default")
		}
	})
}

func TestConfig_Signals_AllDisabledByDefault(test *testing.T) {
	config := gotel.NewConfig()

	test.Run("Tracing deve estar desabilitado", func(test *testing.T) {
		if config.TracingEnabled {
			test.Error("esperado 'false'")
		}
	})

	test.Run("Metrics deve estar desabilitado", func(test *testing.T) {
		if config.MetricsEnabled {
			test.Error("esperado 'false'")
		}
	})

	test.Run("Logging deve estar desabilitado", func(test *testing.T) {
		if config.LoggingEnabled {
			test.Error("esperado 'false'")
		}
	})
}

func TestConfig_RequiredOptions(test *testing.T) {
	test.Run("WithServiceName deve setar o nome do serviço", func(test *testing.T) {
		config := gotel.NewConfig(
			gotel.WithServiceName("orders-ms"),
		)

		if config.ServiceName != "orders-ms" {
			test.Errorf("esperado 'orders-ms', obtido '%s'", config.ServiceName)
		}
	})

	test.Run("WithCollectorEndpoint deve setar o endpoint do collector", func(test *testing.T) {
		config := gotel.NewConfig(
			gotel.WithCollectorEndpoint("otel-collector:4317"),
		)

		if config.CollectorEndpoint != "otel-collector:4317" {
			test.Errorf("esperado 'otel-collector:4317', obtido '%s'", config.CollectorEndpoint)
		}
	})
}

func TestConfig_OptionalOptions(test *testing.T) {
	test.Run("WithServiceVersion deve setar a versão do serviço", func(test *testing.T) {
		config := gotel.NewConfig(
			gotel.WithServiceVersion("1.2.3"),
		)

		if config.ServiceVersion != "1.2.3" {
			test.Errorf("esperado '1.2.3', obtido '%s'", config.ServiceVersion)
		}
	})

	test.Run("WithEnvironment deve setar o ambiente", func(test *testing.T) {
		config := gotel.NewConfig(
			gotel.WithEnvironment("production"),
		)

		if config.Environment != "production" {
			test.Errorf("esperado 'production', obtido '%s'", config.Environment)
		}
	})

	test.Run("WithTimeout deve setar o timeout", func(test *testing.T) {
		config := gotel.NewConfig(
			gotel.WithTimeout(10 * time.Second),
		)

		if config.Timeout != 10*time.Second {
			test.Errorf("esperado '10s', obtido '%s'", config.Timeout)
		}
	})

	test.Run("WithInsecure true deve habilitar o modo inseguro", func(test *testing.T) {
		config := gotel.NewConfig(
			gotel.WithInsecure(true),
		)

		if !config.Insecure {
			test.Error("esperado 'true' após WithInsecure(true)")
		}
	})
}

func TestConfig_Signals(test *testing.T) {
	test.Run("WithTracing deve habilitar tracing", func(test *testing.T) {
		config := gotel.NewConfig(gotel.WithTracing())

		if !config.TracingEnabled {
			test.Error("esperado 'true'")
		}
	})

	test.Run("WithMetrics deve habilitar metrics", func(test *testing.T) {
		config := gotel.NewConfig(gotel.WithMetrics())

		if !config.MetricsEnabled {
			test.Error("esperado 'true'")
		}
	})

	test.Run("WithLogging deve habilitar logging", func(test *testing.T) {
		config := gotel.NewConfig(gotel.WithLogging())

		if !config.LoggingEnabled {
			test.Error("esperado 'true'")
		}
	})
}

func TestConfig_Validate(test *testing.T) {
	test.Run("deve retornar erro quando ServiceName está ausente", func(test *testing.T) {
		config := gotel.NewConfig(
			gotel.WithCollectorEndpoint("localhost:4317"),
			gotel.WithTracing(),
		)

		if err := config.Validate(); err == nil {
			test.Error("esperado erro, obtido nil")
		}
	})

	test.Run("deve retornar erro quando CollectorEndpoint está ausente", func(test *testing.T) {
		config := gotel.NewConfig(
			gotel.WithServiceName("my-service"),
			gotel.WithTracing(),
		)

		if err := config.Validate(); err == nil {
			test.Error("esperado erro, obtido nil")
		}
	})

	test.Run("deve retornar erro quando nenhum sinal está habilitado", func(test *testing.T) {
		config := gotel.NewConfig(
			gotel.WithServiceName("my-service"),
			gotel.WithCollectorEndpoint("localhost:4317"),
		)

		if err := config.Validate(); err == nil {
			test.Error("esperado erro, obtido nil")
		}
	})

	test.Run("deve ser válido com campos obrigatórios e ao menos um sinal", func(test *testing.T) {
		config := gotel.NewConfig(
			gotel.WithServiceName("my-service"),
			gotel.WithCollectorEndpoint("localhost:4317"),
			gotel.WithTracing(),
		)

		if err := config.Validate(); err != nil {
			test.Errorf("não esperado erro, obtido '%s'", err)
		}
	})
}
