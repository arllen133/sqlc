package sqlc

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/arllen133/sqlc/clause"
)

var ErrNotFound = errors.New("orm: record not found")

// QueryBuilder is a generic SQL builder for model T
type QueryBuilder[T any] struct {
	session *Session
	schema  Schema[T]
	builder sq.SelectBuilder
	columns []string
}

func Query[T any](session *Session) *QueryBuilder[T] {
	schema := LoadSchema[T]()
	sb := sq.Select().
		From(schema.TableName()).
		PlaceholderFormat(session.dialect.PlaceholderFormat())

	return &QueryBuilder[T]{
		session: session,
		schema:  schema,
		builder: sb,
	}
}

func (q *QueryBuilder[T]) Where(expr clause.Expression) *QueryBuilder[T] {
	sql, args := expr.Build()
	q.builder = q.builder.Where(sq.Expr(sql, args...))
	return q
}

type OrderBuilder interface {
	Build() string
}

func (q *QueryBuilder[T]) OrderBy(orders ...OrderBuilder) *QueryBuilder[T] {
	for _, order := range orders {
		q.builder = q.builder.OrderBy(order.Build())
	}
	return q
}

func (q *QueryBuilder[T]) Limit(n uint64) *QueryBuilder[T] {
	q.builder = q.builder.Limit(n)
	return q
}

func (q *QueryBuilder[T]) Offset(n uint64) *QueryBuilder[T] {
	q.builder = q.builder.Offset(n)
	return q
}

// Select replaces the selected columns
// arguments must implement clause.Columnar (e.g. field.Field, clause.Column)
func (q *QueryBuilder[T]) Select(columns ...clause.Columnar) *QueryBuilder[T] {
	q.columns = append(q.columns, ResolveColumnNames(columns)...)
	return q
}

// Join adds an INNER JOIN
func (q *QueryBuilder[T]) Join(table string, on clause.Expression) *QueryBuilder[T] {
	sql, args := on.Build()
	q.builder = q.builder.Join(table+" ON "+sql, args...)
	return q
}

// LeftJoin adds a LEFT JOIN
func (q *QueryBuilder[T]) LeftJoin(table string, on clause.Expression) *QueryBuilder[T] {
	sql, args := on.Build()
	q.builder = q.builder.LeftJoin(table+" ON "+sql, args...)
	return q
}

// RightJoin adds a RIGHT JOIN
func (q *QueryBuilder[T]) RightJoin(table string, on clause.Expression) *QueryBuilder[T] {
	sql, args := on.Build()
	q.builder = q.builder.RightJoin(table+" ON "+sql, args...)
	return q
}

// GroupBy adds GROUP BY clause
// arguments must implement clause.Columnar (e.g. field.Field, clause.Column)
func (q *QueryBuilder[T]) GroupBy(columns ...clause.Columnar) *QueryBuilder[T] {
	q.builder = q.builder.GroupBy(ResolveColumnNames(columns)...)
	return q
}

// Having adds HAVING clause
func (q *QueryBuilder[T]) Having(expr clause.Expression) *QueryBuilder[T] {
	sql, args := expr.Build()
	q.builder = q.builder.Having(sql, args...)
	return q
}

func (q *QueryBuilder[T]) Find(ctx context.Context) ([]*T, error) {
	// Apply columns to builder
	cols := q.columns
	if len(cols) == 0 {
		cols = q.schema.SelectColumns()
	}
	b := q.builder.Columns(cols...)
	query, args, err := b.ToSql()
	if err != nil {
		return nil, fmt.Errorf("orm: failed to build sql: %w", err)
	}

	var results []*T
	if err := q.session.Select(ctx, &results, query, args...); err != nil {
		return nil, fmt.Errorf("orm: query failed: %w", err)
	}

	return results, nil
}

// Scan executes the query and scans the results into a custom destination.
// dest can be a pointer to a struct or a pointer to a slice of structs.
// This is useful for partial selections or joins mapping to DTOs.
func (q *QueryBuilder[T]) Scan(ctx context.Context, dest any) error {
	// Apply columns to builder
	cols := q.columns
	if len(cols) == 0 {
		cols = q.schema.SelectColumns()
	}
	b := q.builder.Columns(cols...)
	query, args, err := b.ToSql()
	if err != nil {
		return fmt.Errorf("orm: failed to build sql: %w", err)
	}

	if err := q.session.Select(ctx, dest, query, args...); err != nil {
		return fmt.Errorf("orm: query failed: %w", err)
	}
	return nil
}

func (q *QueryBuilder[T]) First(ctx context.Context) (*T, error) {
	results, err := q.Limit(1).Find(ctx)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, ErrNotFound
	}
	return results[0], nil
}

func (q *QueryBuilder[T]) Count(ctx context.Context) (int64, error) {
	// Use explicit cleaner count query
	// Note: squirrel's SelectBuilder is immutable, so we can work on q.builder safely if we copy?
	// Actually sq.SelectBuilder IS a struct value, so copying it works.
	b := q.builder.Columns("COUNT(*)")

	// Remove Limit/Offset for Count
	b = b.RemoveLimit().RemoveOffset()

	query, args, err := b.ToSql()
	if err != nil {
		return 0, fmt.Errorf("orm: failed to build count sql: %w", err)
	}

	var count int64
	err = q.session.Get(ctx, &count, query, args...)
	return count, err
}

// WithBuilder allow users to manipulate the underlying squirrel.SelectBuilder.
// This provides an escape hatch for complex queries (Joins, CTEs, Window functions)
// that are not directly supported by the simplified ORM API.
func (q *QueryBuilder[T]) WithBuilder(fn func(b sq.SelectBuilder) sq.SelectBuilder) *QueryBuilder[T] {
	q.builder = fn(q.builder)
	return q
}
