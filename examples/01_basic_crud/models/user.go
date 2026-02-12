package models

// User represents a simple user model
type User struct {
	ID    int64  `db:"id,primaryKey,autoIncrement"`
	Name  string `db:"name"`
	Email string `db:"email"`
	Age   int    `db:"age"`
}
