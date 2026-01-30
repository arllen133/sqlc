package sqlc

import (
	"fmt"
	"reflect"

	"github.com/arllen133/sqlc/clause"
)

type PK = clause.Eq

// TableSchema defines how to map a model to a table and back
type Schema[T any] interface {
	// Table Metadata
	TableName() string

	// Read Operations
	SelectColumns() []string

	// Write Operations
	InsertRow(*T) ([]string, []any)

	// Update Operations
	UpdateMap(*T) map[string]any

	// Primary Key
	PK(*T) PK
	SetPK(m *T, val int64)
	AutoIncrement() bool
}

var schemas = make(map[reflect.Type]any)

func RegisterSchema[T any](schema Schema[T]) {
	var t T
	typ := reflect.TypeOf(t)
	schemas[typ] = schema
}

func LoadSchema[T any]() Schema[T] {
	var t T
	typ := reflect.TypeOf(t)
	if s, ok := schemas[typ]; ok {
		return s.(Schema[T])
	}
	panic(fmt.Sprintf("orm: schema not registered for type %v", typ))
}

// ScanRows was removed as part of sqlx refactor
