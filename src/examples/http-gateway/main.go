// Exemplo de uso da lib gotel em um API Gateway HTTP.
//
// Demonstra como inicializar o SDK e aplicar o middleware de instrumentação
// em um servidor HTTP padrão.
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	gotel "github.com/DSanches92/go-otel"
	httpgotel "github.com/DSanches92/go-otel/src/instrumentation/http"
)

func main() {
	// ---- Inicialização do SDK
	sdk, err := gotel.New(
		gotel.WithServiceName("api-gateway"),
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

	// ---- Rotas
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("GET /orders", handleListOrders)
	mux.HandleFunc("POST /orders", handleCreateOrder)

	// ---- Middleware de Instrumentação
	// O middleware envolve todo o mux — instrumenta todas as rotas de uma vez,
	// sem precisar adicionar tracing individualmente em cada handler.
	handler := httpgotel.NewMiddleware(sdk.TracerProvider())(mux)

	// ---- Servidor HTTP com graceful shutdown
	srv := &http.Server{
		Addr:    envOrDefault("HTTP_ADDR", ":8080"),
		Handler: handler,
	}

	go func() {
		log.Printf("api-gateway ouvindo em %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("falha ao iniciar servidor: %v", err)
		}
	}()

	waitClosing()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("falha no graceful shutdown: %v", err)
	}

	log.Println("api-gateway encerrado")
}

// ---- Handlers

func handleHealth(writer http.ResponseWriter, req *http.Request) {
	respondJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
}

func handleListOrders(writer http.ResponseWriter, req *http.Request) {
	// O span já foi criado pelo middleware com "GET /orders".
	// Aqui você pode criar spans filhos para operações específicas,
	// como consultas ao banco de dados.
	respondJSON(writer, http.StatusOK, map[string]any{
		"orders": []any{},
	})
}

func handleCreateOrder(writer http.ResponseWriter, req *http.Request) {
	respondJSON(writer, http.StatusCreated, map[string]string{
		"id":     "ord_123",
		"status": "created",
	})
}

// ---- Helpers

// respondJSON serializa o payload como JSON e escreve na response.
func respondJSON(writer http.ResponseWriter, status int, payload any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	json.NewEncoder(writer).Encode(payload)
}

// envOrDefault retorna o valor da variável de ambiente ou o valor default fornecido.
func envOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// aguardarEncerramento bloqueia até receber SIGINT ou SIGTERM.
func waitClosing() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("encerrando...")
}
