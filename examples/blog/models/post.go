package models

import "time"

type Post struct {
	ID        int64     `db:"id,primaryKey,autoIncrement"`
	UserID    int64     `db:"user_id"`
	Title     string    `db:"title,size:200"`
	Content   string    `db:"content"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
