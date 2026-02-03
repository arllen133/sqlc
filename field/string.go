package field

import "github.com/arllen133/sqlc/clause"

// String represents a string field for building SQL queries.
type String struct {
	column clause.Column
}

// Column returns the underlying column for this field
func (s String) Column() clause.Column { return s.column }

// ColumnName implements the clause.Columnar interface
func (s String) ColumnName() string {
	return s.column.ColumnName()
}

var _ clause.Columnar = String{}

// WithColumn creates a new String field with the specified column name.
func (s String) WithColumn(name string) String {
	column := s.column
	column.Name = name
	return String{column: column}
}

// WithTable creates a new String field with the specified table name.
func (s String) WithTable(name string) String {
	column := s.column
	column.Table = name
	return String{column: column}
}

// Query functions

// Eq creates an equality comparison expression (field = value).
func (s String) Eq(value string) clause.Expression {
	return clause.Eq{Column: s.column, Value: value}
}

// Neq creates a not equal comparison expression (field != value).
func (s String) Neq(value string) clause.Expression {
	return clause.Neq{Column: s.column, Value: value}
}

// Like creates a LIKE comparison expression (field LIKE pattern).
func (s String) Like(pattern string) clause.Expression {
	return clause.Like{Column: s.column, Value: pattern}
}

// NotLike creates a NOT LIKE comparison expression (field NOT LIKE pattern).
func (s String) NotLike(pattern string) clause.Expression {
	return clause.NotLike{Column: s.column, Value: pattern}
}

// In creates an IN comparison expression (field IN (values...)).
func (s String) In(values ...string) clause.Expression {
	interfaceValues := make([]any, len(values))
	for i, v := range values {
		interfaceValues[i] = v
	}
	return clause.IN{Column: s.column, Values: interfaceValues}
}

// NotIn creates a NOT IN comparison expression (field NOT IN (values...)).
func (s String) NotIn(values ...string) clause.Expression {
	interfaceValues := make([]any, len(values))
	for i, v := range values {
		interfaceValues[i] = v
	}
	return clause.Not{Expr: clause.IN{Column: s.column, Values: interfaceValues}}
}

// IsNull creates a NULL check expression (field IS NULL).
func (s String) IsNull() clause.Expression {
	return clause.IsNull{Column: s.column}
}

// IsNotNull creates a NOT NULL check expression (field IS NOT NULL).
func (s String) IsNotNull() clause.Expression {
	return clause.IsNotNull{Column: s.column}
}

// Set functions for UPDATE operations

// Set creates an assignment expression for UPDATE operations (field = value).
func (s String) Set(val string) clause.Assignment {
	return clause.Assignment{Column: s.column, Value: val}
}

// Order expressions for sorting operations

// Asc creates an ascending order expression for ORDER BY clauses.
func (s String) Asc() clause.OrderByColumn {
	return clause.OrderByColumn{Column: s.column, Desc: false}
}

// Desc creates a descending order expression for ORDER BY clauses.
func (s String) Desc() clause.OrderByColumn {
	return clause.OrderByColumn{Column: s.column, Desc: true}
}

// InExpr creates an IN expression with a subquery (field IN (SELECT ...)).
func (s String) InExpr(expr clause.Expression) clause.Expression {
	return clause.InExpr{Column: s.column, Expr: expr}
}

// NotInExpr creates a NOT IN expression with a subquery (field NOT IN (SELECT ...)).
func (s String) NotInExpr(expr clause.Expression) clause.Expression {
	return clause.NotInExpr{Column: s.column, Expr: expr}
}
