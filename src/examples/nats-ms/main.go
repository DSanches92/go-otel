// Exemplo de uso da lib gotel em um microsserviço NATS.
//
// Demonstra como inicializar o SDK, propagar contexto entre mensagens
// e criar spans para operações de negócio.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	gotel "github.com/DSanches92/go-otel"
	natsotel "github.com/DSanches92/go-otel/src/instrumentation/nats"
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func main() {
	// ---- Inicialização do SDK
	sdk, err := gotel.New(
		gotel.WithServiceName("orders-ms"),
		gotel.WithCollectorEndpoint(envOrDefault("OTEL_COLLECTOR_ENDPOINT", "localhost:4317")),
		gotel.WithServiceVersion("1.0.0"),
		gotel.WithEnvironment(envOrDefault("APP_ENV", "development")),
		gotel.WithInsecure(envOrDefault("APP_ENV", "development") == "development"),
		gotel.WithTracing(),
		gotel.WithMetrics(),
		gotel.WithLogging(),
	)

	if err != nil {
		log.Fatalf("falha ao inicializar gotel: %v", err)
	}
	defer sdk.Shutdown(context.Background())

	tracer := sdk.Tracer()
	propagator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)

	// ---- Conexão NATS
	natsConn, err := nats.Connect(envOrDefault("NATS_URL", nats.DefaultURL))
	if err != nil {
		log.Fatalf("falha ao conectar ao NATS: %v", err)
	}
	defer natsConn.Drain()

	// ---- Consumidor: extrai contexto e cria span filho
	_, err = natsConn.Subscribe("orders.created", func(msg *nats.Msg) {
		// Extrai o contexto de trace do header da mensagem recebida.
		// Se o publicador injetou um trace, este span será filho dele.
		carrier := natsotel.NewCarrier(msg)
		ctx := propagator.Extract(context.Background(), carrier)

		ctx, span := tracer.Start(ctx, "orders.created.processar")
		defer span.End()

		log.Printf("processando pedido — traceID: %s", span.SpanContext().TraceID())

		// Simula o processamento e publica um evento downstream
		if err := publishConfirmedOrder(ctx, natsConn, tracer, propagator); err != nil {
			log.Printf("erro ao confirmar pedido: %v", err)
		}
	})

	if err != nil {
		log.Fatalf("falha ao subscrever: %v", err)
	}

	log.Println("orders-ms aguardando mensagens...")
	waitClosing()
}

// publishConfirmedOrder publica um evento downstream propagando o contexto de trace.
func publishConfirmedOrder(
	ctx context.Context,
	natsConn *nats.Conn,
	tracer trace.Tracer,
	propagator propagation.TextMapPropagator,
) error {
	ctx, span := tracer.Start(ctx, "orders.confirmed.publicar")
	defer span.End()

	// Injeta o contexto de trace no header da mensagem de saída.
	// O consumidor downstream poderá continuar o trace como span filho.
	msg := &nats.Msg{Subject: "orders.confirmed"}
	carrier := natsotel.NewCarrier(msg)
	propagator.Inject(ctx, carrier)

	return natsConn.PublishMsg(msg)
}

// envOrDefault retorna o valor da variável de ambiente ou o valor default fornecido.
func envOrDefault(chave, valorDefault string) string {
	if valor := os.Getenv(chave); valor != "" {
		return valor
	}
	return valorDefault
}

// waitClosing bloqueia até receber SIGINT ou SIGTERM.
func waitClosing() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("encerrando...")
}
