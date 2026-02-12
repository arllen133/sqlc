// Package sqlc provides aggregate function support for QueryBuilder.
// This file implements common SQL aggregate functions like SUM, AVG, MIN, MAX.
//
// Aggregate functions operate on a set of values and return a single summary value.
// They are commonly used with GROUP BY clauses for data analysis and reporting.
//
// Usage examples:
//
//	// Calculate total revenue
//	total, err := orderRepo.Query().Sum(ctx, generated.Order.Total)
//
//	// Calculate average age of active users
//	avgAge, err := userRepo.Query().
//	    Where(generated.User.Status.Eq("active")).
//	    Avg(ctx, generated.User.Age)
//
//	// Find minimum and maximum prices
//	minPrice, err := productRepo.Query().Min(ctx, generated.Product.Price)
//	maxPrice, err := productRepo.Query().Max(ctx, generated.Product.Price)
//
// Design considerations:
//   - All aggregate functions respect WHERE conditions and soft delete filters
//   - LIMIT and OFFSET are ignored in aggregate calculations
//   - NULL values are excluded from calculations (standard SQL behavior)
//   - Functions return appropriate Go types based on the aggregate operation
package sqlc

import (
	"context"
	"fmt"
	"strings"

	"github.com/arllen133/sqlc/clause"
)

// Sum calculates the sum of values in a numeric column.
// Returns the total sum of all non-NULL values in the specified column.
//
// Parameters:
//   - ctx: Context for cancellation and tracing
//   - column: The numeric column to sum (must implement clause.Columnar)
//
// Returns:
//   - float64: Sum of all values (0 if no rows or all NULL)
//   - error: Query execution error
//
// Usage example:
//
//	// Total order amount
//	total, err := orderRepo.Query().Sum(ctx, generated.Order.Total)
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("Total revenue: $%.2f\n", total)
//
//	// Sum with conditions
//	activeTotal, err := orderRepo.Query().
//	    Where(generated.Order.Status.Eq("completed")).
//	    Sum(ctx, generated.Order.Total)
//
//	// Sum with join
//	userTotal, err := orderRepo.Query().
//	    Join(generated.UserSchema{},
//	        sqlc.On(generated.Order.UserID, generated.User.ID),
//	    ).
//	    Where(generated.User.ID.Eq(123)).
//	    Sum(ctx, generated.Order.Total)
//
// Note:
//   - Returns 0 if no matching records found
//   - NULL values are excluded from the calculation
//   - Column should be numeric type (INT, FLOAT, DECIMAL, etc.)
//   - Result is always float64 regardless of column type
//   - Respects soft delete filter (unless WithTrashed() called)
func (q *QueryBuilder[T]) Sum(ctx context.Context, column clause.Columnar) (float64, error) {
	return q.aggregateFloat(ctx, "SUM", column.ColumnName())
}

// Avg calculates the average (mean) of values in a numeric column.
// Returns the arithmetic mean of all non-NULL values in the specified column.
//
// Parameters:
//   - ctx: Context for cancellation and tracing
//   - column: The numeric column to average (must implement clause.Columnar)
//
// Returns:
//   - float64: Average of all values (0 if no rows or all NULL)
//   - error: Query execution error
//
// Usage example:
//
//	// Average product price
//	avgPrice, err := productRepo.Query().Avg(ctx, generated.Product.Price)
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("Average price: $%.2f\n", avgPrice)
//
//	// Average with conditions
//	avgActivePrice, err := productRepo.Query().
//	    Where(generated.Product.Status.Eq("active")).
//	    Avg(ctx, generated.Product.Price)
//
//	// Average with grouping (manual approach)
//	// For grouped averages, use Select with custom SQL
//	query.
//	    Select(generated.User.Country, clause.Expr{SQL: "AVG(age)"}).
//	    GroupBy(generated.User.Country)
//
// Note:
//   - Returns 0 if no matching records found
//   - NULL values are excluded from the calculation
//   - Column should be numeric type
//   - Result is always float64
//   - For grouped averages, use Select() with raw SQL instead
//   - Respects soft delete filter (unless WithTrashed() called)
func (q *QueryBuilder[T]) Avg(ctx context.Context, column clause.Columnar) (float64, error) {
	return q.aggregateFloat(ctx, "AVG", column.ColumnName())
}

