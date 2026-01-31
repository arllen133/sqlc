package field

import (
	"time"

	"github.com/arllen133/sqlc/clause"
)

// Time represents a time/date field for building SQL queries.
type Time struct {
	column clause.Column
}

// Column returns the underlying column for this field
func (t Time) Column() clause.Column { return t.column }

// ColumnName implements the clause.Columnar interface
func (t Time) ColumnName() string {
	return t.column.ColumnName()
}

var _ clause.Columnar = Time{}

// WithColumn creates a new Time field with the specified column name.
func (t Time) WithColumn(name string) Time {
	column := t.column
	column.Name = name
	return Time{column: column}
}

// WithTable creates a new Time field with the specified table name.
func (t Time) WithTable(name string) Time {
	column := t.column
	column.Table = name
	return Time{column: column}
}

// Query functions

// Eq creates an equality comparison expression (field = value).
func (t Time) Eq(value time.Time) clause.Expression {
	return clause.Eq{Column: t.column, Value: value}
}

// Neq creates a not equal comparison expression (field != value).
func (t Time) Neq(value time.Time) clause.Expression {
	return clause.Neq{Column: t.column, Value: value}
}

// Gt creates a greater than comparison expression (field > value).
func (t Time) Gt(value time.Time) clause.Expression {
	return clause.Gt{Column: t.column, Value: value}
}

// Gte creates a greater than or equal comparison expression (field >= value).
func (t Time) Gte(value time.Time) clause.Expression {
	return clause.Gte{Column: t.column, Value: value}
}

// Lt creates a less than comparison expression (field < value).
func (t Time) Lt(value time.Time) clause.Expression {
	return clause.Lt{Column: t.column, Value: value}
}

// Lte creates a less than or equal comparison expression (field <= value).
func (t Time) Lte(value time.Time) clause.Expression {
	return clause.Lte{Column: t.column, Value: value}
}

// Between creates a range comparison expression (field BETWEEN v1 AND v2).
func (t Time) Between(v1, v2 time.Time) clause.Expression {
	return clause.Between{Column: t.column, Min: v1, Max: v2}
}

// IsNull creates a NULL check expression (field IS NULL).
func (t Time) IsNull() clause.Expression {
	return clause.IsNull{Column: t.column}
}

// IsNotNull creates a NOT NULL check expression (field IS NOT NULL).
func (t Time) IsNotNull() clause.Expression {
	return clause.IsNotNull{Column: t.column}
}

// Set functions for UPDATE operations

// Set creates an assignment expression for UPDATE operations (field = value).
func (t Time) Set(val time.Time) clause.Assignment {
	return clause.Assignment{Column: t.column, Value: val}
}

// Order expressions for sorting operations

// Asc creates an ascending order expression for ORDER BY clauses.
func (t Time) Asc() clause.OrderByColumn {
	return clause.OrderByColumn{Column: t.column, Desc: false}
}

// Desc creates a descending order expression for ORDER BY clauses.
func (t Time) Desc() clause.OrderByColumn {
	return clause.OrderByColumn{Column: t.column, Desc: true}
}
