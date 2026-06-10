// Package sql fornece instrumentação OpenTelemetry para qualquer banco de dados
// compatível com o pacote padrão database/sql do Go.
//
// Ao contrário de wrappers específicos por banco, este pacote instrumenta
// diretamente as interfaces do database/sql — funcionando com qualquer driver
// registrado, incluindo Oracle (go-ora), PostgreSQL (pgx), MySQL e outros.
//
// # Tipos instrumentados
//
//   - [DB]  — wrapper sobre *sql.DB com QueryContext, ExecContext e BeginTx
//   - [Tx]  — wrapper sobre *sql.Tx com QueryContext, ExecContext, Commit e Rollback
//
// # Spans gerados
//
//   - sql.query              → QueryContext em DB ou Tx
//   - sql.exec               → ExecContext em DB ou Tx
//   - sql.transaction.begin  → BeginTx
//   - sql.transaction.commit → Tx.Commit
//   - sql.transaction.rollback → Tx.Rollback
//
// # Atributos semânticos (OpenTelemetry Semantic Conventions)
//
//   - db.system    → sistema de banco (ex: "oracle", "postgresql")
//   - db.name      → nome do schema/banco (opcional)
//   - db.operation → operação SQL (SELECT, INSERT, UPDATE, DELETE)
//   - db.statement → SQL executado (desabilitado por default)
//   - db.parameters → parâmetros da query (desabilitado por default)
//   - server.address → host do servidor (opcional)
//   - server.port    → porta do servidor (opcional)
//
// # Uso com Oracle
//
//	import (
//	    "database/sql"
//	    _ "github.com/sijms/go-ora/v2"
//	    sqlgotel "github.com/DSanches92/go-otel/src/sql"
//	)
//
//	sqlDB, _ := sql.Open("oracle", "oracle://user:pass@host:1521/schema")
//
//	db, err := sqlgotel.NewDB(sqlDB, sdk.Tracer(),
//	    sqlgotel.WithDBSystem("oracle"),
//	    sqlgotel.WithDBName("myschema"),
//	)
package sql
