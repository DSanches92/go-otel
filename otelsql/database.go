package otelsql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var (
	ErrDBNil            = errors.New("otelsql: sql.DB must not be nil")
	ErrDBSystemRequired = errors.New("otelsql: WithDBSystem is required")
)

const (
	attrDBSystem     = attribute.Key("db.system")
	attrDBName       = attribute.Key("db.name")
	attrDBStatement  = attribute.Key("db.statement")
	attrDBOperation  = attribute.Key("db.operation")
	attrDBParameters = attribute.Key("db.parameters")
	attrServerAddr   = attribute.Key("server.address")
	attrServerPort   = attribute.Key("server.port")
)

type config struct {
	dbSystem           string
	dbName             string
	serverAddress      string
	serverPort         int
	statementRecording bool
	parameterRecording bool
}

type Option func(*config)

func WithDBSystem(system string) Option {
	return func(config *config) {
		config.dbSystem = system
	}
}

func WithDBName(name string) Option {
	return func(config *config) {
		config.dbName = name
	}
}

func WithServerAddress(host string, port int) Option {
	return func(config *config) {
		config.serverAddress = host
		config.serverPort = port
	}
}

func WithStatementRecording(enabled bool) Option {
	return func(config *config) {
		config.statementRecording = enabled
	}
}

func WithParameterRecording(enabled bool) Option {
	return func(config *config) {
		config.parameterRecording = enabled
	}
}

type Database struct {
	database *sql.DB
	tracer   trace.Tracer
	config   *config
}

func NewDB(database *sql.DB, tracer trace.Tracer, opts ...Option) (*Database, error) {
	if database == nil {
		return nil, ErrDBNil
	}

	config := &config{}
	for _, opt := range opts {
		opt(config)
	}

	if config.dbSystem == "" {
		return nil, ErrDBSystemRequired
	}

	return &Database{
		database: database,
		tracer:   tracer,
		config:   config,
	}, nil
}

func (database *Database) StatementRecording() bool {
	return database.config.statementRecording
}

func (database *Database) ParameterRecording() bool {
	return database.config.parameterRecording
}

func (database *Database) Unwrap() *sql.DB {
	return database.database
}

func (database *Database) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	ctx, span := database.startSpan(ctx, "sql.query")
	defer span.End()

	database.setQueryAttributes(span, query, args)

	rows, err := database.database.QueryContext(ctx, query, args...)
	if err != nil {
		recordError(span, err)
		return nil, err
	}

	return rows, nil
}

func (database *Database) QueryRowContext(ctx context.Context, query string, args ...any) (*Row, error) {
	ctx, span := database.startSpan(ctx, "sql.query")
	defer span.End()

	database.setQueryAttributes(span, query, args)

	rows, err := database.database.QueryContext(ctx, query, args...)
	if err != nil {
		recordError(span, err)
		return nil, err
	}

	return &Row{rows: rows}, nil
}

func (database *Database) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	ctx, span := database.startSpan(ctx, "sql.exec")
	defer span.End()

	database.setQueryAttributes(span, query, args)

	result, err := database.database.ExecContext(ctx, query, args...)
	if err != nil {
		recordError(span, err)
		return nil, err
	}

	return result, nil
}

func (database *Database) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Transaction, error) {
	_, span := database.startSpan(ctx, "sql.transaction.begin")
	defer span.End()

	transaction, err := database.database.BeginTx(ctx, opts)
	if err != nil {
		recordError(span, err)
		return nil, err
	}

	return &Transaction{transaction: transaction, database: database}, nil
}

func (database *Database) startSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	ctx, span := database.tracer.Start(ctx, name)

	attrs := []attribute.KeyValue{
		attrDBSystem.String(database.config.dbSystem),
	}

	if database.config.dbName != "" {
		attrs = append(attrs, attrDBName.String(database.config.dbName))
	}

	if database.config.serverAddress != "" {
		attrs = append(attrs,
			attrServerAddr.String(database.config.serverAddress),
			attrServerPort.Int(database.config.serverPort),
		)
	}

	span.SetAttributes(attrs...)

	return ctx, span
}

func (database *Database) setQueryAttributes(span trace.Span, query string, args []any) {
	span.SetAttributes(attrDBOperation.String(extractOperation(query)))

	if database.config.statementRecording {
		span.SetAttributes(attrDBStatement.String(query))
	}

	if database.config.parameterRecording && len(args) > 0 {
		span.SetAttributes(attrDBParameters.String(fmt.Sprintf("%v", args)))
	}
}

func recordError(span trace.Span, err error) {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

type Row struct {
	rows *sql.Rows
}

func (row *Row) Scan(dest ...any) error {
	defer row.rows.Close()

	if !row.rows.Next() {
		if err := row.rows.Err(); err != nil {
			return err
		}
		return sql.ErrNoRows
	}

	err := row.rows.Scan(dest...)
	if err != nil {
		return err
	}

	return row.rows.Close()
}

func extractOperation(query string) string {
	tokens := strings.Fields(query)
	if len(tokens) == 0 {
		return "UNKNOWN"
	}

	switch strings.ToUpper(tokens[0]) {
	case "SELECT", "INSERT", "UPDATE", "DELETE",
		"WITH", "CALL", "MERGE", "REPLACE", "TRUNCATE":
		return tokens[0]
	default:
		return "UNKNOWN"
	}
}
