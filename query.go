// Package sqlc provides a type-safe ORM library using generics and code generation.
// This file implements the QueryBuilder type, providing fluent SQL query building functionality.
//
// QueryBuilder is the query core of sqlc ORM, providing type-safe chainable query API.
// It supports all common SQL query operations, including:
//   - Conditional filtering (WHERE)
//   - Sorting (ORDER BY)
//   - Pagination (LIMIT/OFFSET)
//   - Column selection (SELECT)
//   - Deduplication (DISTINCT)
//   - Grouping (GROUP BY/HAVING)
//   - Joining (JOIN)
//   - Aggregation (COUNT)
//   - Preloading (Preload)
//   - Subqueries (Subquery)
//
// Usage examples:
//
//	// Basic query
//	users, err := userRepo.Query().Find(ctx)
//
//	// Conditional query
//	activeUsers, err := userRepo.Query().
//	    Where(generated.User.Status.Eq("active")).
//	    Where(generated.User.Age.Gt(18)).
//	    Find(ctx)
//
//	// Sorting and pagination
//	users, err := userRepo.Query().
//	    OrderBy(generated.User.CreatedAt.Desc()).
//	    Limit(10).
//	    Offset(20).
//	    Find(ctx)
//
//	// Join query
//	users, err := userRepo.Query().
//	    Join(generated.OrderSchema{},
//	        sqlc.On(generated.User.ID, generated.Order.UserID),
//	    ).
//	    Find(ctx)
//
//	// Aggregation query
//	count, err := userRepo.Query().
//	    Where(generated.User.Status.Eq("active")).
//	    Count(ctx)
//
// Design principles:
//   - Immutability: Most methods return new QueryBuilder instances
//   - Type safety: Leverages generics for compile-time type checking
//   - Composability: Build complex queries through method chaining
//   - Automation: Automatically handles soft delete, column selection, etc.
package sqlc

import (
	"context"
	"errors"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/arllen133/sqlc/clause"
)

// ErrNotFound indicates that no record was found.
// Returned when Take(), First(), Last(), FindOne() and other methods find no matching record.
//
// Usage example:
//
//	user, err := userRepo.FindOne(ctx, 123)
//	if err != nil {
//	    if errors.Is(err, sqlc.ErrNotFound) {
//	        // User not found
//	        return nil
//	    }
//	    // Other error
//	    return err
//	}
var ErrNotFound = errors.New("sqlc: record not found")

// QueryBuilder is a generic SQL query builder for model T.
// It provides a fluent chainable API to build complex SQL queries.
//
// Type parameter:
//   - T: Model type, must be registered via RegisterSchema[T]()
//
// Core features:
//   - Automatically applies soft delete filter (if model supports it)
//   - Supports all common SQL query operations
//   - Supports relation preloading
//   - Supports being used as subquery
//
// Usage example:
//
//	// Create query builder
//	query := userRepo.Query()
//
//	// Build query through chaining
//	users, err := query.
//	    Where(generated.User.Status.Eq("active")).
//	    OrderBy(generated.User.CreatedAt.Desc()).
//	    Limit(10).
//	    Find(ctx)
//
// Notes:
//   - QueryBuilder is not completely immutable, some methods modify internal state
//   - If you need to reuse query, use WithBuilder() to create a copy
//   - Soft delete filter is automatically applied on creation
type QueryBuilder[T any] struct {
	// session is the database session for executing queries
	session *Session

	// schema is the model's Schema implementation, providing table name, column names, etc.
	schema Schema[T]

	// builder is the underlying Squirrel SelectBuilder for building SQL
	builder sq.SelectBuilder

	// columns is the list of column names to select
	// If empty, uses schema.SelectColumns()
	columns []string

	// table is the main table name
	table string

	// hasJoin indicates whether the query contains JOIN operations
	// Used to decide whether to add table name prefix to column names
	hasJoin bool

	// preloads is the list of preload executors
	// Executed after main query completes, used to load associated data
	preloads []preloadExecutor[T]

	// withTrashed indicates whether to include soft-deleted records
	// By default, soft-deleted records are automatically filtered out
	withTrashed bool

	// onlyTrashed indicates whether to query only soft-deleted records
	// When set, only returns records where deleted_at IS NOT NULL
	onlyTrashed bool

	// err stores the first error that occurred during query building
	err error
}

