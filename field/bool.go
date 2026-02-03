package field

import "github.com/arllen133/sqlc/clause"

// Bool represents a boolean field for building SQL queries.
type Bool struct {
	column clause.Column
}

// Column returns the underlying column for this field
func (b Bool) Column() clause.Column { return b.column }

// ColumnName implements the clause.Columnar interface
func (b Bool) ColumnName() string {
	return b.column.ColumnName()
}

var _ clause.Columnar = Bool{}

// WithColumn creates a new Bool field with the specified column name.
func (b Bool) WithColumn(name string) Bool {
	column := b.column
	column.Name = name
	return Bool{column: column}
}

// WithTable creates a new Bool field with the specified table name.
func (b Bool) WithTable(name string) Bool {
	column := b.column
	column.Table = name
	return Bool{column: column}
}

// Query functions

// Eq creates an equality comparison expression (field = value).
func (b Bool) Eq(value bool) clause.Expression {
	return clause.Eq{Column: b.column, Value: value}
}

// Neq creates a not equal comparison expression (field != value).
func (b Bool) Neq(value bool) clause.Expression {
	return clause.Neq{Column: b.column, Value: value}
}

// IsTrue creates a TRUE check expression (field = TRUE).
func (b Bool) IsTrue() clause.Expression {
	return clause.Eq{Column: b.column, Value: true}
}

// IsFalse creates a FALSE check expression (field = FALSE).
func (b Bool) IsFalse() clause.Expression {
	return clause.Eq{Column: b.column, Value: false}
}

// IsNull creates a NULL check expression (field IS NULL).
func (b Bool) IsNull() clause.Expression {
	return clause.IsNull{Column: b.column}
}

// IsNotNull creates a NOT NULL check expression (field IS NOT NULL).
func (b Bool) IsNotNull() clause.Expression {
	return clause.IsNotNull{Column: b.column}
}

// Set functions for UPDATE operations

// Set creates an assignment expression for UPDATE operations (field = value).
func (b Bool) Set(val bool) clause.Assignment {
	return clause.Assignment{Column: b.column, Value: val}
}

// Order expressions for sorting operations

// Asc creates an ascending order expression for ORDER BY clauses.
func (b Bool) Asc() clause.OrderByColumn {
	return clause.OrderByColumn{Column: b.column, Desc: false}
}

// Desc creates a descending order expression for ORDER BY clauses.
func (b Bool) Desc() clause.OrderByColumn {
	return clause.OrderByColumn{Column: b.column, Desc: true}
}

// InExpr creates an IN expression with a subquery (field IN (SELECT ...)).
func (b Bool) InExpr(expr clause.Expression) clause.Expression {
	return clause.InExpr{Column: b.column, Expr: expr}
}

// NotInExpr creates a NOT IN expression with a subquery (field NOT IN (SELECT ...)).
func (b Bool) NotInExpr(expr clause.Expression) clause.Expression {
	return clause.NotInExpr{Column: b.column, Expr: expr}
}