// Min finds the minimum value in a column.
// Returns the smallest non-NULL value from the specified column.
//
// Parameters:
//   - ctx: Context for cancellation and tracing
//   - column: The column to find minimum value (must implement clause.Columnar)
//
// Returns:
//   - any: Minimum value (nil if no rows or all NULL)
//   - error: Query execution error
//
// Usage example:
//
//	// Find lowest price
//	minPrice, err := productRepo.Query().Min(ctx, generated.Product.Price)
//	if err != nil {
//	    return err
//	}
//	if minPrice != nil {
//	    fmt.Printf("Lowest price: %v\n", minPrice)
//	}
//
//	// Min with conditions
//	minActivePrice, err := productRepo.Query().
//	    Where(generated.Product.Status.Eq("active")).
//	    Min(ctx, generated.Product.Price)
//
//	// Min date
//	earliestOrder, err := orderRepo.Query().
//	    Min(ctx, generated.Order.CreatedAt)
//
// Note:
//   - Returns nil if no matching records found
//   - NULL values are excluded from the calculation
//   - Works with numeric, string, and date/time columns
//   - Return type is any; type assertion may be needed
//   - Actual type depends on database driver and column type
//   - Respects soft delete filter (unless WithTrashed() called)
func (q *QueryBuilder[T]) Min(ctx context.Context, column clause.Columnar) (any, error) {
	return q.aggregateAny(ctx, "MIN", column.ColumnName())
}

// Max finds the maximum value in a column.
// Returns the largest non-NULL value from the specified column.
//
// Parameters:
//   - ctx: Context for cancellation and tracing
//   - column: The column to find maximum value (must implement clause.Columnar)
//
// Returns:
//   - any: Maximum value (nil if no rows or all NULL)
//   - error: Query execution error
//
// Usage example:
//
//	// Find highest price
//	maxPrice, err := productRepo.Query().Max(ctx, generated.Product.Price)
//	if err != nil {
//	    return err
//	}
//	if maxPrice != nil {
//	    fmt.Printf("Highest price: %v\n", maxPrice)
//	}
//
//	// Max with conditions
//	maxActivePrice, err := productRepo.Query().
//	    Where(generated.Product.Status.Eq("active")).
//	    Max(ctx, generated.Product.Price)
//
//	// Max date
//	latestOrder, err := orderRepo.Query().
//	    Max(ctx, generated.Order.CreatedAt)
//
// Note:
//   - Returns nil if no matching records found
//   - NULL values are excluded from the calculation
//   - Works with numeric, string, and date/time columns
//   - Return type is any; type assertion may be needed
//   - Actual type depends on database driver and column type
//   - Respects soft delete filter (unless WithTrashed() called)
func (q *QueryBuilder[T]) Max(ctx context.Context, column clause.Columnar) (any, error) {
	return q.aggregateAny(ctx, "MAX", column.ColumnName())
}

// aggregateFloat executes an aggregate function that returns a float64 value.
// This is an internal helper method used by Sum() and Avg().
//
// Parameters:
//   - ctx: Context for cancellation and tracing
//   - funcName: SQL aggregate function name (e.g., "SUM", "AVG")
//   - column: Column name to aggregate
//
// Returns:
//   - float64: Aggregate result (0 if no rows or NULL)
//   - error: Query execution or type conversion error
//
// Type handling:
//   - float64: returned as-is
//   - int64: converted to float64
//   - []byte: returns error (unexpected type)
//   - other types: returns error
//
// Note:
//   - This is an internal method, not part of public API
//   - Used by Sum() and Avg() for type-safe float results
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

// aggregateAny executes an aggregate function and returns the raw result.
// This is an internal helper method used by Min(), Max(), and aggregateFloat().
//
// Parameters:
//   - ctx: Context for cancellation and tracing
//   - funcName: SQL aggregate function name (e.g., "MIN", "MAX", "SUM", "AVG")
//   - column: Column name to aggregate
//
// Returns:
//   - any: Aggregate result (nil if no rows or all NULL)
//   - error: Query execution error
//
// SQL generation:
//   - Extracts FROM clause and subsequent clauses from the original query
//   - Replaces SELECT list with aggregate function
//   - Preserves WHERE, JOIN, and other clauses
//   - Example: SELECT MAX(price) FROM products WHERE status = 'active'
//
// Note:
//   - This is an internal method, not part of public API
//   - Returns raw database driver type
//   - Type depends on database driver and column type
//   - Respects all query conditions (WHERE, soft delete, etc.)
func (q *QueryBuilder[T]) aggregateAny(ctx context.Context, funcName, column string) (any, error) {
	// Add dummy column to ensure valid SQL generation
	b := q.builder.Columns("1")
	sqlStr, args, err := b.ToSql()
	if err != nil {
		return nil, err
	}

	// Build aggregate SQL
	// Start with basic aggregate query
	aggSql := fmt.Sprintf("SELECT %s(%s) FROM %s", funcName, column, q.schema.TableName())

	// Extract FROM clause and everything after it from the original query
	// This preserves WHERE, JOIN, ORDER BY, etc.
	upperSql := strings.ToUpper(sqlStr)
	fromIdx := strings.Index(upperSql, " FROM ")
	if fromIdx != -1 {
		// Replace the SELECT portion with our aggregate function
		aggSql = fmt.Sprintf("SELECT %s(%s)%s", funcName, column, sqlStr[fromIdx:])
	}

	var result any
	err = q.session.QueryRow(ctx, aggSql, args...).Scan(&result)
	return result, err
}
