package json

import (
	"fmt"
	"strings"

	"github.com/arllen133/sqlc/clause"
)

type sqliteDialect struct{}

func (s *sqliteDialect) Name() string { return "sqlite3" }

func (s *sqliteDialect) ExtractPath(column, path string) (string, []any) {
	return fmt.Sprintf("json_extract(%s, ?)", column), []any{path}
}

func (s *sqliteDialect) PathEq(column, path string, value any) (string, []any) {
	return fmt.Sprintf("json_extract(%s, ?) = ?", column), []any{path, marshalValue(value)}
}

func (s *sqliteDialect) PathNeq(column, path string, value any) (string, []any) {
	return fmt.Sprintf("json_extract(%s, ?) != ?", column), []any{path, marshalValue(value)}
}

func (s *sqliteDialect) PathGt(column, path string, value any) (string, []any) {
	return fmt.Sprintf("json_extract(%s, ?) > ?", column), []any{path, marshalValue(value)}
}

func (s *sqliteDialect) PathGte(column, path string, value any) (string, []any) {
	return fmt.Sprintf("json_extract(%s, ?) >= ?", column), []any{path, marshalValue(value)}
}

func (s *sqliteDialect) PathLt(column, path string, value any) (string, []any) {
	return fmt.Sprintf("json_extract(%s, ?) < ?", column), []any{path, marshalValue(value)}
}

func (s *sqliteDialect) PathLte(column, path string, value any) (string, []any) {
	return fmt.Sprintf("json_extract(%s, ?) <= ?", column), []any{path, marshalValue(value)}
}

func (s *sqliteDialect) Contains(column string, value any, path string) (string, []any) {
	// SQLite doesn't have a direct JSON_CONTAINS, use json_extract + LIKE or similar
	// For now, we use a simple approach
	if path != "" {
		return fmt.Sprintf("json_extract(%s, ?) LIKE ?", column), []any{path, "%" + fmt.Sprint(value) + "%"}
	}
	return fmt.Sprintf("json(%s) LIKE ?", column), []any{"%" + fmt.Sprint(value) + "%"}
}

func (s *sqliteDialect) SetPath(column, path string, value any) clause.Expr {
	return clause.Expr{
		SQL:  fmt.Sprintf("json_set(%s, ?, ?)", column),
		Vars: []any{path, marshalValue(value)},
	}
}

func (s *sqliteDialect) SetMultiplePaths(column string, paths []string, values []any) clause.Expr {
	if len(paths) == 0 {
		return clause.Expr{}
	}

	placeholders := make([]string, len(paths)*2)
	vars := make([]any, 0, len(paths)*2)
	for i, path := range paths {
		placeholders[i*2] = "?"
		placeholders[i*2+1] = "?"
		vars = append(vars, path, marshalValue(values[i]))
	}

	return clause.Expr{
		SQL:  fmt.Sprintf("json_set(%s, %s)", column, strings.Join(placeholders, ", ")),
		Vars: vars,
	}
}

func (s *sqliteDialect) RemovePath(column, path string) clause.Expr {
	return clause.Expr{
		SQL:  fmt.Sprintf("json_remove(%s, ?)", column),
		Vars: []any{path},
	}
}

func (s *sqliteDialect) RemoveMultiplePaths(column string, paths []string) clause.Expr {
	if len(paths) == 0 {
		return clause.Expr{}
	}

	placeholders := make([]string, len(paths))
	vars := make([]any, len(paths))
	for i, path := range paths {
		placeholders[i] = "?"
		vars[i] = path
	}

	return clause.Expr{
		SQL:  fmt.Sprintf("json_remove(%s, %s)", column, strings.Join(placeholders, ", ")),
		Vars: vars,
	}
}

func (s *sqliteDialect) MergePatch(column string, value any) clause.Expr {
	return clause.Expr{
		SQL:  fmt.Sprintf("json_patch(%s, ?)", column),
		Vars: []any{marshalValue(value)},
	}
}

func (s *sqliteDialect) MergePreserve(column string, value any) clause.Expr {
	// SQLite only supports JSON Merge Patch (RFC 7396) via json_patch.
	// We fallback to json_patch for MergePreserve.
	return clause.Expr{
		SQL:  fmt.Sprintf("json_patch(%s, ?)", column),
		Vars: []any{marshalValue(value)},
	}
}
