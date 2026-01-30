package sqlc

import (
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
)

// Dialect abstracts database-specific SQL features
type Dialect interface {
	Name() string
	PlaceholderFormat() sq.PlaceholderFormat
	UpsertClause(tableName string, conflictCols []string, updateCols []string) string
}

// buildOnConflictUpsert generates ON CONFLICT ... DO UPDATE SET clause
// excludedPrefix controls casing: "EXCLUDED" for Postgres, "excluded" for SQLite
func buildOnConflictUpsert(conflictCols, updateCols []string, excludedPrefix string) string {
	if len(conflictCols) == 0 {
		return ""
	}
	conflictTarget := strings.Join(conflictCols, ", ")

	if len(updateCols) == 0 {
		return fmt.Sprintf("ON CONFLICT (%s) DO NOTHING", conflictTarget)
	}

	clause := fmt.Sprintf("ON CONFLICT (%s) DO UPDATE SET ", conflictTarget)
	updates := make([]string, len(updateCols))
	for i, col := range updateCols {
		updates[i] = fmt.Sprintf("%s=%s.%s", col, excludedPrefix, col)
	}
	return clause + strings.Join(updates, ", ")
}

type MySQLDialect struct{}

func (d *MySQLDialect) Name() string { return "mysql" }
func (d *MySQLDialect) PlaceholderFormat() sq.PlaceholderFormat {
	return sq.Question
}
func (d *MySQLDialect) UpsertClause(tableName string, conflictCols []string, updateCols []string) string {
	// ON DUPLICATE KEY UPDATE col1=VALUES(col1), col2=VALUES(col2)
	// MySQL determines conflict target automatically (PK or Unique)
	if len(updateCols) == 0 {
		return ""
	}
	clause := "ON DUPLICATE KEY UPDATE "
	updates := make([]string, len(updateCols))
	for i, col := range updateCols {
		updates[i] = fmt.Sprintf("%s=VALUES(%s)", col, col)
	}
	return clause + strings.Join(updates, ", ")
}

type PostgreSQLDialect struct{}

func (d *PostgreSQLDialect) Name() string { return "postgres" }
func (d *PostgreSQLDialect) PlaceholderFormat() sq.PlaceholderFormat {
	return sq.Dollar
}
func (d *PostgreSQLDialect) UpsertClause(tableName string, conflictCols []string, updateCols []string) string {
	return buildOnConflictUpsert(conflictCols, updateCols, "EXCLUDED")
}

type SQLiteDialect struct{}

func (d *SQLiteDialect) Name() string { return "sqlite3" }
func (d *SQLiteDialect) PlaceholderFormat() sq.PlaceholderFormat {
	return sq.Question
}
func (d *SQLiteDialect) UpsertClause(tableName string, conflictCols []string, updateCols []string) string {
	return buildOnConflictUpsert(conflictCols, updateCols, "excluded")
}
