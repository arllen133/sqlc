package field

import (
	"encoding/json"

	"github.com/arllen133/sqlc/clause"
)

// JSON represents a JSON field for building SQL queries.
// It supports any type that implements json.Marshaler/Unmarshaler indirectly.
type JSON[T any] struct {
	column clause.Column
}

// Column returns the underlying column for this field
func (j JSON[T]) Column() clause.Column { return j.column }

// ColumnName implements the clause.Columnar interface
func (j JSON[T]) ColumnName() string {
	return j.column.ColumnName()
}

var _ clause.Columnar = JSON[any]{}

// WithColumn creates a new JSON field with the specified column name.
func (j JSON[T]) WithColumn(name string) JSON[T] {
	column := j.column
	column.Name = name
	return JSON[T]{column: column}
}

// WithTable creates a new JSON field with the specified table name.
func (j JSON[T]) WithTable(name string) JSON[T] {
	column := j.column
	column.Table = name
	return JSON[T]{column: column}
}

// Query functions

// Eq creates an equality comparison expression.
// Note: Direct JSON equality might depend on database dialect support.
func (j JSON[T]) Eq(value T) clause.Expression {
	bytes, _ := json.Marshal(value)
	return clause.Eq{Column: j.column, Value: string(bytes)}
}

// Neq creates a not equal comparison expression.
func (j JSON[T]) Neq(value T) clause.Expression {
	bytes, _ := json.Marshal(value)
	return clause.Neq{Column: j.column, Value: string(bytes)}
}

// IsNull creates a NULL check expression (field IS NULL).
func (j JSON[T]) IsNull() clause.Expression {
	return clause.Expr{SQL: j.column.ColumnName() + " IS NULL", Vars: nil}
}

// IsNotNull creates a NOT NULL check expression (field IS NOT NULL).
func (j JSON[T]) IsNotNull() clause.Expression {
	return clause.Expr{SQL: j.column.ColumnName() + " IS NOT NULL", Vars: nil}
}

// Set functions for UPDATE operations

// Set creates an assignment expression with JSON marshaling.
func (j JSON[T]) Set(val T) clause.Assignment {
	bytes, _ := json.Marshal(val) // In a real app handle error? But Set returns helper object.
	// For most DB drivers (postgres driver), passing string or []byte works for JSON columns.
	return clause.Assignment{Column: j.column, Value: string(bytes)}
}

// RawSet allows setting raw JSON string/bytes directly if needed
func (j JSON[T]) RawSet(val any) clause.Assignment {
	return clause.Assignment{Column: j.column, Value: val}
}
