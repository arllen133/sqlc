// Package sqlc provides a type-safe ORM library using generics and code generation.
// This file implements database dialect abstraction to handle SQL differences between databases.
//
// Dialect is the key abstraction for sqlc to support multiple databases, responsible for:
//   - Database identification (MySQL, PostgreSQL, SQLite)
//   - Placeholder format (? vs $1, $2)
//   - Upsert syntax (ON DUPLICATE KEY vs ON CONFLICT)
//
// Currently supported databases:
//   - MySQL 5.7+
//   - PostgreSQL 12+
//   - SQLite 3.24+ (with UPSERT support)
//
// Usage example:
//
//	// MySQL
//	session := sqlc.NewSession(db, sqlc.MySQLDialect{})
//
//	// PostgreSQL
//	session := sqlc.NewSession(db, sqlc.PostgreSQLDialect{})
//
//	// SQLite
//	session := sqlc.NewSession(db, sqlc.SQLiteDialect{})
package sqlc

import (
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
)

var (
	SQLite     = SQLiteDialect{}
	MySQL      = MySQLDialect{}
	PostgreSQL = PostgreSQLDialect{}
)

// Dialect abstracts database-specific SQL features.
// Different databases have SQL syntax differences, and the Dialect interface provides a unified abstraction layer.
//
// Main differences handled:
//   - Placeholder format: MySQL/SQLite use ?, PostgreSQL uses $1, $2
//   - Upsert syntax: MySQL uses ON DUPLICATE KEY UPDATE, PostgreSQL/SQLite use ON CONFLICT
//   - Other database-specific features (future expansion)
//
// Implementations:
//   - MySQLDialect: MySQL dialect
//   - PostgreSQLDialect: PostgreSQL dialect
//   - SQLiteDialect: SQLite dialect
type Dialect interface {
	// Name returns the database type name.
	// Used for logging, metrics collection, and driver selection.
	//
	// Returns:
	//   - "mysql" for MySQL
	//   - "postgres" for PostgreSQL
	//   - "sqlite3" for SQLite
	Name() string

	// PlaceholderFormat returns the placeholder format used by the database.
	// Squirrel uses this format to generate parameterized queries.
	//
	// Common formats:
	//   - sq.Question: ? placeholder (MySQL, SQLite)
	//   - sq.Dollar: $1, $2 placeholders (PostgreSQL)
	PlaceholderFormat() sq.PlaceholderFormat

	// UpsertClause generates the SQL clause for insert or update (Upsert).
	// This handles database-specific syntax for "update if exists, insert if not".
	//
	// Parameters:
	//   - tableName: Table name
	//   - conflictCols: Conflict detection columns (unique constraint or primary key)
	//   - updateCols: Columns to update when conflict occurs
	//
	// Returns:
	//   - string: Complete Upsert clause (e.g., "ON CONFLICT ... DO UPDATE SET ...")
	//
	// Example output:
	//   MySQL: "ON DUPLICATE KEY UPDATE name=VALUES(name), email=VALUES(email)"
	//   PostgreSQL: "ON CONFLICT (email) DO UPDATE SET name=EXCLUDED.name"
	//   SQLite: "ON CONFLICT (email) DO UPDATE SET name=excluded.name"
	UpsertClause(tableName string, conflictCols []string, updateCols []string) string
}

