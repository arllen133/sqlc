package models

import (
	"context"
	"fmt"
	"time"
)

type Task struct {
	ID        int64     `db:"id,primaryKey,autoIncrement"`
	Title     string    `db:"title"`
	CreatedAt time.Time `db:"created_at"`
	Status    string    `db:"status"`
}

// BeforeCreate hook
func (t *Task) BeforeCreate(ctx context.Context) error {
	fmt.Println("[Hook] BeforeCreate: Setting default status and time")
	if t.Status == "" {
		t.Status = "pending"
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now()
	}
	return nil
}

// AfterCreate hook
func (t *Task) AfterCreate(ctx context.Context) error {
	fmt.Printf("[Hook] AfterCreate: Task '%s' created with ID %d\n", t.Title, t.ID)
	return nil
}
