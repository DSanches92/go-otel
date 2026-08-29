// Package otelsql fornece instrumentação OpenTelemetry para qualquer banco de dados
// compatível com o pacote padrão database/sql do Go.
//
// Ao contrário de wrappers específicos por banco, este pacote instrumenta
// diretamente as interfaces do database/sql — funcionando com qualquer driver
// registrado, incluindo Oracle (go-ora), PostgreSQL (pgx), MySQL e outros.
//
// # Tipos instrumentados
//
//   - [Database] — wrapper sobre *sql.DB com QueryContext, QueryRowContext, ExecContext e BeginTx
//   - [Transaction] — wrapper sobre *sql.Tx com QueryContext, QueryRowContext, ExecContext, Commit e Rollback
//   - [Row] — retornado por QueryRowContext, com Scan(dest ...any) error
//
// # Spans gerados
//
//   - sql.query                → QueryContext/QueryRowContext em Database ou Transaction
//   - sql.exec                 → ExecContext em Database ou Transaction
//   - sql.transaction.begin    → BeginTx
//   - sql.transaction.commit   → Transaction.Commit
//   - sql.transaction.rollback → Transaction.Rollback
//
// # Atributos semânticos (OpenTelemetry Semantic Conventions)
//
//   - db.system     → sistema de banco (ex: "oracle", "postgresql")
//   - db.name       → nome do schema/banco (opcional)
//   - db.operation  → operação SQL extraída da query (SELECT, INSERT, UPDATE,
//     DELETE, WITH, CALL, MERGE, REPLACE, TRUNCATE, CREATE, ALTER, DROP)
//   - db.statement  → SQL executado (desabilitado por default)
//   - db.parameters → parâmetros da query (desabilitado por default)
//   - server.address → host do servidor (opcional)
//   - server.port    → porta do servidor (opcional)
//
// # Uso com Oracle
//
//	import (
//	    "database/sql"
//	    _ "github.com/sijms/go-ora/v2"
//	    "github.com/DSanches92/go-otel/otelsql"
//	)
//
//	sqlDB, _ := sql.Open("oracle", "oracle://user:pass@host:1521/schema")
//
//	db, err := otelsql.NewDB(sqlDB, sdk.Tracer(),
//	    otelsql.WithDBSystem("oracle"),
//	    otelsql.WithDBName("myschema"),
//	)
package otelsql