// preloadExecutor is the function type for preload operations.
// Called after main query completes, used to load associated data.
//
// Parameters:
//   - ctx: Context for propagating cancellation signals and trace information
//   - session: Database session for executing associated queries
//   - results: Result list from main query
//
// Returns:
//   - error: Preload error
//
// Use cases:
//   - HasOne relation: Load single associated model
//   - HasMany relation: Load multiple associated models
//
// Example (internal use):
//
//	preload := func(ctx context.Context, session *Session, users []*User) error {
//	    // Load user's posts
//	    for _, user := range users {
//	        posts, err := postRepo.Query().
//	            Where(generated.Post.UserID.Eq(user.ID)).
//	            Find(ctx)
//	        if err != nil {
//	            return err
//	        }
//	        user.Posts = posts
//	    }
//	    return nil
//	}
type preloadExecutor[T any] func(ctx context.Context, session *Session, results []*T) error

// Query creates a new QueryBuilder instance.
// This is the starting point for building queries, usually called via Repository.Query().
//
// Type parameter:
//   - T: Model type, must be registered via RegisterSchema[T]()
//
// Parameters:
//   - session: Database session
//
// Returns:
//   - *QueryBuilder[T]: Initialized query builder
//
// Automatic behavior:
//   - If model supports soft delete, automatically adds deleted_at IS NULL filter
//   - Sets correct placeholder format (based on database dialect)
//   - Initializes table name and schema
//
// Usage example:
//
//	// Create via Repository (recommended)
//	query := userRepo.Query()
//
//	// Create directly (advanced usage)
//	query := sqlc.Query[models.User](session)
//
// Note:
//   - Model T must be registered, otherwise will panic
//   - Returned QueryBuilder already contains soft delete filter (if applicable)
func Query[T any](session *Session) *QueryBuilder[T] {
	// Load model's Schema
	schema := LoadSchema[T]()
	table := schema.TableName()

	// Create Squirrel SelectBuilder
	// Initially don't set columns, will be set as needed in Find()
	sb := sq.Select().
		From(table).
		PlaceholderFormat(session.dialect.PlaceholderFormat())

	// Create QueryBuilder instance
	q := &QueryBuilder[T]{
		session: session,
		schema:  schema,
		builder: sb,
		table:   table,
	}

	// If model supports soft delete, automatically add filter condition
	// This ensures deleted records are not returned by default
	if sdCol := q.schema.SoftDeleteColumn(); sdCol != "" {
		q.builder = q.builder.Where(sq.Eq{sdCol: nil})
	}

	return q
}

// Where adds WHERE condition to the query.
// Multiple calls to Where() will connect all conditions with AND.
//
// Parameters:
//   - expr: Condition expression (e.g., clause.Eq, clause.And, clause.Or, etc.)
//
// Returns:
//   - *QueryBuilder[T]: Returns itself to support chaining
//
// Supported expression types:
//   - clause.Eq: Equals (=)
//   - clause.Ne: Not equals (!=)
//   - clause.Gt: Greater than (>)
//   - clause.Gte: Greater than or equal (>=)
//   - clause.Lt: Less than (<)
//   - clause.Lte: Less than or equal (<=)
//   - clause.Like: Pattern matching (LIKE)
//   - clause.In: Contains (IN)
//   - clause.And: Logical AND
//   - clause.Or: Logical OR
//   - clause.Not: Logical NOT
//
// Usage example:
//
//	// Single condition
//	query.Where(generated.User.Status.Eq("active"))
//
//	// Multiple conditions (connected with AND)
//	query.Where(generated.User.Status.Eq("active")).
//	      Where(generated.User.Age.Gte(18))
//
//	// Complex condition
//	query.Where(clause.Or{
//	    generated.User.Role.Eq("admin"),
//	    generated.User.Role.Eq("moderator"),
//	})
//
//	// Combined condition
//	query.Where(clause.And{
//	    generated.User.Status.Eq("active"),
//	    clause.Or{
//	        generated.User.Age.Gt(18),
//	        generated.User.ParentConsent.Eq(true),
//	    },
//	})
//
// Note:
//   - Modifies current QueryBuilder instance and returns it
//   - Conditions are connected with AND
//   - Use clause.Or for OR conditions
func (q *QueryBuilder[T]) Where(expr clause.Expression) *QueryBuilder[T] {
	if q.err != nil {
		return q
	}
	// Build expression to SQL and parameters
	sql, args, err := expr.Build()
	if err != nil {
		q.err = err
		return q
	}
	// Add to WHERE clause
	q.builder = q.builder.Where(sq.Expr(sql, args...))
	return q
}

