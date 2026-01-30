package sqlc

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// JSON is a generic wrapper for handling JSON fields in database.
// It implements sql.Scanner and driver.Valuer.
//
// Usage:
//
//	type User struct {
//	    Metadata sqlc.JSON[UserMetadata] `db:"metadata"`
//	}
//
//	// Access data
//	user.Metadata.Data.Initial = "V"
type JSON[T any] struct {
	Data T
}

// NewJSON creates a new JSON wrapper for the given value.
func NewJSON[T any](v T) JSON[T] {
	return JSON[T]{Data: v}
}

// Scan implements the sql.Scanner interface.
func (j *JSON[T]) Scan(value any) error {
	if value == nil {
		var zero T
		j.Data = zero
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("sqlc: failed to scan JSON: expected []byte or string, got %T", value)
	}

	if len(bytes) == 0 {
		var zero T
		j.Data = zero
		return nil
	}

	return json.Unmarshal(bytes, &j.Data)
}

// Value implements the driver.Valuer interface.
func (j JSON[T]) Value() (driver.Value, error) {
	return json.Marshal(j.Data)
}

// MarshalJSON implements json.Marshaler.
func (j JSON[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(j.Data)
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *JSON[T]) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &j.Data)
}
