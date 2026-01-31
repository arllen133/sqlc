package field

import "github.com/arllen133/sqlc/clause"

// Field represents a generic field for any type.
// Use this for types that don't have a specific field type.
// Field represents a generic field for any type.
// Use this for types that don't have a specific field type.
type Field[T any] struct {
	column clause.Column
}

// Column returns the underlying column for this field
func (f Field[T]) Column() clause.Column { return f.column }

// ColumnName implements the clause.Columnar interface
func (f Field[T]) ColumnName() string {
	return f.column.ColumnName()
}

var _ clause.Columnar = Field[any]{}

// WithColumn creates a new Field with the specified column name.
func (f Field[T]) WithColumn(name string) Field[T] {
	column := f.column
	column.Name = name
	return Field[T]{column: column}
}

// WithTable creates a new Field with the specified table name.
func (f Field[T]) WithTable(name string) Field[T] {
	column := f.column
	column.Table = name
	return Field[T]{column: column}
}

// Query functions

// Eq creates an equality comparison expression (field = value).
func (f Field[T]) Eq(value T) clause.Expression {
	return clause.Eq{Column: f.column, Value: value}
}

// Neq creates a not equal comparison expression (field != value).
func (f Field[T]) Neq(value T) clause.Expression {
	return clause.Neq{Column: f.column, Value: value}
}

// In creates an IN comparison expression (field IN (values...)).
func (f Field[T]) In(values ...T) clause.Expression {
	interfaceValues := make([]any, len(values))
	for i, v := range values {
		interfaceValues[i] = v
	}
	return clause.IN{Column: f.column, Values: interfaceValues}
}

// NotIn creates a NOT IN comparison expression (field NOT IN (values...)).
func (f Field[T]) NotIn(values ...T) clause.Expression {
	interfaceValues := make([]any, len(values))
	for i, v := range values {
		interfaceValues[i] = v
	}
	return clause.Not{Expr: clause.IN{Column: f.column, Values: interfaceValues}}
}

// IsNull creates a NULL check expression (field IS NULL).
func (f Field[T]) IsNull() clause.Expression {
	return clause.IsNull{Column: f.column}
}

// IsNotNull creates a NOT NULL check expression (field IS NOT NULL).
func (f Field[T]) IsNotNull() clause.Expression {
	return clause.IsNotNull{Column: f.column}
}

// Set functions for UPDATE operations

// Set creates an assignment expression for UPDATE operations (field = value).
func (f Field[T]) Set(val T) clause.Assignment {
	return clause.Assignment{Column: f.column, Value: val}
}

// Order expressions for sorting operations

// Asc creates an ascending order expression for ORDER BY clauses.
func (f Field[T]) Asc() clause.OrderByColumn {
	return clause.OrderByColumn{Column: f.column, Desc: false}
}

// Desc creates a descending order expression for ORDER BY clauses.
func (f Field[T]) Desc() clause.OrderByColumn {
	return clause.OrderByColumn{Column: f.column, Desc: true}
}