// OrderBy adds ORDER BY clause to the query.
// Supports ascending (ASC) and descending (DESC) sorting.
//
// Parameters:
//   - orders: Sort columns (variadic)
//
// Returns:
//   - *QueryBuilder[T]: Returns itself to support chaining
//
// Usage example:
//
//	// Single column sorting
//	query.OrderBy(generated.User.CreatedAt.Desc())
//
//	// Multiple column sorting
//	query.OrderBy(
//	    generated.User.Status.Asc(),
//	    generated.User.CreatedAt.Desc(),
//	)
//
//	// Using clause.OrderByColumn
//	query.OrderBy(clause.OrderByColumn{
//	    Column: generated.User.Name.Column(),
//	    Desc:   false, // Ascending
//	})
//
// Note:
//   - Multiple calls will append sort columns
//   - Asc() means ascending, Desc() means descending
func (q *QueryBuilder[T]) OrderBy(orders ...clause.OrderByColumn) *QueryBuilder[T] {
	if q.err != nil {
		return q
	}
	for _, order := range orders {
		// Build sort SQL (e.g., "created_at DESC")
		sql, _, err := order.Build()
		if err != nil {
			q.err = err
			return q
		}
		q.builder = q.builder.OrderBy(sql)
	}
	return q
}

// Limit limits the number of records returned by the query.
// Used to implement pagination or limit result set size.
//
// Parameters:
//   - n: Maximum number of records to return
//
// Returns:
//   - *QueryBuilder[T]: Returns itself to support chaining
//
// Usage example:
//
//	// Limit to 10 records
//	query.Limit(10)
//
//	// Pagination: page 3, 20 records per page
//	query.Limit(20).Offset(40)
//
// Note:
//   - 0 means no limit (some databases may not support this)
//   - Usually used with Offset() for pagination
func (q *QueryBuilder[T]) Limit(n uint64) *QueryBuilder[T] {
	q.builder = q.builder.Limit(n)
	return q
}

// Offset sets the offset for query results.
// Used to implement pagination, skipping the first N records.
//
// Parameters:
//   - n: Number of records to skip
//
// Returns:
//   - *QueryBuilder[T]: Returns itself to support chaining
//
// Usage example:
//
//	// Skip first 20 records
//	query.Offset(20)
//
//	// Pagination: page 3, 10 records per page
//	// OFFSET = (page - 1) * pageSize
//	query.Limit(10).Offset(20)
//
// Note:
//   - Usually used with Limit()
//   - Large offsets may impact performance
//   - Consider using cursor pagination instead of large offsets
func (q *QueryBuilder[T]) Offset(n uint64) *QueryBuilder[T] {
	q.builder = q.builder.Offset(n)
	return q
}

// Distinct adds DISTINCT to the SELECT clause, removing duplicate rows from results.
// Example: repo.Query().Distinct().Select(UserFields.Email).Find(ctx)
func (q *QueryBuilder[T]) Distinct() *QueryBuilder[T] {
	q.builder = q.builder.Distinct()
	return q
}

// Select replaces the selected columns
// arguments must implement clause.Columnar (e.g. field.Field, clause.Column)
func (q *QueryBuilder[T]) Select(columns ...clause.Columnar) *QueryBuilder[T] {
	q.columns = ResolveColumnNames(columns)
	return q
}

// WithTrashed includes soft-deleted records in query results.
// By default, soft-deleted records are filtered out automatically.
//
// Example:
//
//	repo.Query().WithTrashed().Find(ctx)
func (q *QueryBuilder[T]) WithTrashed() *QueryBuilder[T] {
	q.withTrashed = true
	// Remove the soft delete filter by rebuilding the query
	if sdCol := q.schema.SoftDeleteColumn(); sdCol != "" {
		// Rebuild builder without the deleted_at filter
		q.builder = sq.Select().
			From(q.table).
			PlaceholderFormat(q.session.dialect.PlaceholderFormat())
	}
	return q
}