// buildOnConflictUpsert generates ON CONFLICT ... DO UPDATE SET clause.
// This is the Upsert syntax used by PostgreSQL and SQLite.
//
// Syntax format:
//
//	ON CONFLICT (conflict_columns) DO UPDATE SET col1=EXCLUDED.col1, col2=EXCLUDED.col2
//	or
//	ON CONFLICT (conflict_columns) DO NOTHING
//
// Parameters:
//   - conflictCols: Conflict detection columns (e.g., ["email"] or ["user_id", "product_id"])
//   - updateCols: Columns to update when conflict occurs (e.g., ["name", "updated_at"])
//   - excludedPrefix: Reference to EXCLUDED table (PostgreSQL: "EXCLUDED", SQLite: "excluded")
//
// Returns:
//   - string: Complete ON CONFLICT clause
//
// Note:
//   - If conflictCols is empty, returns empty string (invalid configuration)
//   - If updateCols is empty, generates DO NOTHING (no update)
//
// Example:
//
//	// PostgreSQL
//	buildOnConflictUpsert([]string{"email"}, []string{"name", "updated_at"}, "EXCLUDED")
//	// Returns: "ON CONFLICT (email) DO UPDATE SET name=EXCLUDED.name,updated_at=EXCLUDED.updated_at"
//
//	// SQLite
//	buildOnConflictUpsert([]string{"email"}, []string{"name"}, "excluded")
//	// Returns: "ON CONFLICT (email) DO UPDATE SET name=excluded.name"
func buildOnConflictUpsert(conflictCols, updateCols []string, excludedPrefix string) string {
	// No conflict columns, cannot generate valid Upsert clause
	if len(conflictCols) == 0 {
		return ""
	}

	// Build conflict target: ON CONFLICT (col1, col2, ...)
	conflictTarget := strings.Join(conflictCols, ", ")

	// If no update columns, generate DO NOTHING
	if len(updateCols) == 0 {
		return fmt.Sprintf("ON CONFLICT (%s) DO NOTHING", conflictTarget)
	}

	// Build DO UPDATE SET clause
	// Format: col1=EXCLUDED.col1, col2=EXCLUDED.col2, ...
	clause := fmt.Sprintf("ON CONFLICT (%s) DO UPDATE SET ", conflictTarget)
	updates := make([]string, len(updateCols))
	for i, col := range updateCols {
		// EXCLUDED is a special table reference containing the proposed insert row
		// PostgreSQL uses uppercase EXCLUDED, SQLite uses lowercase excluded
		updates[i] = fmt.Sprintf("%s=%s.%s", col, excludedPrefix, col)
	}

	return clause + strings.Join(updates, ", ")
}

// MySQLDialect implements MySQL database dialect.
//
// MySQL features:
//   - Uses ? as placeholder
//   - Uses ON DUPLICATE KEY UPDATE syntax for Upsert
//   - Automatically detects conflict target (primary key or unique key)
//
// Usage example:
//
//	session := sqlc.NewSession(db, sqlc.MySQLDialect{})
//
// Note:
//   - Upsert doesn't need to specify conflict columns, MySQL automatically detects by primary key or unique key
//   - VALUES() function references proposed insert values
type MySQLDialect struct{}

// Name returns the MySQL dialect name.
func (d *MySQLDialect) Name() string { return "mysql" }

// PlaceholderFormat returns MySQL's placeholder format (?).
func (d *MySQLDialect) PlaceholderFormat() sq.PlaceholderFormat {
	return sq.Question
}

// UpsertClause generates MySQL's Upsert clause.
// MySQL uses ON DUPLICATE KEY UPDATE syntax.
//
// Syntax format:
//
//	ON DUPLICATE KEY UPDATE col1=VALUES(col1), col2=VALUES(col2)
//
// MySQL features:
//   - Doesn't need to specify conflict columns (auto-detects primary key or unique key)
//   - VALUES(col) function references proposed insert values
//   - If updateCols is empty, returns empty string (cannot implement DO NOTHING)
//
// Parameters:
//   - tableName: Table name (not used by MySQL, but kept for interface compatibility)
//   - conflictCols: Conflict columns (not used by MySQL, auto-detects)
//   - updateCols: Columns to update
//
// Returns:
//   - string: ON DUPLICATE KEY UPDATE clause
//
// Example:
//
//	dialect.UpsertClause("users", []string{"email"}, []string{"name", "updated_at"})
//	// Returns: "ON DUPLICATE KEY UPDATE name=VALUES(name),updated_at=VALUES(updated_at)"
func (d *MySQLDialect) UpsertClause(tableName string, conflictCols []string, updateCols []string) string {
	// MySQL doesn't support DO NOTHING, skip if no update columns
	if len(updateCols) == 0 {
		return ""
	}

	// Build ON DUPLICATE KEY UPDATE clause
	clause := "ON DUPLICATE KEY UPDATE "
	updates := make([]string, len(updateCols))
	for i, col := range updateCols {
		// VALUES(col) references proposed insert values
		updates[i] = fmt.Sprintf("%s=VALUES(%s)", col, col)
	}

	return clause + strings.Join(updates, ", ")
}

