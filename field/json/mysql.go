package json

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/arllen133/sqlc/clause"
)

type mysqlDialect struct{}

func (m *mysqlDialect) Name() string { return "mysql" }

func (m *mysqlDialect) ExtractPath(column, path string) (string, []any) {
	return fmt.Sprintf("JSON_EXTRACT(%s, ?)", column), []any{path}
}

func (m *mysqlDialect) PathEq(column, path string, value any) (string, []any) {
	return fmt.Sprintf("JSON_EXTRACT(%s, ?) = ?", column), []any{path, marshalValue(value)}
}

func (m *mysqlDialect) PathNeq(column, path string, value any) (string, []any) {
	return fmt.Sprintf("JSON_EXTRACT(%s, ?) != ?", column), []any{path, marshalValue(value)}
}

func (m *mysqlDialect) PathGt(column, path string, value any) (string, []any) {
	return fmt.Sprintf("JSON_EXTRACT(%s, ?) > ?", column), []any{path, marshalValue(value)}
}

func (m *mysqlDialect) PathGte(column, path string, value any) (string, []any) {
	return fmt.Sprintf("JSON_EXTRACT(%s, ?) >= ?", column), []any{path, marshalValue(value)}
}

func (m *mysqlDialect) PathLt(column, path string, value any) (string, []any) {
	return fmt.Sprintf("JSON_EXTRACT(%s, ?) < ?", column), []any{path, marshalValue(value)}
}

func (m *mysqlDialect) PathLte(column, path string, value any) (string, []any) {
	return fmt.Sprintf("JSON_EXTRACT(%s, ?) <= ?", column), []any{path, marshalValue(value)}
}

func (m *mysqlDialect) Contains(column string, value any, path string) (string, []any) {
	if path != "" {
		return fmt.Sprintf("JSON_CONTAINS(%s, ?, ?)", column), []any{marshalValue(value), path}
	}
	return fmt.Sprintf("JSON_CONTAINS(%s, ?)", column), []any{marshalValue(value)}
}

func (m *mysqlDialect) SetPath(column, path string, value any) clause.Expr {
	return clause.Expr{
		SQL:  fmt.Sprintf("JSON_SET(%s, ?, ?)", column),
		Vars: []any{path, marshalValue(value)},
	}
}

func (m *mysqlDialect) SetMultiplePaths(column string, paths []string, values []any) clause.Expr {
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
		SQL:  fmt.Sprintf("JSON_SET(%s, %s)", column, strings.Join(placeholders, ", ")),
		Vars: vars,
	}
}

func (m *mysqlDialect) RemovePath(column, path string) clause.Expr {
	return clause.Expr{
		SQL:  fmt.Sprintf("JSON_REMOVE(%s, ?)", column),
		Vars: []any{path},
	}
}

func (m *mysqlDialect) RemoveMultiplePaths(column string, paths []string) clause.Expr {
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
		SQL:  fmt.Sprintf("JSON_REMOVE(%s, %s)", column, strings.Join(placeholders, ", ")),
		Vars: vars,
	}
}

func (m *mysqlDialect) MergePatch(column string, value any) clause.Expr {
	return clause.Expr{
		SQL:  fmt.Sprintf("JSON_MERGE_PATCH(%s, ?)", column),
		Vars: []any{marshalValue(value)},
	}
}

func (m *mysqlDialect) MergePreserve(column string, value any) clause.Expr {
	return clause.Expr{
		SQL:  fmt.Sprintf("JSON_MERGE_PRESERVE(%s, ?)", column),
		Vars: []any{marshalValue(value)},
	}
}

// marshalValue converts a Go value to JSON string for SQL parameters
func marshalValue(v any) string {
	bytes, _ := json.Marshal(v)
	return string(bytes)
}