// OnlyTrashed returns only soft-deleted records.
//
// Example:
//
//	repo.Query().OnlyTrashed().Find(ctx)
func (q *QueryBuilder[T]) OnlyTrashed() *QueryBuilder[T] {
	q.onlyTrashed = true
	q.withTrashed = true
	if sdCol := q.schema.SoftDeleteColumn(); sdCol != "" {
		// Rebuild builder with deleted_at IS NOT NULL filter
		q.builder = sq.Select().
			From(q.table).
			Where(sq.NotEq{sdCol: nil}).
			PlaceholderFormat(q.session.dialect.PlaceholderFormat())
	}
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

// Join adds an INNER JOIN clause to the query using schema-based table reference.
// This is the type-safe way to join tables using generated schema types.
//
// Parameters:
//   - target: The schema of the table to join (e.g., generated.OrderSchema{})
//   - ons: Join conditions created with On() function
//
// Returns:
//   - *QueryBuilder[T]: Returns itself to support chaining
//
// Usage example:
//
//	query.Join(generated.OrderSchema{},
//	    sqlc.On(generated.User.ID, generated.Order.UserID),
//	)
//
// Note:
//   - Multiple On() conditions are combined with AND
//   - Automatically handles table name prefixes for columns
//   - For custom table names, use JoinAs()
func (q *QueryBuilder[T]) Join(target tableNamer, ons ...JoinOn) *QueryBuilder[T] {
	return q.join(joinTypeInner, target, "", ons...)
}

// JoinAs adds an INNER JOIN clause with a custom table alias.
// Use this when joining the same table multiple times or when you need a custom alias.
//
// Parameters:
//   - target: The schema of the table to join
//   - alias: Custom alias for the joined table
//   - ons: Join conditions created with On() function
//
// Returns:
//   - *QueryBuilder[T]: Returns itself to support chaining
//
// Usage example:
//
//	// Join orders table with alias 'o'
//	query.JoinAs(generated.OrderSchema{}, "o",
//	    sqlc.On(generated.User.ID, clause.Column{Name: "user_id", Table: "o"}),
//	)
//
// Note:
//   - Column references in On() should use the alias when referring to the joined table
func (q *QueryBuilder[T]) JoinAs(target tableNamer, alias string, ons ...JoinOn) *QueryBuilder[T] {
	return q.join(joinTypeInner, target, alias, ons...)
}

// LeftJoin adds a LEFT JOIN clause to the query.
// Returns all records from the left table and matching records from the right table.
// Unmatched records will have NULL values for right table columns.
//
// Parameters:
//   - target: The schema of the table to join
//   - ons: Join conditions created with On() function
//
// Returns:
//   - *QueryBuilder[T]: Returns itself to support chaining
//
// Usage example:
//
//	// Get all users and their orders (if any)
//	query.LeftJoin(generated.OrderSchema{},
//	    sqlc.On(generated.User.ID, generated.Order.UserID),
//	)
//
// Note:
//   - Users without orders will be included with NULL order fields
//   - For custom alias, use LeftJoinAs()
func (q *QueryBuilder[T]) LeftJoin(target tableNamer, ons ...JoinOn) *QueryBuilder[T] {
	return q.join(joinTypeLeft, target, "", ons...)
}

// LeftJoinAs adds a LEFT JOIN clause with a custom table alias.
// Combines LEFT JOIN behavior with custom aliasing.
//
// Parameters:
//   - target: The schema of the table to join
//   - alias: Custom alias for the joined table
//   - ons: Join conditions created with On() function
//
// Returns:
//   - *QueryBuilder[T]: Returns itself to support chaining
//
// Usage example:
//
//	query.LeftJoinAs(generated.OrderSchema{}, "recent_orders",
//	    sqlc.On(generated.User.ID, clause.Column{Name: "user_id", Table: "recent_orders"}),
//	)
func (q *QueryBuilder[T]) LeftJoinAs(target tableNamer, alias string, ons ...JoinOn) *QueryBuilder[T] {
	return q.join(joinTypeLeft, target, alias, ons...)
}

// RightJoin adds a RIGHT JOIN clause to the query.
// Returns all records from the right table and matching records from the left table.
// Unmatched records will have NULL values for left table columns.
//
// Parameters:
//   - target: The schema of the table to join
//   - ons: Join conditions created with On() function
//
// Returns:
//   - *QueryBuilder[T]: Returns itself to support chaining
//
// Usage example:
//
//	// Get all orders and their users (if any)
//	query.RightJoin(generated.OrderSchema{},
//	    sqlc.On(generated.User.ID, generated.Order.UserID),
//	)
//
// Note:
//   - Orders without users will be included with NULL user fields
//   - Not all databases support RIGHT JOIN (e.g., SQLite)
func (q *QueryBuilder[T]) RightJoin(target tableNamer, ons ...JoinOn) *QueryBuilder[T] {
	return q.join(joinTypeRight, target, "", ons...)
}

// RightJoinAs adds a RIGHT JOIN clause with a custom table alias.
// Combines RIGHT JOIN behavior with custom aliasing.
//
// Parameters:
//   - target: The schema of the table to join
//   - alias: Custom alias for the joined table
//   - ons: Join conditions created with On() function
//
// Returns:
//   - *QueryBuilder[T]: Returns itself to support chaining
//
// Usage example:
//
//	query.RightJoinAs(generated.OrderSchema{}, "o",
//	    sqlc.On(generated.User.ID, clause.Column{Name: "user_id", Table: "o"}),
//	)
func (q *QueryBuilder[T]) RightJoinAs(target tableNamer, alias string, ons ...JoinOn) *QueryBuilder[T] {
	return q.join(joinTypeRight, target, alias, ons...)
}

// JoinTable adds an INNER JOIN clause using raw table name and expression.
// This provides maximum flexibility for complex join conditions.
//
// Parameters:
//   - table: Raw table name (can include alias, e.g., "orders o")
//   - on: Join condition as a clause.Expression
//
// Returns:
//   - *QueryBuilder[T]: Returns itself to support chaining
//
// Usage example:
//
//	// Simple join
//	query.JoinTable("orders", clause.Expr{
//	    SQL:  "users.id = orders.user_id",
//	    Vars: nil,
//	})
//
//	// Join with parameters
//	query.JoinTable("orders", clause.Expr{
//	    SQL:  "users.id = orders.user_id AND orders.total > ?",
//	    Vars: []any{100},
//	})
//
// Note:
//   - Use this for complex join conditions not supported by On()
//   - Prefer Join() with On() for type safety when possible
func (q *QueryBuilder[T]) JoinTable(table string, on clause.Expression) *QueryBuilder[T] {
	if q.err != nil {
		return q
	}
	sql, args, err := on.Build()
	if err != nil {
		q.err = err
		return q
	}
	q.builder = q.builder.Join(table+" ON "+sql, args...)
	q.hasJoin = true
	return q
}

// LeftJoinTable adds a LEFT JOIN clause using raw table name and expression.
// Combines LEFT JOIN behavior with maximum flexibility.
//
// Parameters:
//   - table: Raw table name (can include alias)
//   - on: Join condition as a clause.Expression
//
// Returns:
//   - *QueryBuilder[T]: Returns itself to support chaining
//
// Usage example:
//
//	query.LeftJoinTable("orders o", clause.Expr{
//	    SQL:  "users.id = o.user_id",
//	    Vars: nil,
//	})
func (q *QueryBuilder[T]) LeftJoinTable(table string, on clause.Expression) *QueryBuilder[T] {
	if q.err != nil {
		return q
	}
	sql, args, err := on.Build()
	if err != nil {
		q.err = err
		return q
	}
	q.builder = q.builder.LeftJoin(table+" ON "+sql, args...)
	q.hasJoin = true
	return q
}

// RightJoinTable adds a RIGHT JOIN clause using raw table name and expression.
// Combines RIGHT JOIN behavior with maximum flexibility.
//
// Parameters:
//   - table: Raw table name (can include alias)
//   - on: Join condition as a clause.Expression
//
// Returns:
//   - *QueryBuilder[T]: Returns itself to support chaining
//
// Usage example:
//
//	query.RightJoinTable("orders o", clause.Expr{
//	    SQL:  "users.id = o.user_id",
//	    Vars: nil,
//	})
func (q *QueryBuilder[T]) RightJoinTable(table string, on clause.Expression) *QueryBuilder[T] {
	if q.err != nil {
		return q
	}
	sql, args, err := on.Build()
	if err != nil {
		q.err = err
		return q
	}
	q.builder = q.builder.RightJoin(table+" ON "+sql, args...)
	q.hasJoin = true
	return q
}

// GroupBy adds GROUP BY clause to the query for aggregation.
// Used with aggregate functions like COUNT, SUM, AVG, MAX, MIN.
//
// Parameters:
//   - columns: Columns to group by (must implement clause.Columnar)
//
// Returns:
//   - *QueryBuilder[T]: Returns itself to support chaining
//
// Usage example:
//
//	// Group by single column
//	query.GroupBy(generated.User.Status)
//
//	// Group by multiple columns
//	query.GroupBy(
//	    generated.User.Status,
//	    generated.User.Country,
//	)
//
//	// With aggregation
//	query.
//	    Select(generated.User.Status, clause.Count("*")).
//	    GroupBy(generated.User.Status)
//
// Note:
//   - Usually combined with Select() to choose aggregate columns
//   - Use Having() to filter grouped results
//   - Arguments must implement clause.Columnar (e.g., field.Field, clause.Column)
func (q *QueryBuilder[T]) GroupBy(columns ...clause.Columnar) *QueryBuilder[T] {
	q.builder = q.builder.GroupBy(ResolveColumnNames(columns)...)
	return q
}

// Having adds HAVING clause to filter grouped results.
// Similar to WHERE but operates on groups after aggregation.
//
// Parameters:
//   - expr: Filter condition expression
//
// Returns:
//   - *QueryBuilder[T]: Returns itself to support chaining
//
// Usage example:
//
//	// Filter groups with count > 5
//	query.
//	    Select(generated.User.Country, clause.Count("*")).
//	    GroupBy(generated.User.Country).
//	    Having(clause.Gt{clause.Count("*"), 5})
//
//	// Multiple conditions
//	query.
//	    GroupBy(generated.User.Status).
//	    Having(clause.And{
//	        clause.Gt{clause.Count("*"), 10},
//	        clause.Eq{generated.User.Active, true},
//	    })
//
// Note:
//   - Must be used after GroupBy()
//   - Can reference aggregate functions in conditions
//   - Conditions are applied after grouping, not before
func (q *QueryBuilder[T]) Having(expr clause.Expression) *QueryBuilder[T] {
	if q.err != nil {
		return q
	}
	sql, args, err := expr.Build()
	if err != nil {
		q.err = err
		return q
	}
	q.builder = q.builder.Having(sql, args...)
	return q
}

// WithPreload adds a preload executor to load related data after the main query.
// Use with Preload() function to create type-safe preload executors.
func (q *QueryBuilder[T]) WithPreload(preload preloadExecutor[T]) *QueryBuilder[T] {
	q.preloads = append(q.preloads, preload)
	return q
}

// Find executes the query and returns all matching records.
// This is the primary method for retrieving multiple records from the database.
//
// Parameters:
//   - ctx: Context for cancellation and tracing
//
// Returns:
//   - []*T: Slice of model pointers (empty slice if no results)
//   - error: Query execution error
//
// Automatic behavior:
//   - Executes all registered preloads after main query
//   - Applies soft delete filter (unless WithTrashed() called)
//   - Resolves column names from schema if not explicitly set
//
// Usage example:
//
//	// Get all users
//	users, err := userRepo.Query().Find(ctx)
//
//	// Get active users
//	activeUsers, err := userRepo.Query().
//	    Where(generated.User.Status.Eq("active")).
//	    Find(ctx)
//
//	// With preloading
//	users, err := userRepo.Query().
//	    WithPreload(sqlc.Preload(generated.User.Posts)).
//	    Find(ctx)
//
// Note:
//   - Returns empty slice (not nil) if no records found
//   - Preloads are executed in the order they were added
//   - Context is propagated to all database operations
func (q *QueryBuilder[T]) Find(ctx context.Context) ([]*T, error) {
	if q.err != nil {
		return nil, q.err
	}
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

// Pluck queries a single column and returns the values as a slice.
// dest must be a pointer to a slice of the appropriate type (e.g., *[]string, *[]int64).
// This is useful for extracting single column values without loading full models.
//
// Example:
//
//	var emails []string
//	userRepo.Query().Where(generated.User.Active.Eq(true)).Pluck(ctx, generated.User.Email, &emails)
func (q *QueryBuilder[T]) Pluck(ctx context.Context, column clause.Columnar, dest any) error {
	if q.err != nil {
		return q.err
	}
	colName := column.ColumnName()
	b := q.builder.Columns(colName)
	query, args, err := b.ToSql()
	if err != nil {
		return fmt.Errorf("sqlc: failed to build sql: %w", err)
	}

	if err := q.session.Select(ctx, dest, query, args...); err != nil {
		return fmt.Errorf("sqlc: pluck failed: %w", err)
	}

	return nil
}

// Chunk processes query results in batches of the specified size.
// This is useful for processing large datasets without loading everything into memory.
// The callback function receives each batch of records; if it returns an error,
// chunking stops and the error is returned.
//
// Example:
//
//	err := userRepo.Query().Where(generated.User.Active.Eq(true)).Chunk(ctx, 100, func(users []*models.User) error {
//	    for _, u := range users {
//	        processUser(u)
//	    }
//	    return nil
//	})
func (q *QueryBuilder[T]) Chunk(ctx context.Context, size int, fn func([]*T) error) error {
	if size <= 0 {
		return fmt.Errorf("sqlc: chunk size must be positive, got %d", size)
	}

	offset := uint64(0)
	for {
		// Create a new query for each chunk to avoid mutation issues
		chunkQuery := Query[T](q.session)
		chunkQuery.table = q.table
		chunkQuery.schema = q.schema
		chunkQuery.columns = q.columns
		chunkQuery.hasJoin = q.hasJoin
		chunkQuery.preloads = q.preloads

		// Copy the builder state
		chunkQuery.builder = q.builder.Limit(uint64(size)).Offset(offset)

		results, err := chunkQuery.Find(ctx)
		if err != nil {
			return err
		}

		if len(results) == 0 {
			break
		}

		if err := fn(results); err != nil {
			return err
		}

		if len(results) < size {
			break // Last batch
		}

		offset += uint64(size)
	}

	return nil
}

// Scan executes the query and scans the results into a custom destination.
// dest can be a pointer to a struct or a pointer to a slice of structs.
// This is useful for partial selections or joins mapping to DTOs.
func (q *QueryBuilder[T]) Scan(ctx context.Context, dest any) error {
	if q.err != nil {
		return q.err
	}
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

// Take executes the query and returns a single record without any ordering.
// Returns ErrNotFound if no record matches the query conditions.
//
// Parameters:
//   - ctx: Context for cancellation and tracing
//
// Returns:
//   - *T: Single model instance
//   - error: ErrNotFound if no record, or other query error
//
// Usage example:
//
//	// Get any active user
//	user, err := userRepo.Query().
//	    Where(generated.User.Status.Eq("active")).
//	    Take(ctx)
//	if errors.Is(err, sqlc.ErrNotFound) {
//	    // No active user found
//	}
//
// Note:
//   - Adds LIMIT 1 to the query
//   - Does not guarantee which record is returned if multiple match
//   - Use First() or Last() for deterministic ordering
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

// First executes the query and returns the first record ordered by primary key ascending.
// Returns ErrNotFound if no record matches the query conditions.
//
// Parameters:
//   - ctx: Context for cancellation and tracing
//
// Returns:
//   - *T: First model instance (by primary key)
//   - error: ErrNotFound if no record, or other query error
//
// Usage example:
//
//	// Get first user by ID
//	user, err := userRepo.Query().First(ctx)
//
//	// Get first active user
//	user, err := userRepo.Query().
//	    Where(generated.User.Status.Eq("active")).
//	    First(ctx)
//
// Note:
//   - Orders by primary key ascending
//   - Adds LIMIT 1 to the query
//   - For custom ordering, use OrderBy().Take()
func (q *QueryBuilder[T]) First(ctx context.Context) (*T, error) {
	pk := q.schema.PK(nil).Column
	if pk.Table == "" {
		pk.Table = q.table
	}
	return q.OrderBy(clause.OrderByColumn{Column: pk, Desc: false}).Take(ctx)
}

// Last executes the query and returns the last record ordered by primary key descending.
// Returns ErrNotFound if no record matches the query conditions.
//
// Parameters:
//   - ctx: Context for cancellation and tracing
//
// Returns:
//   - *T: Last model instance (by primary key)
//   - error: ErrNotFound if no record, or other query error
//
// Usage example:
//
//	// Get last user by ID
//	user, err := userRepo.Query().Last(ctx)
//
//	// Get last active user
//	user, err := userRepo.Query().
//	    Where(generated.User.Status.Eq("active")).
//	    Last(ctx)
//
// Note:
//   - Orders by primary key descending
//   - Adds LIMIT 1 to the query
//   - For custom ordering, use OrderBy().Take()
func (q *QueryBuilder[T]) Last(ctx context.Context) (*T, error) {
	pk := q.schema.PK(nil).Column
	if pk.Table == "" {
		pk.Table = q.table
	}
	return q.OrderBy(clause.OrderByColumn{Column: pk, Desc: true}).Take(ctx)
}

// FirstOr returns the first matching record, or executes the fallback function
// if no record is found (ErrNotFound).
//
// Example:
//
//	user, err := userRepo.Query().Where(generated.User.Email.Eq("test@example.com")).FirstOr(ctx, func() *models.User {
//	    return &models.User{Email: "test@example.com", Name: "Default"}
//	})
func (q *QueryBuilder[T]) FirstOr(ctx context.Context, fallback func() *T) (*T, error) {
	result, err := q.Take(ctx)
	if err == nil {
		return result, nil
	}
	if errors.Is(err, ErrNotFound) {
		return fallback(), nil
	}
	return nil, err
}

// FirstOrCreate returns the first matching record, or returns the provided defaults
// if no record is found. For actual creation, use Repository.FirstOrCreate.
//
// Example:
//
//	user, err := userRepo.Query().Where(generated.User.Email.Eq("test@example.com")).FirstOrCreate(ctx, &models.User{
//	    Email: "test@example.com",
//	    Name:  "New User",
//	})
func (q *QueryBuilder[T]) FirstOrCreate(ctx context.Context, defaults *T) (*T, error) {
	result, err := q.Take(ctx)
	if err == nil {
		return result, nil
	}
	if errors.Is(err, ErrNotFound) {
		return defaults, nil
	}
	return nil, err
}

// Count returns the number of records matching the query conditions.
// Ignores any Limit/Offset settings to count all matching records.
//
// Parameters:
//   - ctx: Context for cancellation and tracing
//
// Returns:
//   - int64: Number of matching records
//   - error: Query execution error
//
// Usage example:
//
//	// Count all users
//	count, err := userRepo.Query().Count(ctx)
//
//	// Count active users
//	count, err := userRepo.Query().
//	    Where(generated.User.Status.Eq("active")).
//	    Count(ctx)
//
//	// Count with join
//	count, err := userRepo.Query().
//	    Join(generated.OrderSchema{},
//	        sqlc.On(generated.User.ID, generated.Order.UserID),
//	    ).
//	    Count(ctx)
//
// Note:
//   - Generates SELECT COUNT(*) FROM ...
//   - Removes LIMIT and OFFSET from count query
//   - Respects soft delete filter (unless WithTrashed() called)
//   - Does not execute preloads
func (q *QueryBuilder[T]) Count(ctx context.Context) (int64, error) {
	if q.err != nil {
		return 0, q.err
	}
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
func (q *QueryBuilder[T]) Build() (string, []any, error) {
	return q.ToSQL()
}

// BuildE is like Build but returns an error explicitly instead of embedding it in SQL.
// Use this when you need to handle query building errors before execution.
//
// Deprecated: Use Build() instead.
func (q *QueryBuilder[T]) BuildE() (string, []any, error) {
	return q.ToSQL()
}

// ToSQL returns the SQL string and arguments without executing the query.
// This is useful for testing, debugging, or logging generated SQL.
func (q *QueryBuilder[T]) ToSQL() (string, []any, error) {
	if q.err != nil {
		return "", nil, q.err
	}
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
