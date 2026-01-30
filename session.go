package sqlc

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
)

// Executor defines the common database operations for both DB and Tx
type Executor interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
	GetContext(ctx context.Context, dest any, query string, args ...any) error
}

// Session manages the database connection and current transaction
type Session struct {
	db       *sqlx.DB // Underlying DB for starting transactions
	executor Executor // Current executor (DB or Tx)
	dialect  Dialect
}

func NewSession(db *sql.DB, dialect Dialect) *Session {
	xdb := sqlx.NewDb(db, dialect.Name())
	return &Session{
		db:       xdb,
		executor: xdb,
		dialect:  dialect,
	}
}

func (s *Session) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return s.executor.QueryContext(ctx, query, args...)
}

func (s *Session) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return s.executor.QueryRowContext(ctx, query, args...)
}

func (s *Session) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return s.executor.ExecContext(ctx, query, args...)
}

func (s *Session) Select(ctx context.Context, dest any, query string, args ...any) error {
	return s.executor.SelectContext(ctx, dest, query, args...)
}

func (s *Session) Get(ctx context.Context, dest any, query string, args ...any) error {
	return s.executor.GetContext(ctx, dest, query, args...)
}

func (s *Session) Begin(ctx context.Context) (*Session, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	// Return new Session where executor is the transaction
	return &Session{
		db:       s.db,
		executor: tx,
		dialect:  s.dialect,
	}, nil
}

func (s *Session) Commit() error {
	if tx, ok := s.executor.(*sqlx.Tx); ok {
		return tx.Commit()
	}
	return sql.ErrTxDone
}

func (s *Session) Rollback() error {
	if tx, ok := s.executor.(*sqlx.Tx); ok {
		return tx.Rollback()
	}
	return sql.ErrTxDone
}

// Transaction executes a function within a transaction
func (s *Session) Transaction(ctx context.Context, fn func(txSession *Session) error) (err error) {
	// Check if already in transaction
	if _, ok := s.executor.(*sqlx.Tx); ok {
		return fn(s)
	}

	txSession, err := s.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = txSession.Rollback()
			panic(p)
		} else if err != nil {
			_ = txSession.Rollback()
		}
	}()

	err = fn(txSession)
	if err != nil {
		return err
	}

	return txSession.Commit()
}
