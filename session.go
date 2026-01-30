package sqlc

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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
	obs      *ObservabilityConfig
}

func NewSession(db *sql.DB, dialect Dialect, opts ...SessionOption) *Session {
	xdb := sqlx.NewDb(db, dialect.Name())
	s := &Session{
		db:       xdb,
		executor: xdb,
		dialect:  dialect,
		obs:      defaultObservabilityConfig(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// instrument wraps a database operation with tracing, logging, and metrics
func (s *Session) instrument(ctx context.Context, spanName, operation, query string, fn func() error) error {
	ctx, span := s.startSpan(ctx, spanName)
	defer span.End()

	start := time.Now()
	err := fn()
	duration := time.Since(start)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	span.SetAttributes(attribute.String("db.statement", query))
	s.logQuery(ctx, operation, query, duration, err)
	s.recordMetrics(ctx, operation, duration, err)

	return err
}

func (s *Session) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	var rows *sql.Rows
	err := s.instrument(ctx, "sqlc.Query", "query", query, func() error {
		var e error
		rows, e = s.executor.QueryContext(ctx, query, args...)
		return e
	})
	return rows, err
}

// QueryRow executes a query that returns at most one row.
// Note: Full observability (metrics, duration) is not available because
// the actual execution happens when Scan() is called on the returned Row.
func (s *Session) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	ctx, span := s.startSpan(ctx, "sqlc.QueryRow")
	defer span.End()
	span.SetAttributes(attribute.String("db.statement", query))

	// Log the query (without duration/error since execution is deferred)
	if s.obs.Logger != nil && s.obs.LogQueries {
		s.obs.Logger.DebugContext(ctx, "query row", "operation", "query_row", "query", query)
	}

	return s.executor.QueryRowContext(ctx, query, args...)
}

func (s *Session) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	var result sql.Result
	err := s.instrument(ctx, "sqlc.Exec", "exec", query, func() error {
		var e error
		result, e = s.executor.ExecContext(ctx, query, args...)
		return e
	})
	return result, err
}

func (s *Session) Select(ctx context.Context, dest any, query string, args ...any) error {
	return s.instrument(ctx, "sqlc.Select", "select", query, func() error {
		return s.executor.SelectContext(ctx, dest, query, args...)
	})
}

func (s *Session) Get(ctx context.Context, dest any, query string, args ...any) error {
	return s.instrument(ctx, "sqlc.Get", "get", query, func() error {
		return s.executor.GetContext(ctx, dest, query, args...)
	})
}

func (s *Session) Begin(ctx context.Context) (*Session, error) {
	ctx, span := s.startSpan(ctx, "sqlc.Begin")
	defer span.End()

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	// Return new Session where executor is the transaction
	return &Session{
		db:       s.db,
		executor: tx,
		dialect:  s.dialect,
		obs:      s.obs, // Inherit observability config
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