// PostgreSQLDialect implements PostgreSQL database dialect.
//
// PostgreSQL features:
//   - Uses $1, $2, $3 as placeholders
//   - Uses ON CONFLICT ... DO UPDATE syntax for Upsert
//   - Requires explicitly specifying conflict columns
//   - Uses EXCLUDED table to reference proposed insert values
//
// Usage example:
//
//	session := sqlc.NewSession(db, sqlc.PostgreSQLDialect{})
//
// Note:
//   - Upsert needs to specify conflict columns (ON CONFLICT (col))
//   - EXCLUDED is a special table name, must be uppercase
//   - Supports DO NOTHING (no update)
type PostgreSQLDialect struct{}

// Name returns the PostgreSQL dialect name.
func (d *PostgreSQLDialect) Name() string { return "postgres" }

// PlaceholderFormat returns PostgreSQL's placeholder format ($1, $2, ...).
func (d *PostgreSQLDialect) PlaceholderFormat() sq.PlaceholderFormat {
	return sq.Dollar
}

// UpsertClause generates PostgreSQL's Upsert clause.
// PostgreSQL uses ON CONFLICT ... DO UPDATE syntax.
//
// Syntax format:
//
//	ON CONFLICT (conflict_columns) DO UPDATE SET col1=EXCLUDED.col1, col2=EXCLUDED.col2
//	ON CONFLICT (conflict_columns) DO NOTHING
//
// PostgreSQL features:
//   - Requires explicitly specifying conflict columns
//   - EXCLUDED table references proposed insert values (must be uppercase)
//   - Supports DO NOTHING
//
// Parameters:
//   - tableName: Table name (not used by PostgreSQL)
//   - conflictCols: Conflict detection columns
//   - updateCols: Columns to update
//
// Returns:
//   - string: ON CONFLICT clause
//
// Example:
//
//	dialect.UpsertClause("users", []string{"email"}, []string{"name", "updated_at"})
//	// Returns: "ON CONFLICT (email) DO UPDATE SET name=EXCLUDED.name,updated_at=EXCLUDED.updated_at"
func (d *PostgreSQLDialect) UpsertClause(tableName string, conflictCols []string, updateCols []string) string {
	return buildOnConflictUpsert(conflictCols, updateCols, "EXCLUDED")
}

// SQLiteDialect implements SQLite database dialect.
//
// SQLite features:
//   - Uses ? as placeholder
//   - Uses ON CONFLICT ... DO UPDATE syntax for Upsert (version 3.24+)
//   - Requires explicitly specifying conflict columns
//   - Uses excluded table to reference proposed insert values (lowercase)
//
// Usage example:
//
//	session := sqlc.NewSession(db, sqlc.SQLiteDialect{})
//
// Note:
//   - Upsert requires SQLite 3.24+ version
//   - excluded table name is lowercase (different from PostgreSQL)
//   - Supports DO NOTHING
//   - Commonly used in testing and development environments
type SQLiteDialect struct{}

// Name returns the SQLite dialect name.
func (d *SQLiteDialect) Name() string { return "sqlite3" }

// PlaceholderFormat returns SQLite's placeholder format (?).
func (d *SQLiteDialect) PlaceholderFormat() sq.PlaceholderFormat {
	return sq.Question
}

// UpsertClause generates SQLite's Upsert clause.
// SQLite uses ON CONFLICT ... DO UPDATE syntax (version 3.24+).
//
// Syntax format:
//
//	ON CONFLICT (conflict_columns) DO UPDATE SET col1=excluded.col1, col2=excluded.col2
//	ON CONFLICT (conflict_columns) DO NOTHING
//
// SQLite features:
//   - Requires explicitly specifying conflict columns
//   - excluded table references proposed insert values (lowercase)
//   - Supports DO NOTHING
//   - Requires SQLite 3.24+ version
//
// Parameters:
//   - tableName: Table name (not used by SQLite)
//   - conflictCols: Conflict detection columns
//   - updateCols: Columns to update
//
// Returns:
//   - string: ON CONFLICT clause
//
// Example:
//
//	dialect.UpsertClause("users", []string{"email"}, []string{"name", "updated_at"})
//	// Returns: "ON CONFLICT (email) DO UPDATE SET name=excluded.name,updated_at=excluded.updated_at"
func (d *SQLiteDialect) UpsertClause(tableName string, conflictCols []string, updateCols []string) string {
	// SQLite uses lowercase "excluded", different from PostgreSQL's "EXCLUDED"
	return buildOnConflictUpsert(conflictCols, updateCols, "excluded")
}
