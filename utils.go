// Package sqlc provides a type-safe ORM library using generics and code generation.
// This file provides utility functions to support the core functionality of the ORM.
//
// Utility functions include:
//   - ResolveColumnNames: Extract column names from Columnar interface slice
//   - getFieldValue: Extract specified column value from a struct
//
// These functions are infrastructure for internal ORM implementation and are typically not called directly by external code.
package sqlc

import (
	"reflect"
	"strings"

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

// getFieldValue extracts a field value from a struct by column name.
// This is an internal function used to extract foreign key values from models during relation loading.
//
// Parameters:
//   - v: Any type value (usually a pointer to a struct)
//   - columnName: Database column name
//
// Returns:
//   - any: Field value, returns nil if not found
//
// Matching rules:
//  1. First checks struct tag `db:"column_name"`, supports format with options (e.g., `db:"name,primaryKey"`)
//  2. If no db tag, matches by field name (case-insensitive)
//
// Supported types:
//   - Pointer types: Automatically dereferenced
//   - Struct types: Iterates through fields to find match
//   - Other types: Returns nil
//
// Usage scenarios:
//   - Relation loading: Extract foreign key values from child models
//   - Data mapping: Map model fields to database columns
//
// Example:
//
//	type User struct {
//	    ID       int64  `db:"id,primaryKey"`
//	    Email    string `db:"email"`
//	    Name     string `db:"name"`
//	    Password string `db:"password_hash"` // Column name differs from field name
//	}
//
//	user := &User{ID: 123, Email: "test@example.com", Name: "Test"}
//
//	// Match via db tag
//	id := getFieldValue(user, "id")           // 123 (int64)
//	email := getFieldValue(user, "email")     // "test@example.com"
//	pwd := getFieldValue(user, "password_hash") // "" (empty string)
//
//	// Match via field name (case-insensitive)
//	name := getFieldValue(user, "Name")       // "Test"
//	name2 := getFieldValue(user, "NAME")      // "Test"
//
//	// Not found
//	unknown := getFieldValue(user, "unknown") // nil
//
// Notes:
//   - For pointer types, automatically dereferences
//   - If pointer is nil, returns nil (reflect.Value.IsZero)
//   - Only supports exported fields (capitalized first letter)
//   - Does not support recursive lookup in nested structs
//   - Does not support complex types like map, slice, etc.
//
// Implementation details:
//   - Uses reflect package for runtime type checking
//   - Parses db tag with support for comma-separated options
//   - Field name matching uses strings.EqualFold (case-insensitive)
func getFieldValue(v any, columnName string) any {
	// Get reflection value
	val := reflect.ValueOf(v)

	// If pointer, dereference it
	if val.Kind() == reflect.Pointer {
		val = val.Elem()
	}

	// Ensure it's a struct type
	if val.Kind() != reflect.Struct {
		return nil
	}

	// Get struct type information
	typ := val.Type()

	// Iterate through all fields
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// Priority: check db tag
		// Format: `db:"column_name"` or `db:"column_name,options"`
		if dbTag := field.Tag.Get("db"); dbTag != "" {
			// Parse db tag, supports comma-separated options
			// Example: "id,primaryKey" -> ["id", "primaryKey"]
			parts := strings.Split(dbTag, ",")

			// Check if column name matches
			if parts[0] == columnName {
				// Return field value
				return val.Field(i).Interface()
			}
		}

		// Fallback: match by field name (case-insensitive)
		// This allows working without db tags
		if strings.EqualFold(field.Name, columnName) {
			return val.Field(i).Interface()
		}
	}

	// No matching field found
	return nil
}
