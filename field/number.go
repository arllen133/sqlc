package field

import (
	"github.com/arllen133/sqlc/clause"
	"golang.org/x/exp/constraints"
)

// Number represents a numeric field that supports both integer and float types.
// It provides type-safe operations for building SQL queries.
type Number[T constraints.Integer | constraints.Float] struct {
	column clause.Column
}

// Column returns the underlying column for this field
func (n Number[T]) Column() clause.Column { return n.column }

// ColumnName implements the clause.Columnar interface
func (n Number[T]) ColumnName() string {
	return n.column.ColumnName()
}

var _ clause.Columnar = Number[int]{}

// WithColumn creates a new Number field with the specified column name.
func (n Number[T]) WithColumn(name string) Number[T] {
	column := n.column
	column.Name = name
	return Number[T]{column: column}
}

// WithTable creates a new Number field with the specified table name.
func (n Number[T]) WithTable(name string) Number[T] {
	column := n.column
	column.Table = name
	return Number[T]{column: column}
}

// Query functions

// Eq creates an equality comparison expression (field = value).
func (n Number[T]) Eq(value T) clause.Expression {
	return clause.Eq{Column: n.column, Value: value}
}

// Neq creates a not equal comparison expression (field != value).
func (n Number[T]) Neq(value T) clause.Expression {
	return clause.Neq{Column: n.column, Value: value}
}

// Gt creates a greater than comparison expression (field > value).
func (n Number[T]) Gt(value T) clause.Expression {
	return clause.Gt{Column: n.column, Value: value}
}

// Gte creates a greater than or equal comparison expression (field >= value).
func (n Number[T]) Gte(value T) clause.Expression {
	return clause.Gte{Column: n.column, Value: value}
}

// Lt creates a less than comparison expression (field < value).
func (n Number[T]) Lt(value T) clause.Expression {
	return clause.Lt{Column: n.column, Value: value}
}

// Lte creates a less than or equal comparison expression (field <= value).
func (n Number[T]) Lte(value T) clause.Expression {
	return clause.Lte{Column: n.column, Value: value}
}

// Between creates a range comparison expression (field BETWEEN v1 AND v2).
func (n Number[T]) Between(v1, v2 T) clause.Expression {
	return clause.Between{Column: n.column, Min: v1, Max: v2}
}

// In creates an IN comparison expression (field IN (values...)).
func (n Number[T]) In(values ...T) clause.Expression {
	interfaceValues := make([]any, len(values))
	for i, v := range values {
		interfaceValues[i] = v
	}
	return clause.IN{Column: n.column, Values: interfaceValues}
}

// NotIn creates a NOT IN comparison expression (field NOT IN (values...)).
func (n Number[T]) NotIn(values ...T) clause.Expression {
	interfaceValues := make([]any, len(values))
	for i, v := range values {
		interfaceValues[i] = v
	}
	return clause.Not{Expr: clause.IN{Column: n.column, Values: interfaceValues}}
}

// IsNull creates a NULL check expression (field IS NULL).
func (n Number[T]) IsNull() clause.Expression {
	return clause.IsNull{Column: n.column}
}

// IsNotNull creates a NOT NULL check expression (field IS NOT NULL).
func (n Number[T]) IsNotNull() clause.Expression {
	return clause.IsNotNull{Column: n.column}
}

// Set functions for UPDATE operations

// Set creates an assignment expression for UPDATE operations (field = value).
func (n Number[T]) Set(val T) clause.Assignment {
	return clause.Assignment{Column: n.column, Value: val}
}

// Order expressions for sorting operations

// Asc creates an ascending order expression for ORDER BY clauses.
func (n Number[T]) Asc() clause.OrderByColumn {
	return clause.OrderByColumn{Column: n.column, Desc: false}
}

// Desc creates a descending order expression for ORDER BY clauses.
func (n Number[T]) Desc() clause.OrderByColumn {
	return clause.OrderByColumn{Column: n.column, Desc: true}
}

// InExpr creates an IN expression with a subquery (field IN (SELECT ...)).
func (n Number[T]) InExpr(expr clause.Expression) clause.Expression {
	return clause.InExpr{Column: n.column, Expr: expr}
}

// NotInExpr creates a NOT IN expression with a subquery (field NOT IN (SELECT ...)).
func (n Number[T]) NotInExpr(expr clause.Expression) clause.Expression {
	return clause.NotInExpr{Column: n.column, Expr: expr}
}
