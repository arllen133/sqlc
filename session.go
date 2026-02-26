// Package sqlc provides a type-safe ORM library using generics and code generation.
// This file implements database session management, including connection management,
// transaction handling, and observability integration.
package sqlc

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// Executor defines the common database operations for both DB and Tx.
// This interface is implemented by both *sqlx.DB and *sqlx.Tx, allowing Session
// to seamlessly switch between regular queries and transactions without modifying
// calling code.
//
// Implementations:
//   - *sqlx.DB: for regular database operations
//   - *sqlx.Tx: for transactional database operations
type Executor interface {
	// QueryContext executes a query and returns multiple rows
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	// ExecContext executes a write operation (INSERT/UPDATE/DELETE) and returns affected rows
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	// QueryRowContext executes a query expecting a single row result
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	// SelectContext executes a query and scans results into a struct slice
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
	// GetContext executes a query and scans a single row result into a struct
	GetContext(ctx context.Context, dest any, query string, args ...any) error
}

// Session manages the database connection and current transaction state.
// It is the core component of sqlc ORM, responsible for:
//   - Database connection management
//   - Transaction lifecycle management
//   - SQL dialect adaptation
//   - Observability integration (logging, tracing, metrics)
//
// Session supports two modes:
//   - Regular mode: executor points to *sqlx.DB
//   - Transaction mode: executor points to *sqlx.Tx
//
// Usage example:
//
//	// Create session
//	session := sqlc.NewSession(db, sqlc.MySQL)
//
//	// Regular query
//	users, err := userRepo.Query().Find(ctx)
//
//	// Transaction operation
//	err := session.Transaction(ctx, func(txSession *Session) error {
//	    if err := userRepo.WithSession(txSession).Create(ctx, user); err != nil {
//	        return err // Auto rollback
//	    }
//	    return nil // Auto commit
//	})
type Session struct {
	db       *sqlx.DB             // Underlying database connection for starting transactions
	executor Executor             // Current executor (DB or Tx)
	dialect  Dialect              // Database dialect for handling SQL differences
	obs      *ObservabilityConfig // Observability configuration (logging, tracing, metrics)
}

// NewSession creates a new database session.
// This is the entry point for using sqlc ORM.
//
// Parameters:
//   - db: Standard library *sql.DB connection pool
//   - dialect: Database dialect (MySQLDialect/PostgreSQLDialect/SQLiteDialect)
//   - opts: Optional session configuration options (logging, tracing, metrics, etc.)
//
// Returns:
//   - *Session: Initialized session instance
//
// Example:
//
//	// Basic usage
//	session := sqlc.NewSession(db, sqlc.MySQL)
//
//	// With logging
//	session := sqlc.NewSession(db, sqlc.MySQL,
//	    sqlc.WithLogger(slog.Default()),
//	    sqlc.WithQueryLogging(true),
//	)
//
//	// With tracing and metrics
//	session := sqlc.NewSession(db, sqlc.PostgreSQL,
//	    sqlc.WithDefaultTracer(),
//	    sqlc.WithDefaultMeter(),
//	)
func NewSession(db *sql.DB, dialect Dialect, opts ...SessionOption) *Session {
	// Convert standard sql.DB to sqlx.DB for enhanced functionality
	xdb := sqlx.NewDb(db, dialect.Name())

	// Create session instance with default configuration
	s := &Session{
		db:       xdb,
		executor: xdb, // Default to DB as executor
		dialect:  dialect,
		obs:      defaultObservabilityConfig(),
	}

	// Apply all optional configurations
	for _, opt := range opts {
		opt(s)
	}

	return s
}

// instrument wraps a database operation with observability.
// This is an internal method that provides for each database operation:
//   - OpenTelemetry tracing (span creation, error recording)
//   - Structured logging (query statement, execution time, error info)
//   - Performance metrics (operation count, latency distribution, error rate)
//
// Parameters:
//   - ctx: Context for propagating trace information and cancellation signals
//   - spanName: Trace span name (e.g., "sqlc.Query")
//   - operation: Operation type for logging and metrics (e.g., "select", "exec")
//   - query: SQL query statement
//   - fn: Actual database operation function
//
// Returns:
//   - error: Wrapped error if any
//
// This method ensures all database operations have consistent observability,
// making it easy to monitor and debug in production environments.
func (s *Session) instrument(ctx context.Context, spanName, operation, query string, fn func() error) error {
	// Start trace span
	ctx, span := s.startSpan(ctx, spanName)
	defer span.End()

	// Record start time
	start := time.Now()

	// Execute actual database operation
	err := fn()

	// Calculate execution duration
	duration := time.Since(start)

	// If error exists, record it in span
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	// Add SQL statement to span attributes
	span.SetAttributes(attribute.String("db.statement", query))

	// Record logs
	s.logQuery(ctx, operation, query, duration, err)

	// Record metrics
	s.recordMetrics(ctx, operation, duration, err)

	return err
}

