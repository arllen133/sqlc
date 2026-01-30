package models

import (
	"context"
	"time"
)

type User struct {
	ID        int64     `db:"id,primaryKey,autoIncrement"`
	Username  string    `db:"username,size:100,unique"`
	Email     string    `db:"email,size:255,index"`
	CreatedAt time.Time `db:"created_at"`
}

func (User) TableName() string {
	return "users"
}

func (u *User) BeforeCreate(ctx context.Context) error {
	if u.CreatedAt.IsZero() {
		u.CreatedAt = time.Now()
	}
	return nil
}

func (u *User) AfterCreate(ctx context.Context) error {
	return nil
}

// ...
type Category struct {
	ID   string `db:"id,primaryKey,size:36"`
	Name string `db:"name"`
}

func (Category) TableName() string {
	return "categories"
}
