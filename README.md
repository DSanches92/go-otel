# GO-OTEL

[![Go](https://img.shields.io/badge/Go-1.26.4-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Go Reference](https://img.shields.io/badge/pkg.go.dev-reference-00ADD8?logo=go&logoColor=white)](https://pkg.go.dev/github.com/DSanches92/go-otel)
[![CI](https://github.com/DSanches92/go-otel/actions/workflows/ci.yml/badge.svg)](https://github.com/DSanches92/go-otel/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/DSanches92/go-otel)](https://goreportcard.com/report/github.com/DSanches92/go-otel)

Lib de observabilidade padronizada para aplicações Go, construída sobre o [OpenTelemetry SDK](https://opentelemetry.io/docs/languages/go/).

Centraliza a inicialização de **Traces**, **Métricas** e **Logs** via OTLP/gRPC, com suporte a microsserviços NATS, aplicações HTTP e bancos de dados via `database/sql`.

```
Sua aplicação Go
      │
      │  OTLP/gRPC
      ▼
OTel Collector
      │
      ├──► Grafana Tempo   (traces)
      ├──► Prometheus      (métricas)
      └──► Grafana Loki    (logs)
```

> Exemplos de uso em cenários reais estão disponíveis em [go-otel-examples](https://github.com/DSanches92/go-otel-examples).

---

## Instalação

```bash
go get github.com/DSanches92/go-otel
```

---

## Início rápido

```go
package main

import (
    "context"
    "log"
    "log/slog"

    gotel "github.com/DSanches92/go-otel"
)

func main() {
    ctx := context.Background()

    sdk, err := gotel.New(
        gotel.WithServiceName("orders-api"),
        gotel.WithEnvironment("production"),
        gotel.WithCollectorEndpoint("otel-collector:4317"),
        gotel.WithTracing(),
        gotel.WithMetrics(),
        gotel.WithLogging(),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer sdk.Shutdown(ctx)

    // Integração com slog — trace_id e span_id em todo log
    slog.SetDefault(sdk.SlogLogger())

    tracer := sdk.Tracer()
    meter  := sdk.Meter()
    _      = meter // usado para métricas customizadas
    _      = tracer
}
```

---

## Configuração

### Opções obrigatórias

| Opção | Descrição |
|-------|-----------|
| `WithServiceName(name)` | Nome do serviço — identificador principal no Grafana |
| `WithCollectorEndpoint(host:port)` | Endereço do OpenTelemetry Collector |

### Opções opcionais

| Opção | Descrição | Default |
|-------|-----------|---------|
| `WithServiceVersion(version)` | Versão do serviço | `"0.0.0"` |
| `WithEnvironment(env)` | Ambiente de execução | `"development"` |
| `WithTimeout(duration)` | Timeout de conexão com o Collector | `5s` |
| `WithInsecure(bool)` | Desabilita TLS — apenas para desenvolvimento | `false` |
| `WithSampler(sampler)` | Amostragem de traces | `trace.AlwaysSample()` |
| `WithEnvConfig()` | Lê `OTEL_SERVICE_NAME`, `OTEL_EXPORTER_OTLP_ENDPOINT`, `APP_ENV`, `OTEL_INSECURE` do ambiente | — |

### Sinais

Habilite apenas os sinais que sua aplicação precisa:

| Opção | Destino |
|-------|---------|
| `WithTracing()` | Grafana Tempo |
| `WithMetrics()` | Prometheus |
| `WithLogging()` | Grafana Loki |

> Ao menos um sinal deve ser habilitado — `New()` retorna erro caso contrário.

### Segurança

A conexão com o Collector usa **TLS por padrão** (`Insecure = false`).
Em desenvolvimento, habilite o modo inseguro explicitamente:

```go
gotel.WithInsecure(true)
```

### Amostragem (sampling)

Use `WithSampler` para controlar o volume de traces enviados ao Collector:

```go
gotel.WithSampler(trace.TraceIDRatioBased(0.1))    // 10% dos traces
gotel.WithSampler(trace.AlwaysSample())             // 100% (default)
gotel.WithSampler(trace.NeverSample())              // nenhum
```

### Configuração via ambiente

Com `WithEnvConfig()`, as opções são lidas automaticamente de variáveis de ambiente:

```go
sdk, _ := gotel.New(
    gotel.WithEnvConfig(),
    gotel.WithTracing(),
    gotel.WithMetrics(),
    gotel.WithLogging(),
)
```

| Variável | Mapeamento |
|----------|------------|
| `OTEL_SERVICE_NAME` | `WithServiceName(...)` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `WithCollectorEndpoint(...)` |
| `APP_ENV` | `WithEnvironment(...)` |
| `OTEL_INSECURE` | `WithInsecure(true)` |

### Integração com slog

```go
slog.SetDefault(sdk.SlogLogger())

// Todo log carrega trace_id e span_id automaticamente:
slog.InfoContext(ctx, "order created", "order_id", 42)
// {"time":"...","level":"INFO","msg":"order created","order_id":42,"trace_id":"...","span_id":"..."}
```

---

## Microsserviço HTTP

O subpacote `http` fornece middleware e client instrumentation.

### Middleware (server-side)

O middleware instrumenta automaticamente cada request com spans, atributos semânticos e propagação de contexto:

- Nome do span: `"GET /orders"` (método + rota)
- Atributos: `http.request.method`, `url.path`, `http.response.status_code`, `http.route`
- Status de erro: span marcado como `Error` para status `>= 400`
- Propagação: extrai contexto W3C TraceContext de headers de entrada

```go
import httpgotel "github.com/DSanches92/go-otel/internal/http"

mux := http.NewServeMux()
mux.HandleFunc("GET /orders", handleOrders)
mux.HandleFunc("POST /orders", handleCreate)

handler := httpgotel.NewMiddleware(sdk.Tracer(),
    httpgotel.WithMeter(sdk.Meter()),
    httpgotel.WithRouteResolver(func(r *http.Request) string {
        return r.Method + " " + r.URL.Path
    }),
)(mux)

http.ListenAndServe(":8080", handler)
```

#### WithMeter — métricas HTTP

Quando `WithMeter` é fornecido, o middleware registra automaticamente:

- `http.server.request_count` — contador por method, route, status_code
- `http.server.request_duration_seconds` — latência em buckets (5ms–10s)
- `http.server.requests_in_flight` — gauge ativo (incrementa na chegada, decrementa na resposta)

### Client-side

```go
import httpgotel "github.com/DSanches92/go-otel/internal/http"

client := &http.Client{
    Transport: httpgotel.NewTransport(sdk.Tracer(),
        httpgotel.WithTransport(http.DefaultTransport),
    ),
}

// Toda chamada client.Do() cria um span filho, injeta traceparent,
// e marca erro para respostas >= 400
resp, err := client.Get("https://api.example.com/orders")
```

### Exemplo completo — API REST

```go
package main

import (
    "context"
    "log/slog"
    "net/http"
    "os/signal"
    "syscall"

    gotel "github.com/DSanches92/go-otel"
    httpgotel "github.com/DSanches92/go-otel/internal/http"
)

func main() {
    ctx, cancel := signal.NotifyContext(
        context.Background(), syscall.SIGINT, syscall.SIGTERM,
    )
    defer cancel()

    sdk, _ := gotel.New(
        gotel.WithEnvConfig(),
        gotel.WithServiceName("orders-api"),
        gotel.WithTracing(), gotel.WithMetrics(), gotel.WithLogging(),
    )
    defer sdk.Shutdown(ctx)
    slog.SetDefault(sdk.SlogLogger())

    mux := http.NewServeMux()
    mux.HandleFunc("GET /orders", handleList)
    mux.HandleFunc("POST /orders", handleCreate)

    handler := httpgotel.NewMiddleware(sdk.Tracer(),
        httpgotel.WithMeter(sdk.Meter()),
    )(mux)

    slog.InfoContext(ctx, "server starting", "addr", ":8080")
    http.ListenAndServe(":8080", handler)
}

func handleList(w http.ResponseWriter, r *http.Request) {
    // span ativo com contexto propagado
    slog.InfoContext(r.Context(), "listing orders")
    w.Write([]byte("[]"))
}

func handleCreate(w http.ResponseWriter, r *http.Request) {
    slog.InfoContext(r.Context(), "creating order")
    w.WriteHeader(http.StatusCreated)
}
```

---

## Microsserviço NATS

O subpacote `nats` fornece um middleware de alto nível que cria spans automaticamente ao publicar e consumir mensagens, além de propagar o contexto de tracing entre serviços.

### Subscribe com span automático

```go
import natsotel "github.com/DSanches92/go-otel/internal/nats"

nt := natsotel.NewTracer(sdk.Tracer())

nt.Subscribe(nc, "orders.created", func(ctx context.Context, msg *nats.Msg) {
    // span "SUBSCRIBE orders.created" criado automaticamente
    // ctx carrega o contexto propagado do publicador

    slog.InfoContext(ctx, "processing order", "subject", msg.Subject)
    //                           trace_id e span_id propagados ↑

    // Chamadas aninhadas são filhos do span atual:
    db.QueryContext(ctx, "UPDATE orders SET status = $1", "processed")
})
```

### Publicação com contexto

```go
nt.Publish(ctx, nc, "orders.shipped", data)
// "PUBLISH orders.shipped" — span criado, traceparent injetado nos headers
```

### Queue Subscribe

```go
nt.QueueSubscribe(nc, "orders.created", "workers", handler)
// span "SUBSCRIBE orders.created [workers]"
```

### Exemplo completo — Processador de eventos

```go
package main

import (
    "context"
    "log/slog"
    "os/signal"
    "syscall"

    gotel "github.com/DSanches92/go-otel"
    natsotel "github.com/DSanches92/go-otel/internal/nats"
    sqlgotel "github.com/DSanches92/go-otel/internal/sql"
    "github.com/nats-io/nats.go"
)

func main() {
    ctx, cancel := signal.NotifyContext(
        context.Background(), syscall.SIGINT, syscall.SIGTERM,
    )
    defer cancel()

    sdk, _ := gotel.New(
        gotel.WithEnvConfig(),
        gotel.WithServiceName("order-processor"),
        gotel.WithTracing(), gotel.WithMetrics(), gotel.WithLogging(),
    )
    defer sdk.Shutdown(ctx)
    slog.SetDefault(sdk.SlogLogger())

    nc, _ := nats.Connect(nats.DefaultURL)
    defer nc.Close()

    db, _ := sqlgotel.NewDatabase(
        openDB(), sdk.Tracer(),
        sqlgotel.WithDBSystem("postgresql"),
        sqlgotel.WithServerAddress("localhost", 5432),
    )

    nt := natsotel.NewTracer(sdk.Tracer())

    nt.QueueSubscribe(nc, "orders.created", "workers",
        func(ctx context.Context, msg *nats.Msg) {
            // span + contexto propagados automaticamente
            db.ExecContext(ctx, "INSERT INTO events ...")
        },
    )

    <-ctx.Done()
    slog.Info("shutting down")
}
```

### Propagação manual (nível mais baixo)

Caso precise do controle manual, o `TextMapCarrier` segue disponível:

```go
carrier := natsotel.NewCarrier(msg)
propagator.Inject(ctx, carrier)   // publicador
propagator.Extract(ctx, carrier)  // consumidor
```

---

## Banco de dados (database/sql)

O subpacote `sql` fornece um wrapper genérico sobre `database/sql` compatível
com qualquer driver — Oracle, MySQL, PostgreSQL e outros.

### O que é instrumentado automaticamente

- `sql.query` — QueryContext em DB ou Tx
- `sql.exec` — ExecContext em DB ou Tx
- `sql.transaction.begin` — BeginTx
- `sql.transaction.commit` — Tx.Commit
- `sql.transaction.rollback` — Tx.Rollback
- Atributos semânticos: `db.system`, `db.name`, `server.address`, `server.port`
- Operação extraída automaticamente (SELECT, INSERT, CREATE, WITH, CALL etc.)

### Uso

```go
import (
    "database/sql"

    _ "github.com/sijms/go-ora/v2"
    sqlgotel "github.com/DSanches92/go-otel/internal/sql"
)

sqlDB, _ := sql.Open("oracle", connString)

database, err := sqlgotel.NewDatabase(sqlDB, sdk.Tracer(),
    sqlgotel.WithDBSystem("oracle"),
    sqlgotel.WithDBName("myschema"),
    sqlgotel.WithServerAddress("oracle-host", 1521),
)

rows, err := database.QueryContext(ctx, "SELECT * FROM orders WHERE id = :1", id)
result, err := database.ExecContext(ctx, "INSERT INTO orders VALUES (:1)", id)

tx, err := database.BeginTx(ctx, nil)
defer tx.Rollback(ctx)
err = tx.Commit(ctx)
```

### Segurança

SQL e parâmetros **não são registrados por default** — habilite apenas quando necessário:

```go
sqlgotel.WithStatementRecording(true)  // registra a SQL
sqlgotel.WithParameterRecording(true)  // registra parâmetros — nunca em produção
```

---

## Estrutura do projeto

```
go-otel/
├── internal/
│   ├── http/              # Middleware HTTP, client tracing e métricas
│   │   ├── doc.go
│   │   ├── middleware.go
│   │   ├── client.go
│   │   └── metrics.go
│   ├── nats/              # TextMapCarrier e middleware com spans automáticos
│   │   ├── doc.go
│   │   ├── carrier.go
│   │   └── middleware.go
│   └── sql/               # Wrapper database/sql com spans automáticos
│       ├── doc.go
│       ├── database.go
│       └── transaction.go
│
├── test/
│   ├── http/
│   │   ├── main_test.go
│   │   ├── middleware_test.go
│   │   └── client_test.go
│   ├── nats/
│   │   ├── main_test.go
│   │   ├── carrier_test.go
│   │   └── middleware_test.go
│   ├── sql/
│   │   ├── main_test.go
│   │   └── database_test.go
│   ├── main_test.go
│   ├── otel_config_test.go
│   └── otel_config_env_test.go
│
├── doc.go                 # Documentação do pacote raiz
├── go.mod
├── go.sum
├── otel_config.go         # Configuração e functional options
├── otel_provider.go       # Inicialização dos providers OTel via OTLP/gRPC
├── otel_sdk.go            # Ponto de entrada — New() e Shutdown()
└── README.md
```

---

## Executando os testes

```bash
# Todos os testes
go test ./test/... -count=1 -v

# Apenas um pacote
go test ./test/http/... -count=1 -v
go test ./test/nats/... -count=1 -v
go test ./test/sql/... -count=1 -v
go test . -count=1 -v
```

---

## Variáveis de ambiente

| Variável | Descrição | Default |
|----------|-----------|---------|
| `OTEL_SERVICE_NAME` | Nome do serviço | — |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Endereço do Collector | `localhost:4317` |
| `OTEL_INSECURE` | Desabilita TLS | `false` |
| `APP_ENV` | Ambiente (`development`, `production`) | `development` |
| `NATS_URL` | URL do servidor NATS | `nats://localhost:4222` |
| `HTTP_ADDR` | Endereço do servidor HTTP | `:8080` |

---

## Dependências principais

| Pacote | Versão | Uso |
|--------|--------|-----|
| `go.opentelemetry.io/otel` | v1.44.0 | SDK base |
| `go.opentelemetry.io/otel/sdk` | v1.44.0 | Providers |
| `go.opentelemetry.io/otel/exporters/otlp/...` | v1.44.0 | Exporters OTLP/gRPC |
| `github.com/nats-io/nats.go` | v1.52.0 | Cliente NATS |
| `google.golang.org/grpc` | v1.81.1 | Transporte gRPC |

---

<p align="center">
  Criado com ❤️ por <a href="https://github.com/DSanches92">Danilo Sanches</a>
</p>
