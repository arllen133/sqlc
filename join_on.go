// Package sqlc provides a type-safe ORM library using generics and code generation.
// This file implements JOIN query helper functions and subquery expressions.
//
// JOIN is a core feature of relational databases for associating data across multiple tables.
// sqlc provides type-safe JOIN support, including:
//   - INNER JOIN: Inner join, returns only matching records
//   - LEFT JOIN: Left join, returns all records from left table and matching records from right table
//   - RIGHT JOIN: Right join, returns all records from right table and matching records from left table
//
// Usage example:
//
//	// INNER JOIN
//	users, err := userRepo.Query().
//	    Join(generated.OrderSchema{},
//	        sqlc.On(generated.User.ID, generated.Order.UserID),
//	    ).
//	    Find(ctx)
//
//	// LEFT JOIN with alias
//	users, err := userRepo.Query().
//	    LeftJoinAs(generated.ProfileSchema{}, "p",
//	        sqlc.On(generated.User.ID, clause.Column{Name: "user_id", Table: "p"}),
//	    ).
//	    Find(ctx)
//
//	// EXISTS subquery
//	activeUsers, err := userRepo.Query().
//	    Where(sqlc.Exists(
//	        orderRepo.Query().
//	            Select(generated.Order.ID).
//	            Where(clause.Eq{
//	                Column: generated.Order.UserID,
//	                Value:  generated.User.ID.Column(),
//	            }),
//	    )).
//	    Find(ctx)
package sqlc

import "github.com/arllen133/sqlc/clause"

// JoinOn defines the column correspondence in JOIN conditions.
// Used to specify the ON clause of JOIN, connecting columns from left and right tables.
//
// Field descriptions:
//   - Left: Column from left table (usually main table or already joined table)
//   - Right: Column from right table (table to be joined)
//
// Usage example:
//
//	// users.id = orders.user_id
//	joinOn := JoinOn{
//	    Left:  clause.Column{Name: "id", Table: "users"},
//	    Right: clause.Column{Name: "user_id", Table: "orders"},
//	}
//
// Note:
//   - Usually created using On() function instead of direct construction
//   - Supports multiple JoinOn conditions (connected with AND)
type JoinOn struct {
	// Left is the column on the left side of JOIN condition
	Left clause.Column

	// Right is the column on the right side of JOIN condition
	Right clause.Column
}

// On creates a JOIN condition.
// This is the recommended way to build JOIN ON clauses.
//
// Parameters:
//   - left: Column from left table (must implement Column() method, e.g., field.Field)
//   - right: Column from right table (must implement Column() method)
//
// Returns:
//   - JoinOn: JOIN condition
//
// Usage example:
//
//	// Basic usage
//	joinOn := sqlc.On(generated.User.ID, generated.Order.UserID)
//
//	// Use in JOIN
//	users, err := userRepo.Query().
//	    Join(generated.OrderSchema{},
//	        sqlc.On(generated.User.ID, generated.Order.UserID),
//	    ).
//	    Find(ctx)
//
//	// Multiple conditions (connected with AND)
//	users, err := userRepo.Query().
//	    Join(generated.OrderSchema{},
//	        sqlc.On(generated.User.ID, generated.Order.UserID),
//	        sqlc.On(generated.User.TenantID, generated.Order.TenantID),
//	    ).
//	    Find(ctx)
//
// Type constraints:
//   - left and right must implement Column() clause.Column method
//   - Usually generated field.Field types
func On(left, right interface{ Column() clause.Column }) JoinOn {
	return JoinOn{
		Left:  left.Column(),
		Right: right.Column(),
	}
}

// Exists creates an EXISTS subquery expression.
// Used to check if a subquery returns any records.
//
// EXISTS syntax:
//
//	WHERE EXISTS (SELECT ... FROM ... WHERE ...)
//
// Parameters:
//   - expr: Subquery expression (usually QueryBuilder)
//
// Returns:
//   - clause.Expression: EXISTS expression
//
// Use cases:
//   - Check if related records exist
//   - Filter main records with/without related records
//   - Implement complex business rules
//
// Usage example:
//
//	// Query users who have orders
//	usersWithOrders, err := userRepo.Query().
//	    Where(sqlc.Exists(
//	        orderRepo.Query().
//	            Select(generated.Order.ID).
//	            Where(clause.Eq{
//	                Column: generated.Order.UserID,
//	                Value:  generated.User.ID.Column(),
//	            }),
//	    )).
//	    Find(ctx)
//
//	// Query users who have pending orders
//	usersWithPendingOrders, err := userRepo.Query().
//	    Where(sqlc.Exists(
//	        orderRepo.Query().
//	            Select(generated.Order.ID).
//	            Where(clause.And{
//	                clause.Eq{Column: generated.Order.UserID, Value: generated.User.ID.Column()},
//	                clause.Eq{Column: generated.Order.Status, Value: "pending"},
//	            }),
//	    )).
//	    Find(ctx)
//
// Performance notes:
//   - EXISTS stops scanning after finding the first matching record
//   - Usually more efficient than IN or JOIN (for existence checks)
//   - Database optimizer automatically chooses the best execution plan
func Exists(expr clause.Expression) clause.Expression {
	return clause.ExistsExpr{Expr: expr}
}

// NotExists creates a NOT EXISTS subquery expression.
// Used to check if a subquery returns no records.
//
// NOT EXISTS syntax:
//
//	WHERE NOT EXISTS (SELECT ... FROM ... WHERE ...)
//
// Parameters:
//   - expr: Subquery expression (usually QueryBuilder)
//
// Returns:
//   - clause.Expression: NOT EXISTS expression
//
// Use cases:
//   - Check if related records don't exist
//   - Filter main records without related records
//   - Implement exclusive business rules
//
// Usage example:
//
//	// Query users who have no orders
//	usersWithoutOrders, err := userRepo.Query().
//	    Where(sqlc.NotExists(
//	        orderRepo.Query().
//	            Select(generated.Order.ID).
//	            Where(clause.Eq{
//	                Column: generated.Order.UserID,
//	                Value:  generated.User.ID.Column(),
//	            }),
//	    )).
//	    Find(ctx)
//
//	// Query users who have no unread messages
//	usersWithNoUnread, err := userRepo.Query().
//	    Where(sqlc.NotExists(
//	        messageRepo.Query().
//	            Select(generated.Message.ID).
//	            Where(clause.And{
//	                clause.Eq{Column: generated.Message.UserID, Value: generated.User.ID.Column()},
//	                clause.Eq{Column: generated.Message.Read, Value: false},
//	            }),
//	    )).
//	    Find(ctx)
//
// Performance notes:
//   - NOT EXISTS must scan entire subquery result
//   - For large tables, may be slower than LEFT JOIN + IS NULL
//   - Ensure appropriate indexes exist on subquery
func NotExists(expr clause.Expression) clause.Expression {
	return clause.NotExistsExpr{Expr: expr}
}
