// Package gotel fornece uma camada de observabilidade padronizada para
// aplicações Go, construída sobre o OpenTelemetry SDK.
//
// A lib centraliza a inicialização de Traces, Métricas e Logs via OTLP/gRPC,
// exportando para um OpenTelemetry Collector que repassa os dados para:
//   - Grafana Tempo  (traces)
//   - Prometheus     (métricas)
//   - Grafana Loki   (logs)
//
// # Uso básico
//
// Inicialize o SDK no ponto de entrada da aplicação e registre o shutdown:
//
//	sdk, err := gotel.New(
//	    gotel.WithServiceName("orders-ms"),
//	    gotel.WithCollectorEndpoint("otel-collector:4317"),
//	    gotel.WithServiceVersion("1.0.0"),
//	    gotel.WithEnvironment("production"),
//	    gotel.WithTracing(),
//	    gotel.WithMetrics(),
//	    gotel.WithLogging(),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer sdk.Shutdown(context.Background())
//
// # Sinais disponíveis
//
// Os sinais são opcionais e independentes — habilite apenas o que precisar:
//
//	gotel.WithTracing()  // exporta traces para o Grafana Tempo
//	gotel.WithMetrics()  // exporta métricas para o Prometheus
//	gotel.WithLogging()  // exporta logs para o Grafana Loki
//
// # Segurança
//
// Por padrão, a conexão com o Collector usa TLS (Insecure=false).
// Em ambiente de desenvolvimento, habilite o modo inseguro explicitamente:
//
//	gotel.WithInsecure(true)
//
// # Subpacotes
//
// O pacote gotel é complementado por subpacotes de instrumentação:
//
//   - [github.com/DSanches92/go-otel/nats] — propagação de contexto via NATS headers
//   - [github.com/DSanches92/go-otel/http]  — middleware HTTP com spans automáticos
package gotel
