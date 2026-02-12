package sqlc_test

import (
	"testing"

	"github.com/arllen133/sqlc"
)

func TestSoftDeleteSQLGeneration(t *testing.T) {
	session := sqlc.NewSession(nil, &sqlc.SQLiteDialect{})
	productRepo := sqlc.NewRepository[SoftDeleteProduct](session)

	t.Run("DefaultQueryFilter", func(t *testing.T) {
		gotSQL, _, _ := productRepo.Query().ToSQL()
		want := "SELECT id, name, deleted_at FROM products WHERE deleted_at IS NULL"
		if !contains(gotSQL, want) {
			t.Errorf("got %s, want %s", gotSQL, want)
		}
	})

	t.Run("WithTrashedFilter", func(t *testing.T) {
		gotSQL, _, _ := productRepo.Query().WithTrashed().ToSQL()
		// WithTrashed should not have the IS NULL filter
		if contains(gotSQL, "WHERE deleted_at IS NULL") {
			t.Errorf("SQL should not contain soft delete filter: %s", gotSQL)
		}
	})

	t.Run("OnlyTrashedFilter", func(t *testing.T) {
		gotSQL, _, _ := productRepo.Query().OnlyTrashed().ToSQL()
		want := "SELECT id, name, deleted_at FROM products WHERE deleted_at IS NOT NULL"
		if !contains(gotSQL, want) {
			t.Errorf("got %s, want %s", gotSQL, want)
		}
	})
}
