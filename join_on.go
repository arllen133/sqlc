package sqlc

import "github.com/arllen133/sqlc/clause"

type JoinOn struct {
	Left  clause.Column
	Right clause.Column
}

func On(left, right interface{ Column() clause.Column }) JoinOn {
	return JoinOn{
		Left:  left.Column(),
		Right: right.Column(),
	}
}

// Exists creates an EXISTS expression for subqueries.
// Usage: sqlc.Exists(subquery) where subquery is any Expression (e.g., QueryBuilder)
func Exists(expr clause.Expression) clause.Expression {
	return clause.ExistsExpr{Expr: expr}
}

// NotExists creates a NOT EXISTS expression for subqueries.
// Usage: sqlc.NotExists(subquery) where subquery is any Expression (e.g., QueryBuilder)
func NotExists(expr clause.Expression) clause.Expression {
	return clause.NotExistsExpr{Expr: expr}
}
