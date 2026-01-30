package benchmarks

import (
	"testing"

	"github.com/arllen133/sqlc"
	"github.com/arllen133/sqlc/clause"
	"github.com/arllen133/sqlc/field"
)

func BenchmarkResolveColumnNames_Field(b *testing.B) {
	f1 := field.String{}.WithColumn("id")
	f2 := field.String{}.WithColumn("name")
	f3 := field.String{}.WithColumn("email")
	args := []clause.Columnar{f1, f2, f3}
	for b.Loop() {
		_ = sqlc.ResolveColumnNames(args)
	}
}

func BenchmarkResolveColumnNames_ClauseColumn(b *testing.B) {
	c1 := clause.Column{Name: "id"}
	c2 := clause.Column{Name: "name"}
	c3 := clause.Column{Name: "email"}
	args := []clause.Columnar{c1, c2, c3}
	for b.Loop() {
		_ = sqlc.ResolveColumnNames(args)
	}
}

func BenchmarkResolveColumnNames_Mixed(b *testing.B) {
	f1 := field.String{}.WithColumn("name")
	args := []clause.Columnar{clause.Column{Name: "id"}, f1, clause.Column{Name: "email"}}
	for b.Loop() {
		_ = sqlc.ResolveColumnNames(args)
	}
}
