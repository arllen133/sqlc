package json

import (
	"fmt"
	"strings"

	"github.com/arllen133/sqlc/clause"
)

type postgresDialect struct{}

func (p *postgresDialect) Name() string { return "postgres" }

func (p *postgresDialect) ExtractPath(column, path string) (string, []any) {
	// PostgreSQL uses ->> for text extraction
	return fmt.Sprintf("%s->>'%s'", column, path), nil
}

func (p *postgresDialect) PathEq(column, path string, value any) (string, []any) {
	return fmt.Sprintf("%s #> %s = ?::jsonb", column, formatPath(path)), []any{marshalValue(value)}
}

func (p *postgresDialect) PathNeq(column, path string, value any) (string, []any) {
	return fmt.Sprintf("%s #> %s != ?::jsonb", column, formatPath(path)), []any{marshalValue(value)}
}

func (p *postgresDialect) PathGt(column, path string, value any) (string, []any) {
	return fmt.Sprintf("%s #> %s > ?::jsonb", column, formatPath(path)), []any{marshalValue(value)}
}

func (p *postgresDialect) PathGte(column, path string, value any) (string, []any) {
	return fmt.Sprintf("%s #> %s >= ?::jsonb", column, formatPath(path)), []any{marshalValue(value)}
}

func (p *postgresDialect) PathLt(column, path string, value any) (string, []any) {
	return fmt.Sprintf("%s #> %s < ?::jsonb", column, formatPath(path)), []any{marshalValue(value)}
}

func (p *postgresDialect) PathLte(column, path string, value any) (string, []any) {
	return fmt.Sprintf("%s #> %s <= ?::jsonb", column, formatPath(path)), []any{marshalValue(value)}
}

func formatPath(path string) string {
	path = strings.TrimPrefix(path, "$")
	path = strings.TrimPrefix(path, ".")
	parts := strings.Split(path, ".")
	return fmt.Sprintf("'{%s}'", strings.Join(parts, ","))
}

func (p *postgresDialect) Contains(column string, value any, path string) (string, []any) {
	if path != "" {
		return fmt.Sprintf("%s->'%s' @> ?::jsonb", column, path), []any{marshalValue(value)}
	}
	return fmt.Sprintf("%s @> ?::jsonb", column), []any{marshalValue(value)}
}

func (p *postgresDialect) SetPath(column, path string, value any) clause.Expr {
	return clause.Expr{
		SQL:  fmt.Sprintf("jsonb_set(%s, '{%s}', ?::jsonb)", column, path),
		Vars: []any{marshalValue(value)},
	}
}

func (p *postgresDialect) SetMultiplePaths(column string, paths []string, values []any) clause.Expr {
	if len(paths) == 0 {
		return clause.Expr{}
	}

	// PostgreSQL jsonb_set only supports one path per call, so we nest them
	innerSQL := column
	vars := make([]any, 0, len(paths))

	for i, path := range paths {
		innerSQL = fmt.Sprintf("jsonb_set(%s, '{%s}', ?::jsonb)", innerSQL, path)
		vars = append(vars, marshalValue(values[i]))
	}

	return clause.Expr{SQL: innerSQL, Vars: vars}
}

func (p *postgresDialect) RemovePath(column, path string) clause.Expr {
	return clause.Expr{
		SQL:  fmt.Sprintf("%s - ?", column),
		Vars: []any{path},
	}
}

func (p *postgresDialect) RemoveMultiplePaths(column string, paths []string) clause.Expr {
	if len(paths) == 0 {
		return clause.Expr{}
	}

	var sql strings.Builder
	sql.WriteString(column)
	vars := make([]any, len(paths))
	for i, path := range paths {
		sql.WriteString(" - ?")
		vars[i] = path
	}

	return clause.Expr{SQL: sql.String(), Vars: vars}
}

func (p *postgresDialect) MergePatch(column string, value any) clause.Expr {
	return clause.Expr{
		SQL:  fmt.Sprintf("%s || ?::jsonb", column),
		Vars: []any{marshalValue(value)},
	}
}

func (p *postgresDialect) MergePreserve(column string, value any) clause.Expr {
	// PostreSQL || operator concatenates arrays (Merge Preserve behavior)
	return clause.Expr{
		SQL:  fmt.Sprintf("%s || ?::jsonb", column),
		Vars: []any{marshalValue(value)},
	}
}
