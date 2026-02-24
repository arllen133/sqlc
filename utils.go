// Package sqlc provides a type-safe ORM library using generics and code generation.
// This file provides utility functions to support the core functionality of the ORM.
//
// Utility functions include:
//   - ResolveColumnNames: Extract column names from Columnar interface slice
//
// These functions are infrastructure for internal ORM implementation and are typically not called directly by external code.
package sqlc

import (
	"github.com/arllen133/sqlc/clause"
)

// ResolveColumnNames extracts column names from a slice of types implementing clause.Columnar.
// This is a general-purpose column name resolution function for converting type-safe field references to string column names.
//
// Parameters:
//   - args: Slice of objects implementing clause.Columnar interface
//
// Returns:
//   - []string: Slice of column names, returns nil if input is empty
//
// Usage scenarios:
//   - QueryBuilder.Select(): Resolve selected columns
//   - QueryBuilder.GroupBy(): Resolve group-by columns
//   - Repository.Upsert(): Resolve conflict columns and update columns
//
// clause.Columnar interface:
//   - Any type implementing ColumnName() string method can be used as parameter
//   - Common implementations: field.String, field.Number[T], clause.Column
//
// Example:
//
//	// Extract column names from field references
//	columns := ResolveColumnNames([]clause.Columnar{
//	    generated.User.ID,       // field.Number[int64]
//	    generated.User.Email,    // field.String
//	    generated.User.Name,     // field.String
//	})
//	// columns = ["id", "email", "name"]
//
//	// Use in Select
//	query.Select(generated.User.ID, generated.User.Email)
//	// Internally calls: ResolveColumnNames([]clause.Columnar{...})
//
//	// Use in GroupBy
//	query.GroupBy(generated.User.Status)
//	// Internally calls: ResolveColumnNames([]clause.Columnar{...})
//
// Performance considerations:
//   - Pre-allocates result slice to avoid multiple expansions
//   - Returns nil quickly for empty input, avoiding empty slice creation
func ResolveColumnNames(args []clause.Columnar) []string {
	// Fast return: empty input
	if len(args) == 0 {
		return nil
	}

	// Pre-allocate result slice for performance
	cols := make([]string, len(args))
	for i, arg := range args {
		// Call ColumnName() method to get column name
		cols[i] = arg.ColumnName()
	}
	return cols
}
