# GO-OTEL

[![CI](https://github.com/DSanches92/go-otel/actions/workflows/ci.yml/badge.svg)](https://github.com/DSanches92/go-otel/actions/workflows/ci.yml)

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
sdk, err := gotel.New(
    gotel.WithServiceName("orders-ms"),
    gotel.WithCollectorEndpoint("otel-collector:4317"),
    gotel.WithServiceVersion("1.0.0"),
    gotel.WithEnvironment("production"),
    gotel.WithTracing(),
    gotel.WithMetrics(),
    gotel.WithLogging(),
)
if err != nil {
    log.Fatal(err)
}
defer sdk.Shutdown(context.Background())

tracer := sdk.Tracer()
meter  := sdk.Meter()
logger := sdk.Logger()
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

---

## Microsserviço NATS

O subpacote `nats` fornece um `TextMapCarrier` para propagar contexto de trace via headers de mensagem NATS.

### Publicando (inject)

```go
import natsotel "github.com/DSanches92/go-otel/src/nats"

msg := &nats.Msg{Subject: "orders.created"}

carrier := natsotel.NewCarrier(msg)
propagator.Inject(ctx, carrier)

nc.PublishMsg(msg)
```

### Consumindo (extract)

```go
nc.Subscribe("orders.created", func(msg *nats.Msg) {
    carrier := natsotel.NewCarrier(msg)
    ctx := propagator.Extract(context.Background(), carrier)

    ctx, span := tracer.Start(ctx, "orders.created.processar")
    defer span.End()

    // span é filho do span do publicador
})
```

---

## API Gateway HTTP

O subpacote `http` fornece um middleware que instrumenta automaticamente
cada request com spans, atributos semânticos e propagação de contexto.

### O que é instrumentado automaticamente

- Nome do span: `"MÉTODO /rota"` — ex: `"GET /orders"`
- `http.request.method` — método HTTP
- `url.path` — caminho da URL
- `http.response.status_code` — status code da resposta
- Status de erro — marcado automaticamente para status `>= 400`

### Uso com net/http

```go
import httpgotel "github.com/DSanches92/go-otel/src/http"

mux := http.NewServeMux()
mux.HandleFunc("GET /orders", handleOrders)

handler := httpgotel.NewMiddleware(provider)(mux)
http.ListenAndServe(":8080", handler)
```

### Uso com chi

```go
r := chi.NewRouter()
r.Use(httpgotel.NewMiddleware(provider))
r.Get("/orders", handleOrders)
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
### Uso

```go
import (
    "database/sql"

    _ "github.com/sijms/go-ora/v2"
    sqlgotel "github.com/DSanches92/go-otel/src/sql"
)

sqlDB, _ := sql.Open("oracle", connString)

database, err := sqlgotel.NewDatabase(sqlDB, sdk.Tracer(),
    sqlgotel.WithDBSystem("oracle"),
    sqlgotel.WithDBName("myschema"),
    sqlgotel.WithServerAddress("oracle-host", 1521),
    // sqlgotel.WithStatementRecording(true), // apenas para debug
)

rows, err := database.QueryContext(ctx, "SELECT * FROM orders WHERE id = :1", id)
result, err := database.ExecContext(ctx, "INSERT INTO orders VALUES (:1)", id)

tx, err := database.BeginTx(ctx, nil)
defer tx.Rollback(ctx)
err = tx.Commit(ctx)
```

### Segurança

SQL e parâmetros **não são registrados por default** — habilite explicitamente apenas quando necessário:

```go
sqlgotel.WithStatementRecording(true)  // registra SQL
sqlgotel.WithParameterRecording(true)  // registra parâmetros — nunca em produção
```

---

## Estrutura do projeto

```
go-otel/
├── src/
│   ├── http/              # Middleware HTTP com spans automáticos
│   │   ├── doc.go
│   │   └── middleware.go
│   ├── nats/              # TextMapCarrier para headers NATS
│   │   ├── doc.go
│   │   └── carrier.go
│   └── sql/              # Wrapper database/sql com spans automáticos
│       ├── doc.go
│       ├── database.go
│       └── transaction.go
│
├── tests/
│   ├── http/
│   │   └── middleware_test.go
│   ├── nats/
│   │   └── carrier_test.go
│   ├── sql/
│   │   └── database_test.go
│   ├── otel_config_test.go
│   └── otel_sdk_test.go
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
go test ./test/... -v

# Apenas um pacote
go test ./test/http/... -v
go test ./test/nats/... -v
go test ./test/sql/... -v
```

---

## Variáveis de ambiente (exemplos)

| Variável                  | Descrição                              | Default                 |
|---------------------------|----------------------------------------|-------------------------|
| `OTEL_COLLECTOR_ENDPOINT` | Endereço do Collector                  | `localhost:4317`        |
| `APP_ENV`                 | Ambiente (`development`, `production`) | `development`           |
| `NATS_URL`                | URL do servidor NATS                   | `nats://localhost:4222` |
| `HTTP_ADDR`               | Endereço do servidor HTTP              | `:8080`                 |

---

## Dependências principais

| Pacote                                        | Versão  | Uso                 |
|-----------------------------------------------|---------|---------------------|
| `go.opentelemetry.io/otel`                    | v1.44.0 | SDK base            |
| `go.opentelemetry.io/otel/sdk`                | v1.44.0 | Providers           |
| `go.opentelemetry.io/otel/exporters/otlp/...` | v1.44.0 | Exporters OTLP/gRPC |
| `github.com/nats-io/nats.go`                  | v1.52.0 | Cliente NATS        |
| `google.golang.org/grpc`                      | v1.81.1 | Transporte gRPC     |

---

<p align="center">
  Criado com ❤️ por <a href="https://github.com/DSanches92">Danilo Sanches</a>
</p>
