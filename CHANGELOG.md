# Changelog

Todas as mudanças relevantes deste projeto são documentadas neste arquivo.

O formato é baseado em [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Changed

- **Breaking:** os subpacotes de instrumentação foram movidos de `internal/{http,nats,sql}`
  para pacotes exportados na raiz — `otelhttp`, `otelnats`, `otelsql` — já que pacotes
  em `internal/` não podem ser importados por consumidores externos do módulo.
- **Breaking:** `otelhttp.NewMiddleware` agora retorna `(func(http.Handler) http.Handler, error)`
  em vez de descartar silenciosamente o erro de criação das métricas quando `WithMeter` é usado.
- `otelnats.Tracer.QueueSubscribe` agora inclui o grupo de fila no nome do span
  (ex: `SUBSCRIBE orders.created [workers]`, antes apenas `SUBSCRIBE orders.created`).

### Added

- `SDK.MeterProvider()` e `SDK.LoggerProvider()`, espelhando o já existente `SDK.TracerProvider()`.
- `otelsql` agora reconhece `CREATE`, `ALTER` e `DROP` como operações SQL
  (antes caíam em `UNKNOWN`).
- Comentários de documentação de pacote (`doc.go`) para o pacote raiz `gotel` e para
  `otelhttp`, `otelnats`, `otelsql`.
- `.golangci.yml` e um step de `golangci-lint` no CI.

### Fixed

- `gotel.New` não vaza mais a conexão gRPC compartilhada nem os processors/goroutines
  de providers já inicializados quando um provider subsequente falha ao inicializar.
- `SDK.SlogLogger()` agora retorna um logger cacheado em vez de construir um novo a
  cada chamada.

## [0.2.0] - 2026-06-09

- Adicionada instrumentação de `database/sql` (`otelsql`, então `internal/sql`), com
  suporte a Oracle/PostgreSQL/MySQL via `NewDB`.
- Correções de compatibilidade de versão dos pacotes do SDK.

## [0.1.0] - 2026-05-23

- Lançamento inicial: wrapper do SDK OpenTelemetry (`gotel.New`) com Traces, Métricas
  e Logs via OTLP/gRPC.
