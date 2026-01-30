package json

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMySQLDialect(t *testing.T) {
	d := MySQL

	t.Run("ExtractPath", func(t *testing.T) {
		sql, vars := d.ExtractPath("meta", "$.tags")
		assert.Equal(t, "JSON_EXTRACT(meta, ?)", sql)
		assert.Equal(t, []any{"$.tags"}, vars)
	})

	t.Run("PathEq", func(t *testing.T) {
		sql, vars := d.PathEq("meta", "$.count", 10)
		assert.Equal(t, "JSON_EXTRACT(meta, ?) = ?", sql)
		assert.Equal(t, []any{"$.count", "10"}, vars)
	})

	t.Run("Contains", func(t *testing.T) {
		sql, vars := d.Contains("meta", "foo", "$.tags")
		assert.Equal(t, "JSON_CONTAINS(meta, ?, ?)", sql)
		assert.Equal(t, []any{`"foo"`, "$.tags"}, vars)
	})

	t.Run("SetPath", func(t *testing.T) {
		expr := d.SetPath("meta", "$.count", 20)
		assert.Equal(t, "JSON_SET(meta, ?, ?)", expr.SQL)
		assert.Equal(t, []any{"$.count", "20"}, expr.Vars)
	})

	t.Run("MergePatch", func(t *testing.T) {
		expr := d.MergePatch("meta", map[string]int{"a": 1})
		assert.Equal(t, "JSON_MERGE_PATCH(meta, ?)", expr.SQL)
		assert.Equal(t, []any{`{"a":1}`}, expr.Vars)
	})

	t.Run("MergePreserve", func(t *testing.T) {
		expr := d.MergePreserve("meta", map[string]int{"a": 1})
		assert.Equal(t, "JSON_MERGE_PRESERVE(meta, ?)", expr.SQL)
		assert.Equal(t, []any{`{"a":1}`}, expr.Vars)
	})
}

func TestPostgresDialect(t *testing.T) {
	d := Postgres

	t.Run("ExtractPath", func(t *testing.T) {
		sql, vars := d.ExtractPath("meta", "view_count")
		// Postgres formatPath adds curly braces logic
		// If input is "view_count" (simple key)
		// ExtractPath uses ->>
		assert.Equal(t, "meta->>'view_count'", sql)
		assert.Nil(t, vars)
	})

	t.Run("PathEq", func(t *testing.T) {
		// Postgres #> '{view_count}'
		sql, vars := d.PathEq("meta", "view_count", 10)
		assert.Equal(t, "meta #> '{view_count}' = ?::jsonb", sql)
		assert.Equal(t, []any{"10"}, vars)
	})

	t.Run("Contains", func(t *testing.T) {
		sql, vars := d.Contains("meta", "foo", "tags")
		// Postgres @>
		// If path provided: meta->'tags' @> ?
		assert.Equal(t, "meta->'tags' @> ?::jsonb", sql)
		assert.Equal(t, []any{`"foo"`}, vars)
	})

	t.Run("SetPath", func(t *testing.T) {
		expr := d.SetPath("meta", "view_count", 20)
		assert.Equal(t, "jsonb_set(meta, '{view_count}', ?::jsonb)", expr.SQL)
		assert.Equal(t, []any{"20"}, expr.Vars)
	})

	t.Run("MergePatch", func(t *testing.T) {
		expr := d.MergePatch("meta", map[string]int{"a": 1})
		assert.Equal(t, "meta || ?::jsonb", expr.SQL)
		assert.Equal(t, []any{`{"a":1}`}, expr.Vars)
	})
}

func TestSQLiteDialect(t *testing.T) {
	d := SQLite

	t.Run("ExtractPath", func(t *testing.T) {
		sql, vars := d.ExtractPath("meta", "$.tags")
		assert.Equal(t, "json_extract(meta, ?)", sql)
		assert.Equal(t, []any{"$.tags"}, vars)
	})

	t.Run("SetPath", func(t *testing.T) {
		expr := d.SetPath("meta", "$.count", 20)
		assert.Equal(t, "json_set(meta, ?, ?)", expr.SQL)
		assert.Equal(t, []any{"$.count", "20"}, expr.Vars)
	})

	t.Run("MergePatch", func(t *testing.T) {
		expr := d.MergePatch("meta", map[string]int{"a": 1})
		assert.Equal(t, "json_patch(meta, ?)", expr.SQL)
		assert.Equal(t, []any{`{"a":1}`}, expr.Vars)
	})

	t.Run("MergePreserve", func(t *testing.T) {
		expr := d.MergePreserve("meta", map[string]int{"a": 1})
		// Fallback to json_patch
		assert.Equal(t, "json_patch(meta, ?)", expr.SQL)
		assert.Equal(t, []any{`{"a":1}`}, expr.Vars)
	})
}
