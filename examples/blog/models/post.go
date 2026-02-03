package models

import (
	"time"

	"github.com/arllen133/sqlc"
)

// PostMetadata represents the JSON metadata for a post
type PostMetadata struct {
	ViewCount    int      `json:"view_count"`
	LikeCount    int      `json:"like_count"`
	Tags         []string `json:"tags"`
	IsFeatured   bool     `json:"is_featured"`
	CategoryName string   `json:"category_name"`
}

type Post struct {
	ID        int64                   `db:"id,primaryKey,autoIncrement"`
	UserID    int64                   `db:"user_id"`
	Title     string                  `db:"title,size:200"`
	Content   string                  `db:"content"`
	Metadata  sqlc.JSON[PostMetadata] `db:"metadata,type:json"` // JSON field with type:json tag
	CreatedAt time.Time               `db:"created_at"`
	UpdatedAt time.Time               `db:"updated_at"`

	// Relation: Post has one Author (User)
	Author *User `db:"-" relation:"belongsTo,foreignKey:user_id"`
}
