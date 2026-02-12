package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/arllen133/sqlc"
	"github.com/arllen133/sqlc/examples/06_hooks/models"
	_ "github.com/arllen133/sqlc/examples/06_hooks/models/generated"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	dsn := "file:test_hooks.db?cache=shared&mode=rwc"

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS tasks (id INTEGER PRIMARY KEY, title TEXT, created_at TIMESTAMP, status TEXT);`); err != nil {
		log.Fatal(err)
	}

	sess := sqlc.NewSession(db, &sqlc.SQLiteDialect{})
	ctx := context.Background()

	repo := sqlc.NewRepository[models.Task](sess)

	fmt.Println("--- Creating Task ---")
	task := &models.Task{Title: "Do Laundry"}
	// Hooks should trigger
	if err := repo.Create(ctx, task); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Task Status: %s\n", task.Status)       // Should be "pending"
	fmt.Printf("Task CreatedAt: %v\n", task.CreatedAt) // Should be set

	os.Remove("test_hooks.db")
}