// Query executes a SQL query that returns multiple rows.
// Suitable for scenarios requiring iteration over large result sets.
//
// Parameters:
//   - ctx: Context supporting cancellation and timeout
//   - query: SQL query statement (using placeholders)
//   - args: Query parameters
//
// Returns:
//   - *sql.Rows: Query result set, caller must call Close()
//   - error: Query error
//
// Example:
//
//	rows, err := session.Query(ctx, "SELECT * FROM users WHERE age > ?", 18)
//	if err != nil {
//	    return err
//	}
//	defer rows.Close()
//
//	for rows.Next() {
//	    var user User
//	    if err := rows.Scan(&user.ID, &user.Name); err != nil {
//	        return err
//	    }
//	    // Process user
//	}
func (s *Session) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	var rows *sql.Rows
	err := s.instrument(ctx, "sqlc.Query", "query", query, func() error {
		var e error
		rows, e = s.executor.QueryContext(ctx, query, args...)
		return e
	})
	return rows, err
}

// QueryRow executes a SQL query expecting at most one row.
// The returned *sql.Row needs to call Scan() method to retrieve data.
//
// Note: Since actual execution happens when Scan() is called, this method cannot
// provide complete observability (execution duration, error statistics).
// For complete observability, use the Get() method instead.
//
// Parameters:
//   - ctx: Context supporting cancellation and timeout
//   - query: SQL query statement (using placeholders)
//   - args: Query parameters
//
// Returns:
//   - *sql.Row: Single row result
//
// Example:
//
//	var name string
//	err := session.QueryRow(ctx, "SELECT name FROM users WHERE id = ?", 1).Scan(&name)
//	if err == sql.ErrNoRows {
//	    // Record not found
//	}
func (s *Session) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	// Start trace span
	ctx, span := s.startSpan(ctx, "sqlc.QueryRow")
	defer span.End()
	span.SetAttributes(attribute.String("db.statement", query))

	// Log query (without duration/error since execution is deferred to Scan())
	if s.obs.Logger != nil && s.obs.LogQueries {
		s.obs.Logger.DebugContext(ctx, "query row",
			"operation", "query_row",
			"query", query,
		)
	}

	return s.executor.QueryRowContext(ctx, query, args...)
}

// Exec executes a SQL statement that doesn't return rows (INSERT/UPDATE/DELETE).
// Suitable for data modification operations.
//
// Parameters:
//   - ctx: Context supporting cancellation and timeout
//   - query: SQL statement (using placeholders)
//   - args: Statement parameters
//
// Returns:
//   - sql.Result: Result containing affected rows and last insert ID
//   - error: Execution error
//
// Example:
//
//	result, err := session.Exec(ctx,
//	    "UPDATE users SET name = ? WHERE id = ?",
//	    "New Name", 1,
//	)
//	if err != nil {
//	    return err
//	}
//	rowsAffected, _ := result.RowsAffected()
func (s *Session) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	var result sql.Result
	err := s.instrument(ctx, "sqlc.Exec", "exec", query, func() error {
		var e error
		result, e = s.executor.ExecContext(ctx, query, args...)
		return e
	})
	return result, err
}

// Select executes a query and scans results into a struct slice.
// This is the recommended way to query multiple rows and map to models.
//
// Parameters:
//   - ctx: Context supporting cancellation and timeout
//   - dest: Pointer to target slice (e.g., *[]*User)
//   - query: SQL query statement (using placeholders)
//   - args: Query parameters
//
// Returns:
//   - error: Query or scan error
//
// Example:
//
//	var users []*User
//	err := session.Select(ctx, &users,
//	    "SELECT * FROM users WHERE age > ? ORDER BY name",
//	    18,
//	)
func (s *Session) Select(ctx context.Context, dest any, query string, args ...any) error {
	return s.instrument(ctx, "sqlc.Select", "select", query, func() error {
		return s.executor.SelectContext(ctx, dest, query, args...)
	})
}

// Get executes a query and scans a single row result into a struct.
// This is the recommended way to query a single record and map to a model.
//
// Parameters:
//   - ctx: Context supporting cancellation and timeout
//   - dest: Pointer to target struct (e.g., *User)
//   - query: SQL query statement (using placeholders)
//   - args: Query parameters
//
// Returns:
//   - error: Query or scan error (sql.ErrNoRows indicates not found)
//
// Example:
//
//	var user User
//	err := session.Get(ctx, &user,
//	    "SELECT * FROM users WHERE id = ?",
//	    1,
//	)
//	if err == sql.ErrNoRows {
//	    // User not found
//	}
func (s *Session) Get(ctx context.Context, dest any, query string, args ...any) error {
	return s.instrument(ctx, "sqlc.Get", "get", query, func() error {
		return s.executor.GetContext(ctx, dest, query, args...)
	})
}

