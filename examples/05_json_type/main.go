package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/arllen133/sqlc"
	"github.com/arllen133/sqlc/examples/05_json_type/models"
	"github.com/arllen133/sqlc/examples/05_json_type/models/generated"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	dsn := "file:test_json.db?cache=shared&mode=rwc"

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// SQLite stores JSON as TEXT
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS user_configs (id INTEGER PRIMARY KEY, username TEXT, settings TEXT);`); err != nil {
		log.Fatal(err)
	}

	sess := sqlc.NewSession(db, &sqlc.SQLiteDialect{})
	ctx := context.Background()

	repo := sqlc.NewRepository[models.UserConfig](sess)

	// Create
	fmt.Println("--- Create with JSON ---")
	cfg := &models.UserConfig{
		Username: "bob",
		Settings: sqlc.JSON[models.Settings]{
			Data: models.Settings{Theme: "dark", Notifications: true},
		},
	}
	if err := repo.Create(ctx, cfg); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created: %s, Settings: %+v\n", cfg.Username, cfg.Settings.Data)

	// Query by JSON path (SQLite supports json_extract)
	// sqlc generic query builder might support raw expressions or specific JSON ops if implemented
	// For now, we can read it back.

	fmt.Println("--- Read Back ---")
	fetched, err := repo.FindOne(ctx, cfg.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Fetched Settings: Theme=%s\n", fetched.Settings.Data.Theme)

	// JSON Query Example (if supported by dialect/builder)
	fmt.Println("--- JSON Path Query ---")
	// Using generated helper for JSON path
	// Assuming generated code provides something like generated.UserConfig.Settings.Theme

	users, err := repo.Query().
		Where(generated.Settings.Theme.Eq("dark")).
		Find(ctx)
	if err != nil {
		log.Printf("Query failed (might need dialect support): %v\n", err)
	} else {
		fmt.Printf("Found %d users with dark theme\n", len(users))
	}

	os.Remove("test_json.db")
}
