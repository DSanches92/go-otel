package sql_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"testing"

	sqlgotel "github.com/DSanches92/go-otel/src/sql"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

type fakeDriver struct{ err error }
type fakeConn struct{ err error }
type fakeStmt struct{ err error }
type fakeRows struct{ closed bool }
type fakeResult struct{}
type fakeTx struct{}

func (driver *fakeDriver) Open(_ string) (driver.Conn, error) {
	if driver.err != nil {
		return nil, driver.err
	}
	return &fakeConn{err: driver.err}, nil
}
func (connection *fakeConn) Prepare(query string) (driver.Stmt, error) {
	return &fakeStmt{err: connection.err}, nil
}
func (connection *fakeConn) Close() error              { return nil }
func (connection *fakeConn) Begin() (driver.Tx, error) { return &fakeTx{}, connection.err }
func (statement *fakeStmt) Close() error               { return nil }
func (statement *fakeStmt) NumInput() int              { return -1 }
func (statement *fakeStmt) Exec(_ []driver.Value) (driver.Result, error) {
	if statement.err != nil {
		return nil, statement.err
	}
	return &fakeResult{}, nil
}
func (statement *fakeStmt) Query(_ []driver.Value) (driver.Rows, error) {
	if statement.err != nil {
		return nil, statement.err
	}
	return &fakeRows{}, nil
}
func (rows *fakeRows) Columns() []string                { return []string{"id"} }
func (rows *fakeRows) Close() error                     { return nil }
func (rows *fakeRows) Next(_ []driver.Value) error      { return io.EOF }
func (result *fakeResult) LastInsertId() (int64, error) { return 1, nil }
func (result *fakeResult) RowsAffected() (int64, error) { return 1, nil }
func (transaction *fakeTx) Commit() error               { return nil }
func (transaction *fakeTx) Rollback() error             { return nil }

var driverRegistered = false

func newSQLDB(test *testing.T) *sql.DB {
	test.Helper()

	if !driverRegistered {
		sql.Register("fakedb", &fakeDriver{})
		driverRegistered = true
	}

	database, err := sql.Open("fakedb", "")
	if err != nil {
		test.Fatalf("setup: falha ao abrir fakedb: %v", err)
	}

	return database
}

func newSQLDBWithError(test *testing.T, err error) *sql.DB {
	test.Helper()

	nome := "fakedb-err-" + err.Error()
	sql.Register(nome, &fakeDriver{err: err})

	database, _ := sql.Open(nome, "")
	return database
}

func newTracerProviderInMemory(test *testing.T) (*sdktrace.TracerProvider, *tracetest.SpanRecorder) {
	test.Helper()

	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(recorder),
	)

	return provider, recorder
}

func assertAttribute(test *testing.T, span sdktrace.ReadOnlySpan, chave, valorEsperado string) {
	test.Helper()

	for _, attr := range span.Attributes() {
		if string(attr.Key) == chave {
			if attr.Value.AsString() != valorEsperado {
				test.Errorf("atributo '%s': esperado '%s', obtido '%s'",
					chave, valorEsperado, attr.Value.AsString())
			}
			return
		}
	}

	test.Errorf("atributo '%s' não encontrado no span", chave)
}

func assertAttributeMissing(test *testing.T, span sdktrace.ReadOnlySpan, chave string) {
	test.Helper()

	for _, attr := range span.Attributes() {
		if string(attr.Key) == chave {
			test.Errorf("atributo '%s' não deveria estar presente no span", chave)
			return
		}
	}
}

