package gotel_test

import (
	"context"
	"testing"

	gotel "github.com/DSanches92/go-otel"
)

func sdkWithValidTracing(test *testing.T) *gotel.SDK {
	test.Helper()

	sdk, err := gotel.New(
		gotel.WithServiceName("test-service"),
		gotel.WithCollectorEndpoint("localhost:4317"),
		gotel.WithTracing(),
		gotel.WithInsecure(true),
	)
	if err != nil {
		test.Fatalf("setup: não esperado erro ao criar SDK, obtido '%s'", err)
	}

	return sdk
}

func TestSDK_New(test *testing.T) {
	test.Run("deve retornar erro quando nenhuma opção é passada", func(test *testing.T) {
		_, err := gotel.New()

		if err == nil {
			test.Error("esperado erro, obtido nil")
		}
	})

	test.Run("deve retornar erro quando config é inválida", func(test *testing.T) {
		_, err := gotel.New(
			gotel.WithServiceName("test-service"),
			// CollectorEndpoint ausente — config inválida
		)

		if err == nil {
			test.Error("esperado erro, obtido nil")
		}
	})

	test.Run("deve retornar SDK válido com config correta", func(test *testing.T) {
		sdk, err := gotel.New(
			gotel.WithServiceName("test-service"),
			gotel.WithCollectorEndpoint("localhost:4317"),
			gotel.WithTracing(),
			gotel.WithInsecure(true),
		)
		defer sdk.Shutdown(context.Background())

		if err != nil {
			test.Errorf("não esperado erro, obtido '%s'", err)
		}

		if sdk == nil {
			test.Error("esperado SDK não-nil")
		}
	})
}

func TestSDK_Shutdown(test *testing.T) {
	test.Run("deve executar sem erro com contexto válido", func(test *testing.T) {
		sdk := sdkWithValidTracing(test)

		err := sdk.Shutdown(context.Background())

		if err != nil {
			test.Errorf("não esperado erro, obtido '%s'", err)
		}
	})

	test.Run("deve retornar erro com contexto já cancelado", func(test *testing.T) {
		sdk := sdkWithValidTracing(test)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancela antes de chamar Shutdown

		err := sdk.Shutdown(ctx)

		if err == nil {
			test.Error("esperado erro com contexto cancelado, obtido nil")
		}
	})
}

func TestSDK_Tracer(test *testing.T) {
	test.Run("deve retornar tracer válido quando tracing está habilitado", func(test *testing.T) {
		sdk := sdkWithValidTracing(test)
		defer sdk.Shutdown(context.Background())

		tracer := sdk.Tracer()

		if tracer == nil {
			test.Error("esperado tracer não-nil")
		}
	})

	test.Run("deve retornar noop tracer quando tracing está desabilitado", func(test *testing.T) {
		sdk, err := gotel.New(
			gotel.WithServiceName("test-service"),
			gotel.WithCollectorEndpoint("localhost:4317"),
			gotel.WithMetrics(), // apenas metrics, sem tracing
			gotel.WithInsecure(true),
		)
		if err != nil {
			test.Fatalf("setup: não esperado erro, obtido '%s'", err)
		}
		defer sdk.Shutdown(context.Background())

		tracer := sdk.Tracer()

		if tracer == nil {
			test.Error("esperado noop tracer não-nil — nunca deve retornar nil")
		}
	})
}

func TestSDK_Metric(test *testing.T) {
	test.Run("deve retornar métrica válida quando estiver habilitado", func(test *testing.T) {
		sdk, err := gotel.New(
			gotel.WithServiceName("test-service"),
			gotel.WithCollectorEndpoint("localhost:4317"),
			gotel.WithMetrics(),
			gotel.WithInsecure(true),
		)
		if err != nil {
			test.Fatalf("setup: não esperado erro, obtido '%s'", err)
		}
		defer sdk.Shutdown(context.Background())

		metric := sdk.Metric()

		if metric == nil {
			test.Error("esperado metric não-nil")
		}
	})

	test.Run("deve retornar noop metric quando estiver desabilitado", func(test *testing.T) {
		sdk := sdkWithValidTracing(test)
		defer sdk.Shutdown(context.Background())

		metric := sdk.Metric()

		if metric == nil {
			test.Error("esperado noop metric não-nil — nunca deve retornar nil")
		}
	})
}

func TestSDK_Logger(test *testing.T) {
	test.Run("deve retornar logger válido quando logging está habilitado", func(test *testing.T) {
		sdk, err := gotel.New(
			gotel.WithServiceName("test-service"),
			gotel.WithCollectorEndpoint("localhost:4317"),
			gotel.WithLogging(),
			gotel.WithInsecure(true),
		)
		if err != nil {
			test.Fatalf("setup: não esperado erro, obtido '%s'", err)
		}
		defer sdk.Shutdown(context.Background())

		logger := sdk.Logger()

		if logger == nil {
			test.Error("esperado logger não-nil")
		}
	})

	test.Run("deve retornar noop logger quando logging está desabilitado", func(test *testing.T) {
		sdk := sdkWithValidTracing(test)
		defer sdk.Shutdown(context.Background())

		logger := sdk.Logger()

		if logger == nil {
			test.Error("esperado noop logger não-nil — nunca deve retornar nil")
		}
	})
}
