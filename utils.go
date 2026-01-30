package sqlc

import (
	"github.com/arllen133/sqlc/clause"
)

// ResolveColumnNames extracts column names from types implementing clause.Columnar.
func ResolveColumnNames(args []clause.Columnar) []string {
	if len(args) == 0 {
		return nil
	}
	cols := make([]string, len(args))
	for i, arg := range args {
		cols[i] = arg.ColumnName()
	}
	return cols
}
