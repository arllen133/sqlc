package json

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJSONPath(t *testing.T) {
	SetDefaultDialect(MySQL)

	t.Run("Eq", func(t *testing.T) {
		path := JSONPath{Column: "meta", Path: "$.count"}
		expr := path.Eq(100)
		sql, args, _ := expr.Build()

		assert.Equal(t, "JSON_EXTRACT(meta, ?) = ?", sql)
		assert.Equal(t, []any{"$.count", "100"}, args)
	})

	t.Run("Neq", func(t *testing.T) {
		path := JSONPath{Column: "meta", Path: "$.status"}
		expr := path.Neq("draft")
		sql, args, _ := expr.Build()

		assert.Equal(t, "JSON_EXTRACT(meta, ?) != ?", sql)
		assert.Equal(t, []any{"$.status", `"draft"`}, args)
	})

	t.Run("Gt", func(t *testing.T) {
		path := JSONPath{Column: "meta", Path: "$.score"}
		expr := path.Gt(50)
		sql, args, _ := expr.Build()

		assert.Equal(t, "JSON_EXTRACT(meta, ?) > ?", sql)
		assert.Equal(t, []any{"$.score", "50"}, args)
	})

	t.Run("Gte", func(t *testing.T) {
		path := JSONPath{Column: "meta", Path: "$.score"}
		expr := path.Gte(50)
		sql, args, _ := expr.Build()

		assert.Equal(t, "JSON_EXTRACT(meta, ?) >= ?", sql)
		assert.Equal(t, []any{"$.score", "50"}, args)
	})

	t.Run("Lt", func(t *testing.T) {
		path := JSONPath{Column: "meta", Path: "$.score"}
		expr := path.Lt(100)
		sql, args, _ := expr.Build()

		assert.Equal(t, "JSON_EXTRACT(meta, ?) < ?", sql)
		assert.Equal(t, []any{"$.score", "100"}, args)
	})

	t.Run("Lte", func(t *testing.T) {
		path := JSONPath{Column: "meta", Path: "$.score"}
		expr := path.Lte(100)
		sql, args, _ := expr.Build()

		assert.Equal(t, "JSON_EXTRACT(meta, ?) <= ?", sql)
		assert.Equal(t, []any{"$.score", "100"}, args)
	})

	t.Run("Contains", func(t *testing.T) {
		path := JSONPath{Column: "meta", Path: "$.tags"}
		expr := path.Contains("golang")
		sql, args, _ := expr.Build()

		assert.Equal(t, "JSON_CONTAINS(meta, ?, ?)", sql)
		assert.Equal(t, []any{`"golang"`, "$.tags"}, args)
	})

	t.Run("Set", func(t *testing.T) {
		path := JSONPath{Column: "meta", Path: "$.count"}
		assign := path.Set(200)
		sql, args, _ := assign.Build()

		assert.Equal(t, "meta = ?", sql)
		assert.Len(t, args, 1)
	})

	t.Run("Remove", func(t *testing.T) {
		path := JSONPath{Column: "meta", Path: "$.deprecated"}
		assign := path.Remove()
		sql, args, _ := assign.Build()

		assert.Equal(t, "meta = ?", sql)
		assert.Len(t, args, 1)
	})

	t.Run("Arg", func(t *testing.T) {
		path := JSONPath{Column: "meta", Path: "$.view_count"}
		arg := path.Arg(500)

		assert.Equal(t, "$.view_count", arg.Path)
		assert.Equal(t, 500, arg.Value)
	})
}

func TestJSONPathWithDialect(t *testing.T) {
	path := JSONPath{Column: "data", Path: "count"}

	t.Run("MySQL", func(t *testing.T) {
		SetDefaultDialect(MySQL)
		expr := path.With(MySQL).Eq(10)
		sql, _, _ := expr.Build()
		assert.Contains(t, sql, "JSON_EXTRACT")
	})

	t.Run("Postgres", func(t *testing.T) {
		SetDefaultDialect(Postgres)
		expr := path.With(Postgres).Eq(10)
		sql, _, _ := expr.Build()
		assert.Contains(t, sql, "#>")
	})

	t.Run("SQLite", func(t *testing.T) {
		SetDefaultDialect(SQLite)
		expr := path.With(SQLite).Eq(10)
		sql, _, _ := expr.Build()
		assert.Contains(t, sql, "json_extract")
	})

	// Reset
	SetDefaultDialect(MySQL)
}
