package field

import (
	"github.com/arllen133/sqlc/clause"
)

// Bytes represents a binary data field (BLOB/BYTEA) for building SQL queries.
type Bytes struct {
	column clause.Column
}

// Column returns the underlying column for this field
func (b Bytes) Column() clause.Column { return b.column }

// ColumnName implements the clause.Columnar interface
func (b Bytes) ColumnName() string {
	return b.column.ColumnName()
}

var _ clause.Columnar = Bytes{}

// WithColumn creates a new Bytes field with the specified column name.
func (b Bytes) WithColumn(name string) Bytes {
	column := b.column
	column.Name = name
	return Bytes{column: column}
}

// WithTable creates a new Bytes field with the specified table name.
func (b Bytes) WithTable(name string) Bytes {
	column := b.column
	column.Table = name
	return Bytes{column: column}
}

// Query functions

// Eq creates an equality comparison expression (field = value).
func (b Bytes) Eq(value []byte) clause.Expression {
	return clause.Eq{Column: b.column, Value: value}
}

// Neq creates a not equal comparison expression (field != value).
func (b Bytes) Neq(value []byte) clause.Expression {
	return clause.Neq{Column: b.column, Value: value}
}

// IsNull creates a NULL check expression (field IS NULL).
func (b Bytes) IsNull() clause.Expression {
	return clause.IsNull{Column: b.column}
}

// IsNotNull creates a NOT NULL check expression (field IS NOT NULL).
func (b Bytes) IsNotNull() clause.Expression {
	return clause.IsNotNull{Column: b.column}
}

// Set functions for UPDATE operations

// Set creates an assignment expression for UPDATE operations (field = value).
func (b Bytes) Set(val []byte) clause.Assignment {
	return clause.Assignment{Column: b.column, Value: val}
}
