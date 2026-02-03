package sqlc

import (
	"reflect"
	"strings"

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

// getFieldValue extracts a field value from a struct by column name.
// It matches against db tags or field names.
func getFieldValue(v any, columnName string) any {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil
	}

	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// Check db tag
		if dbTag := field.Tag.Get("db"); dbTag != "" {
			// Parse db tag (format: "column_name,options...")
			parts := strings.Split(dbTag, ",")
			if parts[0] == columnName {
				return val.Field(i).Interface()
			}
		}

		// Fallback: match field name (case-insensitive)
		if strings.EqualFold(field.Name, columnName) {
			return val.Field(i).Interface()
		}
	}
	return nil
}
