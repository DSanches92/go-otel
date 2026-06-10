package sql

import (
	"context"
	"database/sql"
)

type Transaction struct {
	transaction *sql.Tx
	database    *Database
}

func (transaction *Transaction) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	ctx, span := transaction.database.startSpan(ctx, "sql.query")
	defer span.End()

	transaction.database.setQueryAttributes(span, query, args)

	rows, err := transaction.transaction.QueryContext(ctx, query, args...)
	if err != nil {
		recordError(span, err)
		return nil, err
	}

	return rows, nil
}

func (transaction *Transaction) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	ctx, span := transaction.database.startSpan(ctx, "sql.exec")
	defer span.End()

	transaction.database.setQueryAttributes(span, query, args)

	result, err := transaction.transaction.ExecContext(ctx, query, args...)
	if err != nil {
		recordError(span, err)
		return nil, err
	}

	return result, nil
}

func (transaction *Transaction) Commit(ctx context.Context) error {
	_, span := transaction.database.startSpan(ctx, "sql.transaction.commit")
	defer span.End()

	err := transaction.transaction.Commit()
	if err != nil {
		recordError(span, err)
		return err
	}

	return nil
}

func (transaction *Transaction) Rollback(ctx context.Context) error {
	_, span := transaction.database.startSpan(ctx, "sql.transaction.rollback")
	defer span.End()

	err := transaction.transaction.Rollback()
	if err != nil && err != sql.ErrTxDone {
		recordError(span, err)
		return err
	}

	return nil
}

func (transaction *Transaction) Unwrap() *sql.Tx {
	return transaction.transaction
}