func TestDB_NewDB(test *testing.T) {
	test.Run("deve criar DB com sucesso", func(test *testing.T) {
		provider, _ := newTracerProviderInMemory(test)
		sqlDB := newSQLDB(test)

		database, err := sqlgotel.NewDB(sqlDB, provider.Tracer("test"),
			sqlgotel.WithDBSystem("oracle"),
		)

		if err != nil {
			test.Errorf("não esperado erro, obtido '%s'", err)
		}
		if database == nil {
			test.Error("esperado DB não-nil")
		}
	})

	test.Run("deve retornar erro quando DBSystem não é informado", func(test *testing.T) {
		provider, _ := newTracerProviderInMemory(test)
		sqlDB := newSQLDB(test)

		_, err := sqlgotel.NewDB(sqlDB, provider.Tracer("test"))

		if err == nil {
			test.Error("esperado erro quando DBSystem está ausente")
		}
	})

	test.Run("deve retornar erro quando sql.DB é nil", func(test *testing.T) {
		provider, _ := newTracerProviderInMemory(test)

		_, err := sqlgotel.NewDB(nil, provider.Tracer("test"),
			sqlgotel.WithDBSystem("oracle"),
		)

		if err == nil {
			test.Error("esperado erro quando sql.DB é nil")
		}
	})

	test.Run("StatementRecording deve ser false por default", func(test *testing.T) {
		provider, _ := newTracerProviderInMemory(test)
		sqlDB := newSQLDB(test)

		database, _ := sqlgotel.NewDB(sqlDB, provider.Tracer("test"),
			sqlgotel.WithDBSystem("oracle"),
		)

		if database.StatementRecording() {
			test.Error("esperado 'false' por default — secure by default")
		}
	})

	test.Run("ParameterRecording deve ser false por default", func(test *testing.T) {
		provider, _ := newTracerProviderInMemory(test)
		sqlDB := newSQLDB(test)

		database, _ := sqlgotel.NewDB(sqlDB, provider.Tracer("test"),
			sqlgotel.WithDBSystem("oracle"),
		)

		if database.ParameterRecording() {
			test.Error("esperado 'false' por default — secure by default")
		}
	})
}

func TestDB_QueryContext(test *testing.T) {
	test.Run("deve criar span com nome sql.query", func(test *testing.T) {
		provider, recorder := newTracerProviderInMemory(test)
		sqlDB := newSQLDB(test)
		database, _ := sqlgotel.NewDB(sqlDB, provider.Tracer("test"),
			sqlgotel.WithDBSystem("oracle"),
		)

		rows, err := database.QueryContext(context.Background(), "SELECT 1 FROM dual")
		if err != nil {
			test.Fatalf("não esperado erro, obtido '%s'", err)
		}
		defer rows.Close()

		spans := recorder.Ended()
		if len(spans) != 1 {
			test.Fatalf("esperado 1 span, obtido %d", len(spans))
		}
		if spans[0].Name() != "sql.query" {
			test.Errorf("esperado 'sql.query', obtido '%s'", spans[0].Name())
		}
	})

	test.Run("deve registrar db.system no span", func(test *testing.T) {
		provider, recorder := newTracerProviderInMemory(test)
		sqlDB := newSQLDB(test)
		database, _ := sqlgotel.NewDB(sqlDB, provider.Tracer("test"),
			sqlgotel.WithDBSystem("oracle"),
		)

		rows, _ := database.QueryContext(context.Background(), "SELECT 1 FROM dual")
		defer rows.Close()

		assertAttribute(test, recorder.Ended()[0], "db.system", "oracle")
	})

	test.Run("deve registrar db.operation como SELECT", func(test *testing.T) {
		provider, recorder := newTracerProviderInMemory(test)
		sqlDB := newSQLDB(test)
		database, _ := sqlgotel.NewDB(sqlDB, provider.Tracer("test"),
			sqlgotel.WithDBSystem("oracle"),
		)

		rows, _ := database.QueryContext(context.Background(), "SELECT 1 FROM dual")
		defer rows.Close()

		assertAttribute(test, recorder.Ended()[0], "db.operation", "SELECT")
	})

	test.Run("não deve registrar db.statement por default", func(test *testing.T) {
		provider, recorder := newTracerProviderInMemory(test)
		sqlDB := newSQLDB(test)
		database, _ := sqlgotel.NewDB(sqlDB, provider.Tracer("test"),
			sqlgotel.WithDBSystem("oracle"),
		)

		rows, _ := database.QueryContext(context.Background(), "SELECT 1 FROM dual")
		defer rows.Close()

		assertAttributeMissing(test, recorder.Ended()[0], "db.statement")
	})

	test.Run("deve registrar db.statement quando habilitado", func(test *testing.T) {
		provider, recorder := newTracerProviderInMemory(test)
		sqlDB := newSQLDB(test)
		database, _ := sqlgotel.NewDB(sqlDB, provider.Tracer("test"),
			sqlgotel.WithDBSystem("oracle"),
			sqlgotel.WithStatementRecording(true),
		)

		rows, _ := database.QueryContext(context.Background(), "SELECT 1 FROM dual")
		defer rows.Close()

		assertAttribute(test, recorder.Ended()[0], "db.statement", "SELECT 1 FROM dual")
	})

	test.Run("deve marcar span como erro quando query falha", func(test *testing.T) {
		provider, recorder := newTracerProviderInMemory(test)
		sqlDB := newSQLDBWithError(test, errors.New("connection refused"))
		database, _ := sqlgotel.NewDB(sqlDB, provider.Tracer("test"),
			sqlgotel.WithDBSystem("oracle"),
		)

		_, err := database.QueryContext(context.Background(), "SELECT 1 FROM dual")
		if err == nil {
			test.Fatal("esperado erro")
		}

		spans := recorder.Ended()
		if spans[0].Status().Code != codes.Error {
			test.Error("esperado span com código de erro")
		}
	})
}

