package sqlc

import (
	"context"
	"fmt"
	"strings"

	"github.com/arllen133/sqlc/clause"
)

// Aggregate functions

// Sum calculates the sum of a column
func (q *QueryBuilder[T]) Sum(ctx context.Context, column clause.Columnar) (float64, error) {
	return q.aggregateFloat(ctx, "SUM", column.ColumnName())
}

// Avg calculates the average of a column
func (q *QueryBuilder[T]) Avg(ctx context.Context, column clause.Columnar) (float64, error) {
	return q.aggregateFloat(ctx, "AVG", column.ColumnName())
}

// Min calculates the minimum value of a column
func (q *QueryBuilder[T]) Min(ctx context.Context, column clause.Columnar) (any, error) {
	return q.aggregateAny(ctx, "MIN", column.ColumnName())
}

// Max calculates the maximum value of a column
func (q *QueryBuilder[T]) Max(ctx context.Context, column clause.Columnar) (any, error) {
	return q.aggregateAny(ctx, "MAX", column.ColumnName())
}

func (q *QueryBuilder[T]) aggregateFloat(ctx context.Context, funcName, column string) (float64, error) {
	val, err := q.aggregateAny(ctx, funcName, column)
	if err != nil {
		return 0, err
	}
	if val == nil {
		return 0, nil
	}

	switch v := val.(type) {
	case float64:
		return v, nil
	case int64:
		return float64(v), nil
	case []byte:
		// Handle SQLite weirdness where numbers might be returned as bytes?
		// Or if database returns string.
		return 0, fmt.Errorf("unexpected type %T for aggregate", val)
	default:
		return 0, fmt.Errorf("unexpected type %T for aggregate", val)
	}
}

func (q *QueryBuilder[T]) aggregateAny(ctx context.Context, funcName, column string) (any, error) {
	// Add dummy column to ensure valid SQL generation
	b := q.builder.Columns("1")
	sqlStr, args, err := b.ToSql()
	if err != nil {
		return nil, err
	}

	aggSql := fmt.Sprintf("SELECT %s(%s) FROM %s", funcName, column, q.schema.TableName())

	upperSql := strings.ToUpper(sqlStr)
	fromIdx := strings.Index(upperSql, " FROM ")
	if fromIdx != -1 {
		aggSql = fmt.Sprintf("SELECT %s(%s)%s", funcName, column, sqlStr[fromIdx:])
	}

	var result any
	err = q.session.QueryRow(ctx, aggSql, args...).Scan(&result)
	return result, err
}