// Begin starts a new transaction.
// Returns a new Session instance with the executor being the transaction object.
//
// Note: You need to manually call Commit() or Rollback() after use.
// For automatic transaction management, use the Transaction() method instead.
//
// Parameters:
//   - ctx: Context supporting cancellation and timeout
//
// Returns:
//   - *Session: New session instance bound to the transaction
//   - error: Error starting transaction
//
// Example:
//
//	txSession, err := session.Begin(ctx)
//	if err != nil {
//	    return err
//	}
//	defer txSession.Rollback() // Safety measure, rollback on committed transaction is no-op
//
//	// Execute transaction operations...
//
//	if err := txSession.Commit(); err != nil {
//	    return err
//	}
func (s *Session) Begin(ctx context.Context) (*Session, error) {
	// Start trace span
	ctx, span := s.startSpan(ctx, "sqlc.Begin")
	defer span.End()

	// Begin transaction
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	// Return new Session with transaction as executor
	// This ensures all subsequent operations are in the same transaction
	return &Session{
		db:       s.db,      // Keep reference to original DB for nested transactions
		executor: tx,        // Use transaction as executor
		dialect:  s.dialect, // Inherit dialect configuration
		obs:      s.obs,     // Inherit observability configuration
	}, nil
}

// Commit commits the current transaction.
// Only effective in transaction mode (after calling Begin()).
//
// Returns:
//   - error: Commit error, returns sql.ErrTxDone if not in a transaction
//
// Example:
//
//	txSession, _ := session.Begin(ctx)
//	// ... execute operations
//	if err := txSession.Commit(); err != nil {
//	    log.Error("commit failed", "error", err)
//	}
func (s *Session) Commit() error {
	// Check if in a transaction
	if tx, ok := s.executor.(*sqlx.Tx); ok {
		return tx.Commit()
	}
	return sql.ErrTxDone
}

// Rollback rolls back the current transaction.
// Only effective in transaction mode (after calling Begin()).
//
// Returns:
//   - error: Rollback error, returns sql.ErrTxDone if not in a transaction
//
// Example:
//
//	txSession, _ := session.Begin(ctx)
//	// ... execute operations
//	if err := txSession.Rollback(); err != nil {
//	    log.Error("rollback failed", "error", err)
//	}
func (s *Session) Rollback() error {
	// Check if in a transaction
	if tx, ok := s.executor.(*sqlx.Tx); ok {
		return tx.Rollback()
	}
	return sql.ErrTxDone
}

// Transaction executes a function within a transaction with automatic commit and rollback.
// This is the recommended way to execute transactions, providing:
//   - Automatic commit: Commits automatically when function returns successfully
//   - Automatic rollback: Rolls back automatically when function returns error or panics
//   - Nesting support: If already in a transaction, executes function directly (no nested transaction)
//
// Parameters:
//   - ctx: Context supporting cancellation and timeout
//   - fn: Transaction function, receives transaction session and returns error
//
// Returns:
//   - error: Function error or commit error
//
// Example:
//
//	err := session.Transaction(ctx, func(txSession *Session) error {
//	    // Create user in transaction
//	    userRepo := sqlc.NewRepository[models.User](txSession)
//	    if err := userRepo.Create(ctx, user); err != nil {
//	        return err // Auto rollback
//	    }
//
//	    // Create order in transaction
//	    orderRepo := sqlc.NewRepository[models.Order](txSession)
//	    if err := orderRepo.Create(ctx, order); err != nil {
//	        return err // Auto rollback
//	    }
//
//	    return nil // Auto commit
//	})
//
//	if err != nil {
//	    log.Error("transaction failed", "error", err)
//	}
func (s *Session) Transaction(ctx context.Context, fn func(txSession *Session) error) (err error) {
	// Check if already in a transaction
	// If so, execute function directly to avoid nested transactions
	if _, ok := s.executor.(*sqlx.Tx); ok {
		return fn(s)
	}

	// Begin new transaction
	txSession, err := s.Begin(ctx)
	if err != nil {
		return err
	}

	// Use defer to ensure transaction is always handled (commit or rollback)
	defer func() {
		// Handle panic: rollback and re-panic
		if p := recover(); p != nil {
			_ = txSession.Rollback()
			panic(p)
		}

		// Handle error: rollback transaction
		if err != nil {
			_ = txSession.Rollback()
		}
	}()

	// Execute user function
	err = fn(txSession)
	if err != nil {
		return err
	}

	// Function succeeded, commit transaction
	return txSession.Commit()
}