func TestDB_ExecContext(test *testing.T) {
	test.Run("deve criar span com nome sql.exec", func(test *testing.T) {
		provider, recorder := newTracerProviderInMemory(test)
		sqlDB := newSQLDB(test)
		database, _ := sqlgotel.NewDB(sqlDB, provider.Tracer("test"),
			sqlgotel.WithDBSystem("oracle"),
		)

		_, err := database.ExecContext(context.Background(), "INSERT INTO orders VALUES (1)")
		if err != nil {
			test.Fatalf("não esperado erro, obtido '%s'", err)
		}

		spans := recorder.Ended()
		if spans[0].Name() != "sql.exec" {
			test.Errorf("esperado 'sql.exec', obtido '%s'", spans[0].Name())
		}
	})

	test.Run("deve registrar db.operation como INSERT", func(test *testing.T) {
		provider, recorder := newTracerProviderInMemory(test)
		sqlDB := newSQLDB(test)
		database, _ := sqlgotel.NewDB(sqlDB, provider.Tracer("test"),
			sqlgotel.WithDBSystem("oracle"),
		)

		database.ExecContext(context.Background(), "INSERT INTO orders VALUES (1)")

		assertAttribute(test, recorder.Ended()[0], "db.operation", "INSERT")
	})

	test.Run("deve marcar span como erro quando exec falha", func(test *testing.T) {
		provider, recorder := newTracerProviderInMemory(test)
		sqlDB := newSQLDBWithError(test, errors.New("table not found"))
		database, _ := sqlgotel.NewDB(sqlDB, provider.Tracer("test"),
			sqlgotel.WithDBSystem("oracle"),
		)

		_, err := database.ExecContext(context.Background(), "INSERT INTO orders VALUES (1)")
		if err == nil {
			test.Fatal("esperado erro")
		}

		spans := recorder.Ended()
		if spans[0].Status().Code != codes.Error {
			test.Error("esperado span com código de erro")
		}
	})
}

func TestDB_BeginTx(test *testing.T) {
	test.Run("deve criar span com nome sql.transaction.begin", func(test *testing.T) {
		provider, recorder := newTracerProviderInMemory(test)
		sqlDB := newSQLDB(test)
		database, _ := sqlgotel.NewDB(sqlDB, provider.Tracer("test"),
			sqlgotel.WithDBSystem("oracle"),
		)

		transaction, err := database.BeginTx(context.Background(), nil)
		if err != nil {
			test.Fatalf("não esperado erro, obtido '%s'", err)
		}
		defer transaction.Rollback(context.Background())

		spans := recorder.Ended()
		if len(spans) != 1 {
			test.Fatalf("esperado 1 span, obtido %d", len(spans))
		}
		if spans[0].Name() != "sql.transaction.begin" {
			test.Errorf("esperado 'sql.transaction.begin', obtido '%s'", spans[0].Name())
		}
	})
}

