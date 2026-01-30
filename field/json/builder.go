package json

import "github.com/arllen133/sqlc/clause"

// JSONSetBuilder builds JSON SET expressions with multiple path-value pairs.
type JSONSetBuilder struct {
	column  string
	paths   []string
	values  []any
	dialect JSONDialect
}

// NewSetBuilder creates a new JSONSetBuilder for the given column and dialect.
func NewSetBuilder(column string, dialect JSONDialect) *JSONSetBuilder {
	return &JSONSetBuilder{
		column:  column,
		paths:   make([]string, 0),
		values:  make([]any, 0),
		dialect: dialect,
	}
}

// Path adds a path-value pair to the builder.
func (b *JSONSetBuilder) Path(path string, value any) *JSONSetBuilder {
	b.paths = append(b.paths, path)
	b.values = append(b.values, value)
	return b
}

// Build generates the SQL expression using the configured dialect.
func (b *JSONSetBuilder) Build() clause.Expr {
	if len(b.paths) == 0 {
		return clause.Expr{}
	}
	return b.dialect.SetMultiplePaths(b.column, b.paths, b.values)
}

// Assignment returns the clause.Assignment for use with UpdateColumns.
func (b *JSONSetBuilder) Assignment(col clause.Column) clause.Assignment {
	return clause.Assignment{
		Column: col,
		Value:  b.Build(),
	}
}

// JSONRemoveBuilder builds JSON REMOVE expressions with multiple paths.
type JSONRemoveBuilder struct {
	column  string
	paths   []string
	dialect JSONDialect
}

// NewRemoveBuilder creates a new JSONRemoveBuilder for the given column and dialect.
func NewRemoveBuilder(column string, dialect JSONDialect) *JSONRemoveBuilder {
	return &JSONRemoveBuilder{
		column:  column,
		paths:   make([]string, 0),
		dialect: dialect,
	}
}

// Path adds a path to remove.
func (b *JSONRemoveBuilder) Path(path string) *JSONRemoveBuilder {
	b.paths = append(b.paths, path)
	return b
}

// Build generates the SQL expression using the configured dialect.
func (b *JSONRemoveBuilder) Build() clause.Expr {
	if len(b.paths) == 0 {
		return clause.Expr{}
	}
	return b.dialect.RemoveMultiplePaths(b.column, b.paths)
}

// Assignment returns the clause.Assignment for use with UpdateColumns.
func (b *JSONRemoveBuilder) Assignment(col clause.Column) clause.Assignment {
	return clause.Assignment{
		Column: col,
		Value:  b.Build(),
	}
}

// JSONPathOps provides JSON path operations with a specific dialect.
type JSONPathOps struct {
	column  clause.Column
	path    string
	dialect JSONDialect
}

// NewPathOps creates a JSONPathOps for the given column, path, and dialect.
func NewPathOps(column clause.Column, path string, dialect JSONDialect) JSONPathOps {
	return JSONPathOps{column: column, path: path, dialect: dialect}
}

// Eq creates an equality expression for this JSON path.
func (p JSONPathOps) Eq(value any) clause.Expression {
	sql, vars := p.dialect.PathEq(p.column.ColumnName(), p.path, value)
	return clause.Expr{SQL: sql, Vars: vars}
}

// Neq creates a not-equal expression for this JSON path.
func (p JSONPathOps) Neq(value any) clause.Expression {
	sql, vars := p.dialect.PathNeq(p.column.ColumnName(), p.path, value)
	return clause.Expr{SQL: sql, Vars: vars}
}

// Gt creates a greater-than expression for this JSON path.
func (p JSONPathOps) Gt(value any) clause.Expression {
	sql, vars := p.dialect.PathGt(p.column.ColumnName(), p.path, value)
	return clause.Expr{SQL: sql, Vars: vars}
}

// Gte creates a greater-than-or-equal expression for this JSON path.
func (p JSONPathOps) Gte(value any) clause.Expression {
	sql, vars := p.dialect.PathGte(p.column.ColumnName(), p.path, value)
	return clause.Expr{SQL: sql, Vars: vars}
}

// Lt creates a less-than expression for this JSON path.
func (p JSONPathOps) Lt(value any) clause.Expression {
	sql, vars := p.dialect.PathLt(p.column.ColumnName(), p.path, value)
	return clause.Expr{SQL: sql, Vars: vars}
}

// Lte creates a less-than-or-equal expression for this JSON path.
func (p JSONPathOps) Lte(value any) clause.Expression {
	sql, vars := p.dialect.PathLte(p.column.ColumnName(), p.path, value)
	return clause.Expr{SQL: sql, Vars: vars}
}

// Contains creates a contains expression for this JSON path.
func (p JSONPathOps) Contains(value any) clause.Expression {
	sql, vars := p.dialect.Contains(p.column.ColumnName(), value, p.path)
	return clause.Expr{SQL: sql, Vars: vars}
}

// Set creates an assignment expression for setting this JSON path.
func (p JSONPathOps) Set(value any) clause.Assignment {
	return clause.Assignment{
		Column: p.column,
		Value:  p.dialect.SetPath(p.column.ColumnName(), p.path, value),
	}
}

// Remove creates an assignment expression for removing this JSON path.
func (p JSONPathOps) Remove() clause.Assignment {
	return clause.Assignment{
		Column: p.column,
		Value:  p.dialect.RemovePath(p.column.ColumnName(), p.path),
	}
}
