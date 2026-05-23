# go-otel

Lib de observabilidade padronizada para aplicações Go, construída sobre o [OpenTelemetry SDK](https://opentelemetry.io/docs/languages/go/).

Centraliza a inicialização de **Traces**, **Métricas** e **Logs** via OTLP/gRPC, com suporte a microsserviços NATS e aplicações HTTP.

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
import natsotel "github.com/DSanches92/go-otel/nats"

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
import httpgotel "github.com/DSanches92/go-otel/http"

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

## Estrutura do projeto

```
go-otel/
├── src/
│   ├── examples/
│   │   ├── http-gateway/      # Exemplo: API Gateway HTTP
│   │   │   └── main.go
│   │   └── nats-ms/           # Exemplo: microsserviço NATS
│   │       └── main.go
│   │
│   └── instrumentation/
│       ├── http/              # Middleware HTTP com spans automáticos
│       │   ├── doc.go
│       │   └── middleware.go
│       └── nats/              # TextMapCarrier para headers NATS
│           ├── doc.go
│           └── carrier.go
│
├── tests/
│   ├── instrumentation/
│   │   ├── http/
│   │   │   └── middleware_test.go
│   │   └── nats/
│   │       └── carrier_test.go
│   ├── config_test.go
│   └── sdk_test.go
│
├── doc.go                 # Documentação do pacote raiz
├── config.go              # Configuração e functional options
├── sdk.go                 # Ponto de entrada — New() e Shutdown()
├── provider.go            # Inicialização dos providers OTel via OTLP/gRPC
├── go.mod
├── go.sum
└── README.md
```

---

## Executando os testes

```bash
# Todos os testes
go test ./... -v

# Apenas um pacote
go test ./nats/... -v
go test ./http/... -v
```

---

## Variáveis de ambiente (exemplos)

| Variável | Descrição | Default |
|----------|-----------|---------|
| `OTEL_COLLECTOR_ENDPOINT` | Endereço do Collector | `localhost:4317` |
| `APP_ENV` | Ambiente (`development`, `production`) | `development` |
| `NATS_URL` | URL do servidor NATS | `nats://localhost:4222` |
| `HTTP_ADDR` | Endereço do servidor HTTP | `:8080` |

---

## Dependências principais

| Pacote | Versão | Uso |
|--------|--------|-----|
| `go.opentelemetry.io/otel` | v1.32.0 | SDK base |
| `go.opentelemetry.io/otel/sdk` | v1.32.0 | Providers |
| `go.opentelemetry.io/otel/exporters/otlp/...` | v1.32.0 | Exporters OTLP/gRPC |
| `github.com/nats-io/nats.go` | v1.37.0 | Cliente NATS |
| `google.golang.org/grpc` | v1.68.0 | Transporte gRPC |
