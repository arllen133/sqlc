package models

import "time"

// Product with Soft Delete
type Product struct {
	ID        int64      `db:"id,primaryKey,autoIncrement"`
	Name      string     `db:"name"`
	DeletedAt *time.Time `db:"deleted_at"`
}
