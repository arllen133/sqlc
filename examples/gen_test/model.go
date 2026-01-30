package gentest

import (
	"encoding/json"
	"time"
)

// User represents a user in the system.
// It has various field types to test the generator.
type User struct {
	ID        int64           `db:"column:id;primaryKey;autoIncrement"`
	Name      string          `db:"column:name"`     // Name of the user
	Avatar    []byte          `db:"column:avatar"`   // User avatar image
	Settings  json.RawMessage `db:"column:settings"` // User settings JSON
	CreatedAt time.Time       `db:"column:created_at"`
}
