package field

import (
	"encoding/json"

	"github.com/arllen133/sqlc/clause"
	jsonpkg "github.com/arllen133/sqlc/field/json"
)

// JSON represents a JSON field for building SQL queries.
// It supports any type that implements json.Marshaler/Unmarshaler indirectly.
type JSON[T any] struct {
	column clause.Column
}

// Column returns the underlying column for this field
func (j JSON[T]) Column() clause.Column { return j.column }

// ColumnName implements the clause.Columnar interface
func (j JSON[T]) ColumnName() string {
	return j.column.ColumnName()
}

var _ clause.Columnar = JSON[any]{}

// WithColumn creates a new JSON field with the specified column name.
func (j JSON[T]) WithColumn(name string) JSON[T] {
	column := j.column
	column.Name = name
	return JSON[T]{column: column}
}

// WithTable creates a new JSON field with the specified table name.
func (j JSON[T]) WithTable(name string) JSON[T] {
	column := j.column
	column.Table = name
	return JSON[T]{column: column}
}

// --- Basic Query Functions ---

// IsNull creates a NULL check expression (field IS NULL).
func (j JSON[T]) IsNull() clause.Expression {
	return clause.IsNull{Column: j.column}
}

// IsNotNull creates a NOT NULL check expression (field IS NOT NULL).
func (j JSON[T]) IsNotNull() clause.Expression {
	return clause.IsNotNull{Column: j.column}
}

// --- Set Functions for UPDATE Operations ---

// Set creates an assignment expression with JSON marshaling.
func (j JSON[T]) Set(val T) clause.Assignment {
	bytes, _ := json.Marshal(val)
	return clause.Assignment{Column: j.column, Value: string(bytes)}
}

// RawSet allows setting raw JSON string/bytes directly if needed
func (j JSON[T]) RawSet(val any) clause.Assignment {
	return clause.Assignment{Column: j.column, Value: val}
}

// --- JSON Path Operations with Dialect ---

// JSONPathBuilder holds a JSON column and path for dialect-aware operations.
type JSONPathBuilder struct {
	column clause.Column
	path   string
}

// Path returns a JSONPathBuilder for operating on a specific JSON path.
// Use .With(dialect) to get dialect-specific operations.
//
// Example:
//
//	field.Metadata.Path("$.name").With(json.MySQL).Eq("alice")
//	field.Metadata.Path("name").With(json.Postgres).Eq("alice")
func (j JSON[T]) Path(path string) JSONPathBuilder {
	return JSONPathBuilder{column: j.column, path: path}
}

// With returns JSONPathOps configured with the specified dialect.
func (p JSONPathBuilder) With(dialect jsonpkg.JSONDialect) jsonpkg.JSONPathOps {
	return jsonpkg.NewPathOps(p.column, p.path, dialect)
}

// --- Builder Functions ---

// SetBuilder returns a JSONSetBuilder for constructing multi-path SET expressions.
// Use with a dialect: .SetBuilder(json.MySQL).Path(...).Path(...).Assignment(col)
//
// Example:
//
//	builder := field.Metadata.SetBuilder(json.MySQL).
//	    Path("$.name", "alice").
//	    Path("$.age", 25)
//	repo.UpdateColumns(ctx, id, builder.Assignment(field.Metadata.Column()))
func (j JSON[T]) SetBuilder(dialect jsonpkg.JSONDialect) *jsonpkg.JSONSetBuilder {
	return jsonpkg.NewSetBuilder(j.column.ColumnName(), dialect)
}

// RemoveBuilder returns a JSONRemoveBuilder for constructing multi-path REMOVE expressions.
//
// Example:
//
//	builder := field.Metadata.RemoveBuilder(json.MySQL).
//	    Path("$.temp").
//	    Path("$.cache")
//	repo.UpdateColumns(ctx, id, builder.Assignment(field.Metadata.Column()))
func (j JSON[T]) RemoveBuilder(dialect jsonpkg.JSONDialect) *jsonpkg.JSONRemoveBuilder {
	return jsonpkg.NewRemoveBuilder(j.column.ColumnName(), dialect)
}

// --- Convenience Methods Using Default Dialect ---

// PathEq creates an equality expression for this JSON path using the default dialect.
// For explicit dialect control, use Path("...").With(dialect).Eq(value).
func (j JSON[T]) PathEq(path string, value any) clause.Expression {
	sql, vars := jsonpkg.DefaultDialect().PathEq(j.column.ColumnName(), path, value)
	return clause.Expr{SQL: sql, Vars: vars}
}

// SetPath creates an assignment expression for setting a JSON path using the default dialect.
func (j JSON[T]) SetPath(path string, value any) clause.Assignment {
	return clause.Assignment{
		Column: j.column,
		Value:  jsonpkg.DefaultDialect().SetPath(j.column.ColumnName(), path, value),
	}
}

// RemovePath creates an assignment expression for removing a JSON path using the default dialect.
func (j JSON[T]) RemovePath(path string) clause.Assignment {
	return clause.Assignment{
		Column: j.column,
		Value:  jsonpkg.DefaultDialect().RemovePath(j.column.ColumnName(), path),
	}
}

// SetPaths creates an assignment expression for setting multiple JSON paths using the default dialect.
// It simplifies using SetBuilder by accepting PathValue pairs directly.
//
// Example:
//
//	repo.UpdateColumns(ctx, id, field.Metadata.SetPaths(
//	    generated.UserMetadata.Name.Arg("Alice"),
//	    generated.UserMetadata.Age.Arg(30),
//	))
func (j JSON[T]) SetPaths(args ...jsonpkg.PathValue) clause.Assignment {
	builder := j.SetBuilder(jsonpkg.DefaultDialect())
	for _, arg := range args {
		builder.Path(arg.Path, arg.Value)
	}
	return builder.Assignment(j.column)
}

// MergePatch creates a merge patch assignment (RFC 7396).
// Merges the given JSON object into the existing column value.
func (j JSON[T]) MergePatch(value any) clause.Assignment {
	return clause.Assignment{
		Column: j.column,
		Value:  jsonpkg.DefaultDialect().MergePatch(j.column.ColumnName(), value),
	}
}

// MergePreserve creates a merge preserve assignment (Legacy/Concat).
// Merges values, preserving arrays (concatenating) where supported.
func (j JSON[T]) MergePreserve(value any) clause.Assignment {
	return clause.Assignment{
		Column: j.column,
		Value:  jsonpkg.DefaultDialect().MergePreserve(j.column.ColumnName(), value),
	}
}
