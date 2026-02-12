package json

import (
	"testing"

	"github.com/arllen133/sqlc/clause"
	"github.com/stretchr/testify/assert"
)

func TestJSONSetBuilder(t *testing.T) {
	SetDefaultDialect(MySQL)

	t.Run("Single path", func(t *testing.T) {
		builder := NewSetBuilder("metadata", MySQL)
		builder.Path("$.count", 100)

		expr := builder.Build()
		assert.Contains(t, expr.SQL, "JSON_SET")
		assert.Len(t, expr.Vars, 2)
	})

	t.Run("Multiple paths", func(t *testing.T) {
		builder := NewSetBuilder("metadata", MySQL)
		builder.Path("$.count", 100)
		builder.Path("$.active", true)

		expr := builder.Build()
		assert.Contains(t, expr.SQL, "JSON_SET")
		assert.Len(t, expr.Vars, 4)
	})

	t.Run("Empty builder", func(t *testing.T) {
		builder := NewSetBuilder("metadata", MySQL)
		expr := builder.Build()
		assert.Empty(t, expr.SQL)
	})

	t.Run("Chained calls", func(t *testing.T) {
		builder := NewSetBuilder("data", MySQL).
			Path("$.a", 1).
			Path("$.b", 2).
			Path("$.c", 3)

		expr := builder.Build()
		assert.Contains(t, expr.SQL, "JSON_SET")
		assert.Len(t, expr.Vars, 6)
	})
}

func TestJSONRemoveBuilder(t *testing.T) {
	SetDefaultDialect(MySQL)

	t.Run("Single path", func(t *testing.T) {
		builder := NewRemoveBuilder("metadata", MySQL)
		builder.Path("$.deprecated")

		expr := builder.Build()
		assert.Contains(t, expr.SQL, "JSON_REMOVE")
		assert.Len(t, expr.Vars, 1)
	})

	t.Run("Multiple paths", func(t *testing.T) {
		builder := NewRemoveBuilder("metadata", MySQL)
		builder.Path("$.old_field")
		builder.Path("$.legacy")

		expr := builder.Build()
		assert.Contains(t, expr.SQL, "JSON_REMOVE")
		assert.Len(t, expr.Vars, 2)
	})

	t.Run("Empty builder", func(t *testing.T) {
		builder := NewRemoveBuilder("metadata", MySQL)
		expr := builder.Build()
		assert.Empty(t, expr.SQL)
	})
}

func TestSetBuilderAssignment(t *testing.T) {
	SetDefaultDialect(MySQL)
	col := clause.Column{Name: "meta"}

	builder := NewSetBuilder("meta", MySQL)
	builder.Path("$.views", 500)

	assign := builder.Assignment(col)
	sql, args, _ := assign.Build()

	assert.Equal(t, "meta = ?", sql)
	assert.Len(t, args, 1)
}

func TestRemoveBuilderAssignment(t *testing.T) {
	SetDefaultDialect(MySQL)
	col := clause.Column{Name: "meta"}

	builder := NewRemoveBuilder("meta", MySQL)
	builder.Path("$.old")

	assign := builder.Assignment(col)
	sql, args, _ := assign.Build()

	assert.Equal(t, "meta = ?", sql)
	assert.Len(t, args, 1)
}

func TestBuilderWithDifferentDialects(t *testing.T) {
	t.Run("MySQL SetBuilder", func(t *testing.T) {
		builder := NewSetBuilder("data", MySQL)
		builder.Path("$.key", "value")
		expr := builder.Build()
		assert.Contains(t, expr.SQL, "JSON_SET")
	})

	t.Run("Postgres SetBuilder", func(t *testing.T) {
		builder := NewSetBuilder("data", Postgres)
		builder.Path("key", "value")
		expr := builder.Build()
		assert.Contains(t, expr.SQL, "jsonb_set")
	})

	t.Run("SQLite SetBuilder", func(t *testing.T) {
		builder := NewSetBuilder("data", SQLite)
		builder.Path("$.key", "value")
		expr := builder.Build()
		assert.Contains(t, expr.SQL, "json_set")
	})
}

func TestJSONPathOps(t *testing.T) {
	SetDefaultDialect(MySQL)
	col := clause.Column{Name: "meta"}

	t.Run("Eq", func(t *testing.T) {
		ops := NewPathOps(col, "$.count", MySQL)
		expr := ops.Eq(100)
		sql, args, _ := expr.Build()

		assert.Equal(t, "JSON_EXTRACT(meta, ?) = ?", sql)
		assert.Equal(t, []any{"$.count", "100"}, args)
	})

	t.Run("Neq", func(t *testing.T) {
		ops := NewPathOps(col, "$.status", MySQL)
		expr := ops.Neq("draft")
		sql, args, _ := expr.Build()

		assert.Equal(t, "JSON_EXTRACT(meta, ?) != ?", sql)
		assert.Equal(t, []any{"$.status", `"draft"`}, args)
	})

	t.Run("Gt", func(t *testing.T) {
		ops := NewPathOps(col, "$.score", MySQL)
		expr := ops.Gt(50)
		sql, args, _ := expr.Build()

		assert.Equal(t, "JSON_EXTRACT(meta, ?) > ?", sql)
		assert.Equal(t, []any{"$.score", "50"}, args)
	})

	t.Run("Set", func(t *testing.T) {
		ops := NewPathOps(col, "$.count", MySQL)
		assign := ops.Set(200)
		sql, args, _ := assign.Build()

		assert.Equal(t, "meta = ?", sql)
		assert.Len(t, args, 1)
	})

	t.Run("Remove", func(t *testing.T) {
		ops := NewPathOps(col, "$.deprecated", MySQL)
		assign := ops.Remove()
		sql, args, _ := assign.Build()

		assert.Equal(t, "meta = ?", sql)
		assert.Len(t, args, 1)
	})
}
