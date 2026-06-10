package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var (
	ErrDBNil            = errors.New("sqlgotel: sql.DB não pode ser nil")
	ErrDBSystemRequired = errors.New("sqlgotel: WithDBSystem é obrigatório")
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

// ---- Configuração

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

// ---- Database

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

// ---- Assessores

func (database *Database) StatementRecording() bool {
	return database.config.statementRecording
}

func (database *Database) ParameterRecording() bool {
	return database.config.parameterRecording
}

func (database *Database) Unwrap() *sql.DB {
	return database.database
}

// ---- QueryContext

func (database *Database) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
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

// ---- ExecContext

func (database *Database) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
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

// ---- BeginTx

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

// ---- Helpers internos

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
			attrServerPort.String(fmt.Sprintf("%d", database.config.serverPort)),
		)
	}

	span.SetAttributes(attrs...)

	return ctx, span
}

func (database *Database) setQueryAttributes(span trace.Span, query string, args []interface{}) {
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

func extractOperation(query string) string {
	if len(query) < 6 {
		return "UNKNOWN"
	}

	switch query[:6] {
	case "SELECT", "select":
		return "SELECT"
	case "INSERT", "insert":
		return "INSERT"
	case "UPDATE", "update":
		return "UPDATE"
	case "DELETE", "delete":
		return "DELETE"
	default:
		return "UNKNOWN"
	}
}
