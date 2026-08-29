# GO-OTEL

[![Go](https://img.shields.io/badge/Go-1.26.4-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Go Reference](https://img.shields.io/badge/pkg.go.dev-reference-00ADD8?logo=go&logoColor=white)](https://pkg.go.dev/github.com/DSanches92/go-otel)
[![CI](https://github.com/DSanches92/go-otel/actions/workflows/ci.yml/badge.svg)](https://github.com/DSanches92/go-otel/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/DSanches92/go-otel)](https://goreportcard.com/report/github.com/DSanches92/go-otel)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

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

## Requisitos

- **Go 1.26+** (versão mínima definida em `go.mod`).
- Um **OpenTelemetry Collector** acessível pela aplicação, recebendo OTLP/gRPC
  (porta padrão `4317`) e roteando para os backends que você usa (Tempo,
  Prometheus, Loki, ou equivalentes). A lib fala apenas com o Collector — ela
  não exporta diretamente para os backends.

Para rodar um Collector localmente durante o desenvolvimento, um `docker-compose.yml` mínimo:

```yaml
services:
  otel-collector:
    image: otel/opentelemetry-collector-contrib:latest
    command: ["--config=/etc/otel-collector.yaml"]
    volumes:
      - ./otel-collector.yaml:/etc/otel-collector.yaml
    ports:
      - "4317:4317" # OTLP/gRPC
```

> Exemplos completos de configuração do Collector (`otel-collector.yaml`) e docker-compose com Tempo/Prometheus/Loki estão em [go-otel-examples](https://github.com/DSanches92/go-otel-examples).

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
    meter  := sdk.Metric()
    _      = meter // usado para métricas customizadas
    _      = tracer
}
```

> Além dos acessores de sinal (`Tracer()`, `Metric()`, `Logger()`, `SlogLogger()`), o SDK também expõe os providers "crus" — `TracerProvider()`, `MeterProvider()` e `LoggerProvider()` — úteis para passar a outras libs que esperem um `trace.TracerProvider`/`metric.MeterProvider`/`log.LoggerProvider` diretamente.

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

O subpacote `otelhttp` fornece middleware e client instrumentation.

### Middleware (server-side)

O middleware instrumenta automaticamente cada request com spans, atributos semânticos e propagação de contexto:

- Nome do span: `"GET /orders"` (método + rota)
- Atributos: `http.request.method`, `url.path`, `http.response.status_code`, `http.route`
- Status de erro: span marcado como `Error` para status `>= 400`
- Propagação: extrai contexto W3C TraceContext de headers de entrada

```go
import "github.com/DSanches92/go-otel/otelhttp"

mux := http.NewServeMux()
mux.HandleFunc("GET /orders", handleOrders)
mux.HandleFunc("POST /orders", handleCreate)

middleware, err := otelhttp.NewMiddleware(sdk.Tracer(),
    otelhttp.WithMeter(sdk.Metric()),
    otelhttp.WithRouteResolver(func(r *http.Request) string {
        return r.Method + " " + r.URL.Path
    }),
)
if err != nil {
    log.Fatal(err)
}

http.ListenAndServe(":8080", middleware(mux))
```

> `NewMiddleware` retorna erro quando `WithMeter` é usado e a criação dos instrumentos de métrica falha — trate-o na inicialização, não em runtime.

#### WithMeter — métricas HTTP

Quando `WithMeter` é fornecido, o middleware registra automaticamente:

- `http.server.request_count` — contador por method, route, status_code
- `http.server.request_duration_seconds` — latência em buckets (5ms–10s)
- `http.server.requests_in_flight` — gauge ativo (incrementa na chegada, decrementa na resposta)

### Client-side

```go
import "github.com/DSanches92/go-otel/otelhttp"

client := &http.Client{
    Transport: otelhttp.NewTransport(sdk.Tracer(),
        otelhttp.WithRoundTripper(http.DefaultTransport),
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
    "log"
    "log/slog"
    "net/http"
    "os/signal"
    "syscall"

    gotel "github.com/DSanches92/go-otel"
    "github.com/DSanches92/go-otel/otelhttp"
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

    middleware, err := otelhttp.NewMiddleware(sdk.Tracer(),
        otelhttp.WithMeter(sdk.Metric()),
    )
    if err != nil {
        log.Fatal(err)
    }

    slog.InfoContext(ctx, "server starting", "addr", ":8080")
    http.ListenAndServe(":8080", middleware(mux))
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

O subpacote `otelnats` fornece um middleware de alto nível que cria spans automaticamente ao publicar e consumir mensagens, além de propagar o contexto de tracing entre serviços.

### Subscribe com span automático

```go
import "github.com/DSanches92/go-otel/otelnats"

nt := otelnats.NewTracer(sdk.Tracer())

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
    "github.com/DSanches92/go-otel/otelnats"
    "github.com/DSanches92/go-otel/otelsql"
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

    db, _ := otelsql.NewDB(
        openDB(), sdk.Tracer(),
        otelsql.WithDBSystem("postgresql"),
        otelsql.WithServerAddress("localhost", 5432),
    )

    nt := otelnats.NewTracer(sdk.Tracer())

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
carrier := otelnats.NewCarrier(msg)
propagator.Inject(ctx, carrier)   // publicador
propagator.Extract(ctx, carrier)  // consumidor
```

---

## Banco de dados (database/sql)

O subpacote `otelsql` fornece um wrapper genérico sobre `database/sql` compatível
com qualquer driver — Oracle, MySQL, PostgreSQL e outros.

### O que é instrumentado automaticamente

- `sql.query` — QueryContext/QueryRowContext em Database ou Transaction
- `sql.exec` — ExecContext em Database ou Transaction
- `sql.transaction.begin` — BeginTx
- `sql.transaction.commit` — Transaction.Commit
- `sql.transaction.rollback` — Transaction.Rollback
- Atributos semânticos: `db.system`, `db.name`, `server.address`, `server.port`
- Operação extraída automaticamente (SELECT, INSERT, UPDATE, DELETE, WITH, CALL, MERGE, REPLACE, TRUNCATE, CREATE, ALTER, DROP)

### Uso

```go
import (
    "database/sql"

    _ "github.com/sijms/go-ora/v2"
    "github.com/DSanches92/go-otel/otelsql"
)

sqlDB, _ := sql.Open("oracle", connString)

database, err := otelsql.NewDB(sqlDB, sdk.Tracer(),
    otelsql.WithDBSystem("oracle"),
    otelsql.WithDBName("myschema"),
    otelsql.WithServerAddress("oracle-host", 1521),
)

rows, err := database.QueryContext(ctx, "SELECT * FROM orders WHERE id = :1", id)
row, err := database.QueryRowContext(ctx, "SELECT name FROM orders WHERE id = :1", id)
result, err := database.ExecContext(ctx, "INSERT INTO orders VALUES (:1)", id)

tx, err := database.BeginTx(ctx, nil)
defer tx.Rollback(ctx)
err = tx.Commit(ctx)
```

### Segurança

SQL e parâmetros **não são registrados por default** — habilite apenas quando necessário:

```go
otelsql.WithStatementRecording(true)  // registra a SQL
otelsql.WithParameterRecording(true)  // registra parâmetros — nunca em produção
```

---

## Estrutura do projeto

```
go-otel/
├── otelhttp/               # Middleware HTTP, client tracing e métricas
│   ├── doc.go
│   ├── middleware.go
│   ├── client.go
│   └── metrics.go
├── otelnats/               # TextMapCarrier e middleware com spans automáticos
│   ├── doc.go
│   ├── carrier.go
│   └── middleware.go
├── otelsql/                # Wrapper database/sql com spans automáticos
│   ├── doc.go
│   ├── database.go
│   └── transaction.go
│
├── test/
│   ├── otelhttp/
│   │   ├── main_test.go
│   │   ├── middleware_test.go
│   │   └── client_test.go
│   ├── otelnats/
│   │   ├── main_test.go
│   │   ├── carrier_test.go
│   │   └── middleware_test.go
│   ├── otelsql/
│   │   ├── main_test.go
│   │   └── database_test.go
│   ├── main_test.go
│   ├── otel_config_test.go
│   ├── otel_config_env_test.go
│   └── otel_sdk_test.go
│
├── doc.go                  # Documentação do pacote raiz
├── go.mod
├── go.sum
├── otel_config.go          # Configuração e functional options
├── otel_provider.go        # Inicialização dos providers OTel via OTLP/gRPC
├── otel_sdk.go              # Ponto de entrada — New() e Shutdown()
├── CHANGELOG.md
├── LICENSE
├── .golangci.yml
└── README.md
```

---

## Executando os testes

```bash
# Todos os testes
go test ./test/... -count=1 -v

# Apenas um pacote
go test ./test/otelhttp/... -count=1 -v
go test ./test/otelnats/... -count=1 -v
go test ./test/otelsql/... -count=1 -v
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

## Troubleshooting

| Sintoma | Causa provável | Solução |
|---------|-----------------|---------|
| `gotel: at least one signal must be enabled` | Nenhum `WithTracing()`/`WithMetrics()`/`WithLogging()` foi passado | Habilite ao menos um sinal em `gotel.New(...)` |
| `gotel: ServiceName is required` / `CollectorEndpoint is required` | `WithServiceName`/`WithCollectorEndpoint` não foram fornecidos (nem via `WithEnvConfig()`) | Passe a opção explicitamente ou defina `OTEL_SERVICE_NAME`/`OTEL_EXPORTER_OTLP_ENDPOINT` |
| Erro de handshake TLS ao conectar no Collector local | Por padrão a conexão usa TLS (`Insecure = false`), mas o Collector local não tem certificado configurado | Use `gotel.WithInsecure(true)` (ou `OTEL_INSECURE=true`) — apenas em desenvolvimento |
| `New()` retorna sucesso, mas nenhum dado chega no Collector | `grpc.NewClient` não conecta de imediato — falhas de rede só aparecem ao tentar exportar (ex: no `Shutdown`) | Confirme que o Collector está no ar e acessível no endereço configurado; verifique o erro retornado por `sdk.Shutdown(ctx)` |
| Logs não aparecem com `trace_id`/`span_id` | `slog.SetDefault(sdk.SlogLogger())` não foi chamado, ou o log foi emitido sem contexto (`slog.Info` em vez de `slog.InfoContext`) | Use sempre as variantes `*Context` do `slog` (`InfoContext`, `ErrorContext`, etc.) passando o `ctx` do span ativo |
| `otelhttp.NewMiddleware` retorna erro | Falha ao criar os instrumentos de métrica (`WithMeter`) — normalmente conflito de nome/descrição de instrumento no `Meter` compartilhado | Verifique se o mesmo `meter.Meter` não está sendo usado para registrar instrumentos com o mesmo nome e unidades/descrições diferentes em outro lugar |

---

## Versionamento e compatibilidade

Esta lib ainda está **pré-1.0** (`v0.x`) — breaking changes podem ocorrer em releases menores, sempre documentadas no [`CHANGELOG.md`](CHANGELOG.md). Após a primeira tag `v1.0.0`, passa a seguir [SemVer](https://semver.org/) estrito.

### Migrando de versões anteriores

Se você usava a lib antes desta rodada de mudanças (pacotes ainda em `internal/`):

- Troque os imports de `github.com/DSanches92/go-otel/internal/{http,nats,sql}` para
  `github.com/DSanches92/go-otel/otel{http,nats,sql}` — e pode remover o alias
  (`httpgotel`, `natsotel`, `sqlgotel`), já que o nome do pacote agora é único.
- `sqlgotel.NewDatabase(...)` virou `otelsql.NewDB(...)`.
- `httpgotel.WithTransport(...)` virou `otelhttp.WithRoundTripper(...)`.
- `sdk.Meter()` virou `sdk.Metric()`.
- `otelhttp.NewMiddleware(...)` agora retorna `(func(http.Handler) http.Handler, error)`
  em vez de só o handler — trate o erro na inicialização:

  ```go
  // antes
  handler := httpgotel.NewMiddleware(sdk.Tracer())

  // depois
  middleware, err := otelhttp.NewMiddleware(sdk.Tracer())
  if err != nil {
      log.Fatal(err)
  }
  handler := middleware(mux)
  ```

Veja a seção `[Unreleased]` do [`CHANGELOG.md`](CHANGELOG.md) para a lista completa.

---

## Contribuindo

1. Clone o repositório e rode os testes antes de abrir um PR:

   ```bash
   go build ./...
   go vet ./...
   gofmt -l .              # deve retornar vazio
   go test ./... -race
   ```

2. Rode o lint localmente (mesma config usada no CI):

   ```bash
   go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run
   ```

3. Commits seguem [Conventional Commits](https://www.conventionalcommits.org/) (`feat:`, `fix:`, `refactor:`, `docs:`, `ci:` etc.).
4. Mudanças que alteram comportamento público devem atualizar o [`CHANGELOG.md`](CHANGELOG.md) na seção `[Unreleased]`.

---

## Licença

Distribuído sob a licença MIT. Veja [`LICENSE`](LICENSE) para o texto completo.

---

<p align="center">
  Criado com ❤️ por <a href="https://github.com/DSanches92">Danilo Sanches</a>
</p>
