package field_test

import (
	"testing"

	"github.com/arllen133/sqlc/field"
	"github.com/arllen133/sqlc/field/json"
	"github.com/stretchr/testify/assert"
)

// PostMeta is a test metadata type
type PostMeta struct {
	ViewCount int      `json:"view_count"`
	Tags      []string `json:"tags"`
}

func TestJSONField(t *testing.T) {
	// Set default dialect for tests
	json.SetDefaultDialect(json.MySQL)

	// Create JSON field
	meta := field.JSON[PostMeta]{}.WithColumn("metadata")

	t.Run("ColumnName", func(t *testing.T) {
		assert.Equal(t, "metadata", meta.ColumnName())
	})

	t.Run("Column", func(t *testing.T) {
		col := meta.Column()
		assert.Equal(t, "metadata", col.Name)
	})

	t.Run("Set assignment", func(t *testing.T) {
		assign := meta.Set(PostMeta{ViewCount: 100, Tags: []string{"go"}})
		sql, args := assign.Build()
		assert.Equal(t, "metadata = ?", sql)
		assert.Len(t, args, 1)
	})
}

func TestJSONFieldWithTable(t *testing.T) {
	json.SetDefaultDialect(json.MySQL)

	meta := field.JSON[PostMeta]{}.WithTable("posts").WithColumn("metadata")

	t.Run("ColumnName includes table", func(t *testing.T) {
		// When table is set, ColumnName returns "table.column"
		assert.Equal(t, "posts.metadata", meta.ColumnName())
	})

	t.Run("Column has table", func(t *testing.T) {
		col := meta.Column()
		assert.Equal(t, "posts", col.Table)
		assert.Equal(t, "metadata", col.Name)
	})
}
