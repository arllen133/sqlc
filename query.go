package sqlc

import (
	"context"
	"errors"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/arllen133/sqlc/clause"
)

var ErrNotFound = errors.New("sqlc: record not found")

// QueryBuilder is a generic SQL builder for model T
type QueryBuilder[T any] struct {
	session  *Session
	schema   Schema[T]
	builder  sq.SelectBuilder
	columns  []string
	table    string
	hasJoin  bool
	preloads []preloadExecutor[T]
}

// preloadExecutor executes a preload operation after the main query
type preloadExecutor[T any] func(ctx context.Context, session *Session, results []*T) error

func Query[T any](session *Session) *QueryBuilder[T] {
	schema := LoadSchema[T]()
	table := schema.TableName()
	sb := sq.Select().
		From(table).
		PlaceholderFormat(session.dialect.PlaceholderFormat())

	return &QueryBuilder[T]{
		session: session,
		schema:  schema,
		builder: sb,
		table:   table,
	}
}

func (q *QueryBuilder[T]) Where(expr clause.Expression) *QueryBuilder[T] {
	sql, args := expr.Build()
	q.builder = q.builder.Where(sq.Expr(sql, args...))
	return q
}

func (q *QueryBuilder[T]) OrderBy(orders ...clause.OrderByColumn) *QueryBuilder[T] {
	for _, order := range orders {
		sql, _ := order.Build()
		q.builder = q.builder.OrderBy(sql)
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
	q.columns = ResolveColumnNames(columns)
	return q
}

type tableNamer interface {
	TableName() string
}

type joinType int

const (
	joinTypeInner joinType = iota
	joinTypeLeft
	joinTypeRight
)

func (q *QueryBuilder[T]) join(joinType joinType, target tableNamer, alias string, ons ...JoinOn) *QueryBuilder[T] {
	if len(ons) == 0 {
		return q
	}

	joinTable := target.TableName()
	joinTableRef := joinTable
	joinColumnTable := joinTable
	if alias != "" {
		joinTableRef = joinTable + " " + alias
		joinColumnTable = alias
	}

	onParts := make([]string, 0, len(ons))
	for _, on := range ons {
		left := on.Left
		right := on.Right
		if left.Table == "" {
			left.Table = q.table
		}
		if right.Table == "" {
			right.Table = joinColumnTable
		}
		onParts = append(onParts, left.ColumnName()+" = "+right.ColumnName())
	}

	onSQL := strings.Join(onParts, " AND ")
	switch joinType {
	case joinTypeLeft:
		q.builder = q.builder.LeftJoin(joinTableRef + " ON " + onSQL)
	case joinTypeRight:
		q.builder = q.builder.RightJoin(joinTableRef + " ON " + onSQL)
	default:
		q.builder = q.builder.Join(joinTableRef + " ON " + onSQL)
	}
	q.hasJoin = true
	return q
}

func (q *QueryBuilder[T]) Join(target tableNamer, ons ...JoinOn) *QueryBuilder[T] {
	return q.join(joinTypeInner, target, "", ons...)
}

func (q *QueryBuilder[T]) JoinAs(target tableNamer, alias string, ons ...JoinOn) *QueryBuilder[T] {
	return q.join(joinTypeInner, target, alias, ons...)
}

func (q *QueryBuilder[T]) LeftJoin(target tableNamer, ons ...JoinOn) *QueryBuilder[T] {
	return q.join(joinTypeLeft, target, "", ons...)
}

func (q *QueryBuilder[T]) LeftJoinAs(target tableNamer, alias string, ons ...JoinOn) *QueryBuilder[T] {
	return q.join(joinTypeLeft, target, alias, ons...)
}

func (q *QueryBuilder[T]) RightJoin(target tableNamer, ons ...JoinOn) *QueryBuilder[T] {
	return q.join(joinTypeRight, target, "", ons...)
}

func (q *QueryBuilder[T]) RightJoinAs(target tableNamer, alias string, ons ...JoinOn) *QueryBuilder[T] {
	return q.join(joinTypeRight, target, alias, ons...)
}

func (q *QueryBuilder[T]) JoinTable(table string, on clause.Expression) *QueryBuilder[T] {
	sql, args := on.Build()
	q.builder = q.builder.Join(table+" ON "+sql, args...)
	q.hasJoin = true
	return q
}

func (q *QueryBuilder[T]) LeftJoinTable(table string, on clause.Expression) *QueryBuilder[T] {
	sql, args := on.Build()
	q.builder = q.builder.LeftJoin(table+" ON "+sql, args...)
	q.hasJoin = true
	return q
}

func (q *QueryBuilder[T]) RightJoinTable(table string, on clause.Expression) *QueryBuilder[T] {
	sql, args := on.Build()
	q.builder = q.builder.RightJoin(table+" ON "+sql, args...)
	q.hasJoin = true
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

// WithPreload adds a preload executor to load related data after the main query.
// Use with Preload() function to create type-safe preload executors.
func (q *QueryBuilder[T]) WithPreload(preload preloadExecutor[T]) *QueryBuilder[T] {
	q.preloads = append(q.preloads, preload)
	return q
}

func (q *QueryBuilder[T]) Find(ctx context.Context) ([]*T, error) {
	b := q.builder.Columns(q.resolveColumns()...)
	query, args, err := b.ToSql()
	if err != nil {
		return nil, fmt.Errorf("sqlc: failed to build sql: %w", err)
	}

	var results []*T
	if err := q.session.Select(ctx, &results, query, args...); err != nil {
		return nil, fmt.Errorf("sqlc: query failed: %w", err)
	}

	// Execute preloads
	for _, preload := range q.preloads {
		if err := preload(ctx, q.session, results); err != nil {
			return nil, fmt.Errorf("sqlc: preload failed: %w", err)
		}
	}

	return results, nil
}

// Scan executes the query and scans the results into a custom destination.
// dest can be a pointer to a struct or a pointer to a slice of structs.
// This is useful for partial selections or joins mapping to DTOs.
func (q *QueryBuilder[T]) Scan(ctx context.Context, dest any) error {
	// Apply columns to builder
	b := q.builder.Columns(q.resolveColumns()...)
	query, args, err := b.ToSql()
	if err != nil {
		return fmt.Errorf("sqlc: failed to build sql: %w", err)
	}

	if err := q.session.Select(ctx, dest, query, args...); err != nil {
		return fmt.Errorf("sqlc: query failed: %w", err)
	}
	return nil
}

func (q *QueryBuilder[T]) Take(ctx context.Context) (*T, error) {
	results, err := q.Limit(1).Find(ctx)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, ErrNotFound
	}
	return results[0], nil
}

func (q *QueryBuilder[T]) First(ctx context.Context) (*T, error) {
	pk := q.schema.PK(nil).Column
	if pk.Table == "" {
		pk.Table = q.table
	}
	return q.OrderBy(clause.OrderByColumn{Column: pk, Desc: false}).Take(ctx)
}

func (q *QueryBuilder[T]) Last(ctx context.Context) (*T, error) {
	pk := q.schema.PK(nil).Column
	if pk.Table == "" {
		pk.Table = q.table
	}
	return q.OrderBy(clause.OrderByColumn{Column: pk, Desc: true}).Take(ctx)
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
		return 0, fmt.Errorf("sqlc: failed to build count sql: %w", err)
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

// Build implements clause.Expression, enabling QueryBuilder to be used as a subquery.
// This allows nesting queries in WHERE clauses like: WHERE id IN (SELECT ...)
func (q *QueryBuilder[T]) Build() (string, []any) {
	sql, args, err := q.ToSQL()
	if err != nil {
		// Embed error in SQL comment - will fail gracefully at DB execution
		return "/* ERROR: " + err.Error() + " */", nil
	}
	return "(" + sql + ")", args
}

// ToSQL returns the SQL string and arguments without executing the query.
// This is useful for testing, debugging, or logging generated SQL.
func (q *QueryBuilder[T]) ToSQL() (string, []any, error) {
	b := q.builder.Columns(q.resolveColumns()...)
	return b.ToSql()
}

func (q *QueryBuilder[T]) resolveColumns() []string {
	cols := q.columns
	if len(cols) == 0 {
		cols = q.schema.SelectColumns()
		if q.hasJoin {
			qualified := make([]string, len(cols))
			for i, col := range cols {
				qualified[i] = q.table + "." + col
			}
			cols = qualified
		}
	}
	return cols
}
