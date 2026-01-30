package json

import "github.com/arllen133/sqlc/clause"

// JSONDialect defines the interface for database-specific JSON operations.
// Each database (MySQL, PostgreSQL, SQLite) implements this interface
// to generate the correct SQL syntax for JSON operations.
type JSONDialect interface {
	// Name returns the dialect name (e.g., "mysql", "postgres", "sqlite")
	Name() string

	// ExtractPath generates SQL for extracting a value at a JSON path.
	// Returns the SQL fragment (e.g., "JSON_EXTRACT(col, ?)" for MySQL)
	ExtractPath(column, path string) (sql string, vars []any)

	// PathEq generates SQL for checking if a JSON path equals a value.
	// Returns the SQL and variables for parameterized queries.
	PathEq(column, path string, value any) (sql string, vars []any)

	// PathNeq generates SQL for checking if a JSON path does not equal a value.
	PathNeq(column, path string, value any) (sql string, vars []any)

	// PathGt generates SQL for checking if a JSON path is greater than a value.
	PathGt(column, path string, value any) (sql string, vars []any)

	// PathGte generates SQL for checking if a JSON path is greater than or equal to a value.
	PathGte(column, path string, value any) (sql string, vars []any)

	// PathLt generates SQL for checking if a JSON path is less than a value.
	PathLt(column, path string, value any) (sql string, vars []any)

	// PathLte generates SQL for checking if a JSON path is less than or equal to a value.
	PathLte(column, path string, value any) (sql string, vars []any)

	// Contains generates SQL for checking if JSON contains a value.
	Contains(column string, value any, path string) (sql string, vars []any)

	// SetPath generates SQL for setting a value at a JSON path.
	// Returns the SQL expression for use in UPDATE statements.
	SetPath(column, path string, value any) clause.Expr

	// SetMultiplePaths generates SQL for setting multiple path-value pairs.
	// This is used by the SetBuilder for batch updates.
	SetMultiplePaths(column string, paths []string, values []any) clause.Expr

	// RemovePath generates SQL for removing a JSON path.
	RemovePath(column, path string) clause.Expr

	// RemoveMultiplePaths generates SQL for removing multiple JSON paths.
	RemoveMultiplePaths(column string, paths []string) clause.Expr

	// MergePatch generates SQL for RFC 7396 Merge Patch.
	// (MySQL: JSON_MERGE_PATCH, Postgres: ||, SQLite: json_patch)
	MergePatch(column string, value any) clause.Expr

	// MergePreserve generates SQL for merging with array preservation (Legacy/Concat).
	// (MySQL: JSON_MERGE_PRESERVE, Postgres: ||)
	MergePreserve(column string, value any) clause.Expr
}

// Dialect instances
var (
	MySQL    JSONDialect = &mysqlDialect{}
	Postgres JSONDialect = &postgresDialect{}
	SQLite   JSONDialect = &sqliteDialect{}
)

// dialectRegistry holds the current default dialect
var defaultDialect JSONDialect = MySQL

// SetDefaultDialect sets the default JSON dialect for operations.
func SetDefaultDialect(d JSONDialect) {
	defaultDialect = d
}

// DefaultDialect returns the current default JSON dialect.
func DefaultDialect() JSONDialect {
	return defaultDialect
}

// DialectByName returns a JSONDialect by its name.
func DialectByName(name string) JSONDialect {
	switch name {
	case "mysql":
		return MySQL
	case "postgres":
		return Postgres
	case "sqlite3", "sqlite":
		return SQLite
	default:
		return MySQL
	}
}
