package type_alias

// Custom type aliases
type Status int
type UserID int64
type Email string

// User demonstrates type alias resolution
type User struct {
	ID        UserID `db:"id,primaryKey,autoIncrement"`
	Email     Email  `db:"email"`
	Status    Status `db:"status"`
	CreatedAt string `db:"created_at"`
}
