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

func (c Column) Column() Column { return c }

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
	Build() (sql string, args []any, err error)
}

// Eq represents an equality expression (column = value)
type Eq struct {
	Column Column
	Value  any
}

func (e Eq) Build() (string, []any, error) {
	return e.Column.ColumnName() + " = ?", []any{e.Value}, nil
}

// Neq represents a not equal expression (column != value)
type Neq struct {
	Column Column
	Value  any
}

func (n Neq) Build() (string, []any, error) {
	return n.Column.ColumnName() + " <> ?", []any{n.Value}, nil
}

// Gt represents a greater than expression (column > value)
type Gt struct {
	Column Column
	Value  any
}

func (g Gt) Build() (string, []any, error) {
	return g.Column.ColumnName() + " > ?", []any{g.Value}, nil
}

// Gte represents a greater than or equal expression (column >= value)
type Gte struct {
	Column Column
	Value  any
}

func (g Gte) Build() (string, []any, error) {
	return g.Column.ColumnName() + " >= ?", []any{g.Value}, nil
}

// Lt represents a less than expression (column < value)
type Lt struct {
	Column Column
	Value  any
}

func (l Lt) Build() (string, []any, error) {
	return l.Column.ColumnName() + " < ?", []any{l.Value}, nil
}

// Lte represents a less than or equal expression (column <= value)
type Lte struct {
	Column Column
	Value  any
}

func (l Lte) Build() (string, []any, error) {
	return l.Column.ColumnName() + " <= ?", []any{l.Value}, nil
}

// Like represents a LIKE expression
type Like struct {
	Column Column
	Value  string
}

func (l Like) Build() (string, []any, error) {
	return l.Column.ColumnName() + " LIKE ?", []any{l.Value}, nil
}

// NotLike represents a NOT LIKE expression
type NotLike struct {
	Column Column
	Value  string
}

func (n NotLike) Build() (string, []any, error) {
	return n.Column.ColumnName() + " NOT LIKE ?", []any{n.Value}, nil
}

// IsNull represents an IS NULL expression
type IsNull struct {
	Column Column
}

func (i IsNull) Build() (string, []any, error) {
	return i.Column.ColumnName() + " IS NULL", nil, nil
}

// IsNotNull represents an IS NOT NULL expression
type IsNotNull struct {
	Column Column
}

func (i IsNotNull) Build() (string, []any, error) {
	return i.Column.ColumnName() + " IS NOT NULL", nil, nil
}

// IN represents an IN expression
type IN struct {
	Column Column
	Values []any
}

func (i IN) Build() (string, []any, error) {
	switch len(i.Values) {
	case 0:
		return "1 = 0", nil, nil // IN with empty list is always false
	case 1:
		return i.Column.ColumnName() + " = ?", []any{i.Values[0]}, nil
	default:
		placeholders := make([]string, len(i.Values))
		for idx := range i.Values {
			placeholders[idx] = "?"
		}

		sql := fmt.Sprintf("%s IN (%s)", i.Column.ColumnName(), strings.Join(placeholders, ", "))
		return sql, i.Values, nil
	}
}

// Between represents a BETWEEN expression
type Between struct {
	Column Column
	Min    any
	Max    any
}

func (b Between) Build() (string, []any, error) {
	sql := fmt.Sprintf("%s BETWEEN ? AND ?", b.Column.ColumnName())
	return sql, []any{b.Min, b.Max}, nil
}

// And represents an AND expression
type And []Expression

func (a And) Build() (string, []any, error) {
	if len(a) == 0 {
		return "1 = 1", nil, nil // Empty AND is always true
	}

	var sqls []string
	var args []any

	for _, expr := range a {
		sql, exprArgs, err := expr.Build()
		if err != nil {
			return "", nil, err
		}
		sqls = append(sqls, "("+sql+")")
		args = append(args, exprArgs...)
	}

	return strings.Join(sqls, " AND "), args, nil
}

// Or represents an OR expression
type Or []Expression

func (o Or) Build() (string, []any, error) {
	if len(o) == 0 {
		return "1 = 0", nil, nil // Empty OR is always false
	}

	var sqls []string
	var args []any

	for _, expr := range o {
		sql, exprArgs, err := expr.Build()
		if err != nil {
			return "", nil, err
		}
		sqls = append(sqls, "("+sql+")")
		args = append(args, exprArgs...)
	}

	return strings.Join(sqls, " OR "), args, nil
}

// Not represents a NOT expression
type Not struct {
	Expr Expression
}

func (n Not) Build() (string, []any, error) {
	sql, args, err := n.Expr.Build()
	if err != nil {
		return "", nil, err
	}
	return "NOT (" + sql + ")", args, nil
}

// Expr represents a custom SQL expression
type Expr struct {
	SQL  string
	Vars []any
}

func (e Expr) Build() (string, []any, error) {
	return e.SQL, e.Vars, nil
}

// Assignment represents a column assignment for UPDATE
type Assignment struct {
	Column Column
	Value  any
}

func (a Assignment) Build() (string, []any, error) {
	return a.Column.ColumnName() + " = ?", []any{a.Value}, nil
}

// OrderByColumn represents an ORDER BY column
type OrderByColumn struct {
	Column Column
	Desc   bool
}

func (o OrderByColumn) Build() (string, []any, error) {
	sql := o.Column.ColumnName()
	if o.Desc {
		sql += " DESC"
	}
	return sql, nil, nil
}

// InExpr represents column IN (expression) - typically used for subqueries
type InExpr struct {
	Column Column
	Expr   Expression
}

func (i InExpr) Build() (string, []any, error) {
	sql, args, err := i.Expr.Build()
	if err != nil {
		return "", nil, err
	}
	return fmt.Sprintf("%s IN (%s)", i.Column.ColumnName(), sql), args, nil
}

// NotInExpr represents column NOT IN (expression) - typically used for subqueries
type NotInExpr struct {
	Column Column
	Expr   Expression
}

func (n NotInExpr) Build() (string, []any, error) {
	sql, args, err := n.Expr.Build()
	if err != nil {
		return "", nil, err
	}
	return fmt.Sprintf("%s NOT IN (%s)", n.Column.ColumnName(), sql), args, nil
}

// ExistsExpr represents EXISTS (expression)
type ExistsExpr struct {
	Expr Expression
}

func (e ExistsExpr) Build() (string, []any, error) {
	sql, args, err := e.Expr.Build()
	if err != nil {
		return "", nil, err
	}
	return "EXISTS (" + sql + ")", args, nil
}

// NotExistsExpr represents NOT EXISTS (expression)
type NotExistsExpr struct {
	Expr Expression
}

func (n NotExistsExpr) Build() (string, []any, error) {
	sql, args, err := n.Expr.Build()
	if err != nil {
		return "", nil, err
	}
	return "NOT EXISTS (" + sql + ")", args, nil
}
