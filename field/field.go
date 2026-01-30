package field

import "github.com/arllen133/sqlc/clause"

// Field represents a generic field for any type.
// Use this for types that don't have a specific field type.
type Field struct {
	column clause.Column
}

// Column returns the underlying column for this field
func (f Field) Column() clause.Column { return f.column }

// ColumnName implements the clause.Columnar interface
func (f Field) ColumnName() string {
	return f.column.ColumnName()
}

var _ clause.Columnar = Field{}

// WithColumn creates a new Field with the specified column name.
func (f Field) WithColumn(name string) Field {
	column := f.column
	column.Name = name
	return Field{column: column}
}

// WithTable creates a new Field with the specified table name.
func (f Field) WithTable(name string) Field {
	column := f.column
	column.Table = name
	return Field{column: column}
}

// Query functions

// Eq creates an equality comparison expression (field = value).
func (f Field) Eq(value any) clause.Expression {
	return clause.Eq{Column: f.column, Value: value}
}

// Neq creates a not equal comparison expression (field != value).
func (f Field) Neq(value any) clause.Expression {
	return clause.Neq{Column: f.column, Value: value}
}

// In creates an IN comparison expression (field IN (values...)).
func (f Field) In(values ...any) clause.Expression {
	return clause.IN{Column: f.column, Values: values}
}

// NotIn creates a NOT IN comparison expression (field NOT IN (values...)).
func (f Field) NotIn(values ...any) clause.Expression {
	return clause.Not{Expr: clause.IN{Column: f.column, Values: values}}
}

// IsNull creates a NULL check expression (field IS NULL).
func (f Field) IsNull() clause.Expression {
	return clause.Expr{SQL: "? IS NULL", Vars: []any{f.column}}
}

// IsNotNull creates a NOT NULL check expression (field IS NOT NULL).
func (f Field) IsNotNull() clause.Expression {
	return clause.Expr{SQL: "? IS NOT NULL", Vars: []any{f.column}}
}

// Set functions for UPDATE operations

// Set creates an assignment expression for UPDATE operations (field = value).
func (f Field) Set(val any) clause.Assignment {
	return clause.Assignment{Column: f.column, Value: val}
}

// Order expressions for sorting operations

// Asc creates an ascending order expression for ORDER BY clauses.
func (f Field) Asc() clause.OrderByColumn {
	return clause.OrderByColumn{Column: f.column, Desc: false}
}

// Desc creates a descending order expression for ORDER BY clauses.
func (f Field) Desc() clause.OrderByColumn {
	return clause.OrderByColumn{Column: f.column, Desc: true}
}
