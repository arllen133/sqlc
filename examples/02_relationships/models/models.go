package models

// User has many Posts
type User struct {
	ID    int64   `db:"id,primaryKey,autoIncrement"`
	Name  string  `db:"name"`
	Posts []*Post `db:"-" relation:"hasMany,foreignKey:user_id"`
}

// Post belongs to User
type Post struct {
	ID     int64  `db:"id,primaryKey,autoIncrement"`
	UserID int64  `db:"user_id"`
	Title  string `db:"title"`
	Author *User  `db:"-" relation:"belongsTo,foreignKey:user_id"`
}
