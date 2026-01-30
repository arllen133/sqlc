package json

import (
	"github.com/arllen133/sqlc/clause"
)

// JSONPath represents a path within a JSON column.
// It holds both the column name and the JSON path expression.
type JSONPath struct {
	Column string // Database column name
	Path   string // JSON path expression (e.g. "$.name")
}

// With returns a JSONPathOps that can be used for query operations
// with the specified dialect.
func (p JSONPath) With(dialect JSONDialect) JSONPathOps {
	return NewPathOps(clause.Column{Name: p.Column}, p.Path, dialect)
}

// ops returns a JSONPathOps using the default dialect.
func (p JSONPath) ops() JSONPathOps {
	return p.With(DefaultDialect())
}

// Eq creates an equality expression using the default dialect.
func (p JSONPath) Eq(value any) clause.Expression {
	return p.ops().Eq(value)
}

// Neq creates a not-equal expression using the default dialect.
func (p JSONPath) Neq(value any) clause.Expression {
	return p.ops().Neq(value)
}

// Contains creates a contains expression using the default dialect.
func (p JSONPath) Contains(value any) clause.Expression {
	return p.ops().Contains(value)
}

// Gt creates a greater-than expression using the default dialect.
func (p JSONPath) Gt(value any) clause.Expression {
	return p.ops().Gt(value)
}

// Lt creates a less-than expression using the default dialect.
func (p JSONPath) Lt(value any) clause.Expression {
	return p.ops().Lt(value)
}

// Gte creates a greater-than-or-equal expression using the default dialect.
func (p JSONPath) Gte(value any) clause.Expression {
	return p.ops().Gte(value)
}

// Lte creates a less-than-or-equal expression using the default dialect.
func (p JSONPath) Lte(value any) clause.Expression {
	return p.ops().Lte(value)
}

// Set creates an assignment expression for setting this JSON path using the default dialect.
func (p JSONPath) Set(value any) clause.Assignment {
	return p.ops().Set(value)
}

// Remove creates an assignment expression for removing this JSON path using the default dialect.
func (p JSONPath) Remove() clause.Assignment {
	return p.ops().Remove()
}

// PathValue represents a path-value pair for bulk updates.
type PathValue struct {
	Path  string
	Value any
}

// Arg creates a PathValue pair for use with bulk update methods.
func (p JSONPath) Arg(value any) PathValue {
	return PathValue{Path: p.Path, Value: value}
}