func TestTx_Commit(test *testing.T) {
	test.Run("deve criar span com nome sql.transaction.commit", func(test *testing.T) {
		provider, recorder := newTracerProviderInMemory(test)
		sqlDB := newSQLDB(test)
		database, _ := sqlgotel.NewDB(sqlDB, provider.Tracer("test"),
			sqlgotel.WithDBSystem("oracle"),
		)

		transaction, _ := database.BeginTx(context.Background(), nil)
		err := transaction.Commit(context.Background())
		if err != nil {
			test.Fatalf("não esperado erro, obtido '%s'", err)
		}

		nameFound := false
		for _, span := range recorder.Ended() {
			if span.Name() == "sql.transaction.commit" {
				nameFound = true
				break
			}
		}
		if !nameFound {
			test.Error("esperado span 'sql.transaction.commit'")
		}
	})
}

func TestTx_Rollback(test *testing.T) {
	test.Run("deve criar span com nome sql.transaction.rollback", func(test *testing.T) {
		provider, recorder := newTracerProviderInMemory(test)
		sqlDB := newSQLDB(test)
		database, _ := sqlgotel.NewDB(sqlDB, provider.Tracer("test"),
			sqlgotel.WithDBSystem("oracle"),
		)

		transaction, _ := database.BeginTx(context.Background(), nil)
		err := transaction.Rollback(context.Background())
		if err != nil {
			test.Fatalf("não esperado erro, obtido '%s'", err)
		}

		nameFound := false
		for _, span := range recorder.Ended() {
			if span.Name() == "sql.transaction.rollback" {
				nameFound = true
				break
			}
		}
		if !nameFound {
			test.Error("esperado span 'sql.transaction.rollback'")
		}
	})
}

func TestTx_QueryContext(test *testing.T) {
	test.Run("deve criar span com nome sql.query dentro da transação", func(test *testing.T) {
		provider, recorder := newTracerProviderInMemory(test)
		sqlDB := newSQLDB(test)
		database, _ := sqlgotel.NewDB(sqlDB, provider.Tracer("test"),
			sqlgotel.WithDBSystem("oracle"),
		)

		transaction, _ := database.BeginTx(context.Background(), nil)
		defer transaction.Rollback(context.Background())

		rows, err := transaction.QueryContext(context.Background(), "SELECT 1 FROM dual")
		if err != nil {
			test.Fatalf("não esperado erro, obtido '%s'", err)
		}
		defer rows.Close()

		nameFound := false
		for _, span := range recorder.Ended() {
			if span.Name() == "sql.query" {
				nameFound = true
				break
			}
		}
		if !nameFound {
			test.Error("esperado span 'sql.query' dentro da transação")
		}
	})
}

func TestTx_ExecContext(test *testing.T) {
	test.Run("deve criar span com nome sql.exec dentro da transação", func(test *testing.T) {
		provider, recorder := newTracerProviderInMemory(test)
		sqlDB := newSQLDB(test)
		database, _ := sqlgotel.NewDB(sqlDB, provider.Tracer("test"),
			sqlgotel.WithDBSystem("oracle"),
		)

		transaction, _ := database.BeginTx(context.Background(), nil)
		defer transaction.Rollback(context.Background())

		_, err := transaction.ExecContext(context.Background(), "INSERT INTO orders VALUES (1)")
		if err != nil {
			test.Fatalf("não esperado erro, obtido '%s'", err)
		}

		nameFound := false
		for _, span := range recorder.Ended() {
			if span.Name() == "sql.exec" {
				nameFound = true
				break
			}
		}
		if !nameFound {
			test.Error("esperado span 'sql.exec' dentro da transação")
		}
	})
}

func TestRows_Span(test *testing.T) {
	test.Run("span deve permanecer aberto enquanto Rows não for fechado", func(test *testing.T) {
		provider, recorder := newTracerProviderInMemory(test)
		sqlDB := newSQLDB(test)
		database, _ := sqlgotel.NewDB(sqlDB, provider.Tracer("test"),
			sqlgotel.WithDBSystem("oracle"),
		)

		rows, _ := database.QueryContext(context.Background(), "SELECT 1 FROM dual")

		if len(recorder.Ended()) != 1 {
			test.Errorf("esperado 1 span após QueryContext, obtido %d", len(recorder.Ended()))
		}

		rows.Close()
	})
}
