package clause

import (
	"fmt"
	"strings"
)

// Columnar defines an interface for providing a column name.
type Columnar interface {
	ColumnName() string
}

// Column represents a database column with optional table qualifier
type Column struct {
	Table string
	Name  string
}

// ColumnName returns the full column name (with table prefix if specified)
func (c Column) ColumnName() string {
	if c.Table != "" {
		return c.Table + "." + c.Name
	}
	return c.Name
}

var _ Columnar = Column{}

// Expression is the base interface for all SQL expressions
type Expression interface {
	Build() (sql string, args []any)
}

// Eq represents an equality expression (column = value)
type Eq struct {
	Column Column
	Value  any
}

func (e Eq) Build() (string, []any) {
	return e.Column.ColumnName() + " = ?", []any{e.Value}
}

// Neq represents a not equal expression (column != value)
type Neq struct {
	Column Column
	Value  any
}

func (n Neq) Build() (string, []any) {
	return n.Column.ColumnName() + " <> ?", []any{n.Value}
}

// Gt represents a greater than expression (column > value)
type Gt struct {
	Column Column
	Value  any
}

func (g Gt) Build() (string, []any) {
	return g.Column.ColumnName() + " > ?", []any{g.Value}
}

// Gte represents a greater than or equal expression (column >= value)
type Gte struct {
	Column Column
	Value  any
}

func (g Gte) Build() (string, []any) {
	return g.Column.ColumnName() + " >= ?", []any{g.Value}
}

// Lt represents a less than expression (column < value)
type Lt struct {
	Column Column
	Value  any
}

func (l Lt) Build() (string, []any) {
	return l.Column.ColumnName() + " < ?", []any{l.Value}
}

// Lte represents a less than or equal expression (column <= value)
type Lte struct {
	Column Column
	Value  any
}

func (l Lte) Build() (string, []any) {
	return l.Column.ColumnName() + " <= ?", []any{l.Value}
}

// Like represents a LIKE expression
type Like struct {
	Column Column
	Value  string
}

func (l Like) Build() (string, []any) {
	return l.Column.ColumnName() + " LIKE ?", []any{l.Value}
}

// NotLike represents a NOT LIKE expression
type NotLike struct {
	Column Column
	Value  string
}

func (n NotLike) Build() (string, []any) {
	return n.Column.ColumnName() + " NOT LIKE ?", []any{n.Value}
}

// IsNull represents an IS NULL expression
type IsNull struct {
	Column Column
}

func (i IsNull) Build() (string, []any) {
	return i.Column.ColumnName() + " IS NULL", nil
}

// IsNotNull represents an IS NOT NULL expression
type IsNotNull struct {
	Column Column
}

func (i IsNotNull) Build() (string, []any) {
	return i.Column.ColumnName() + " IS NOT NULL", nil
}

// IN represents an IN expression
type IN struct {
	Column Column
	Values []any
}

func (i IN) Build() (string, []any) {
	if len(i.Values) == 0 {
		return "1 = 0", nil // IN with empty list is always false
	}

	placeholders := make([]string, len(i.Values))
	for idx := range i.Values {
		placeholders[idx] = "?"
	}

	sql := fmt.Sprintf("%s IN (%s)", i.Column.ColumnName(), strings.Join(placeholders, ", "))
	return sql, i.Values
}

// Between represents a BETWEEN expression
type Between struct {
	Column Column
	Min    any
	Max    any
}

func (b Between) Build() (string, []any) {
	sql := fmt.Sprintf("%s BETWEEN ? AND ?", b.Column.ColumnName())
	return sql, []any{b.Min, b.Max}
}

// And represents an AND expression
type And []Expression

func (a And) Build() (string, []any) {
	if len(a) == 0 {
		return "1 = 1", nil // Empty AND is always true
	}

	var sqls []string
	var args []any

	for _, expr := range a {
		sql, exprArgs := expr.Build()
		sqls = append(sqls, "("+sql+")")
		args = append(args, exprArgs...)
	}

	return strings.Join(sqls, " AND "), args
}

// Or represents an OR expression
type Or []Expression

func (o Or) Build() (string, []any) {
	if len(o) == 0 {
		return "1 = 0", nil // Empty OR is always false
	}

	var sqls []string
	var args []any

	for _, expr := range o {
		sql, exprArgs := expr.Build()
		sqls = append(sqls, "("+sql+")")
		args = append(args, exprArgs...)
	}

	return strings.Join(sqls, " OR "), args
}

// Not represents a NOT expression
type Not struct {
	Expr Expression
}

func (n Not) Build() (string, []any) {
	sql, args := n.Expr.Build()
	return "NOT (" + sql + ")", args
}

// Expr represents a custom SQL expression
type Expr struct {
	SQL  string
	Vars []any
}

func (e Expr) Build() (string, []any) {
	return e.SQL, e.Vars
}

// Assignment represents a column assignment for UPDATE
type Assignment struct {
	Column Column
	Value  any
}

func (a Assignment) Build() (string, []any) {
	return a.Column.ColumnName() + " = ?", []any{a.Value}
}

// OrderByColumn represents an ORDER BY column
type OrderByColumn struct {
	Column Column
	Desc   bool
}

func (o OrderByColumn) Build() string {
	sql := o.Column.ColumnName()
	if o.Desc {
		sql += " DESC"
	}
	return sql
}
