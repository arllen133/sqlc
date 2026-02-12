package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/arllen133/sqlc"
	"github.com/arllen133/sqlc/examples/03_soft_delete/models"
	"github.com/arllen133/sqlc/examples/03_soft_delete/models/generated"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	dsn := "file:test_sd.db?cache=shared&mode=rwc"

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS products (id INTEGER PRIMARY KEY, name TEXT, deleted_at TIMESTAMP);`); err != nil {
		log.Fatal(err)
	}

	sess := sqlc.NewSession(db, &sqlc.SQLiteDialect{})
	ctx := context.Background()

	repo := sqlc.NewRepository[models.Product](sess)

	// Create
	p := &models.Product{Name: "Laptop"}
	repo.Create(ctx, p)
	fmt.Printf("Created: %s (ID: %d)\n", p.Name, p.ID)

	// Soft Delete
	fmt.Println("--- Soft Deleting ---")
	if err := repo.Delete(ctx, p.ID); err != nil {
		log.Fatal(err)
	}

	// Try to find (should not find)
	_, err = repo.FindOne(ctx, p.ID)
	if err == sqlc.ErrNotFound {
		fmt.Println("Product not found (as expected)")
	} else {
		fmt.Println("Error:", err)
	}

	// Find with Trash
	fmt.Println("--- Find with Trash ---")
	pTrashed, err := repo.Query().WithTrashed().Where(generated.Product.ID.Eq(p.ID)).Take(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found in trash: %s (DeletedAt: %v)\n", pTrashed.Name, pTrashed.DeletedAt)

	// Restore
	fmt.Println("--- Restoring ---")
	repo.Restore(ctx, p.ID)

	pRestored, err := repo.FindOne(ctx, p.ID)
	if err == nil {
		fmt.Printf("Restored: %s\n", pRestored.Name)
	}

	// Hard Delete
	fmt.Println("--- Hard Deleting (Unscoped) ---")
	if err := repo.Unscoped().Delete(ctx, p.ID); err != nil {
		log.Fatal(err)
	}

	// Verify completely gone
	_, err = repo.Query().WithTrashed().Where(generated.Product.ID.Eq(p.ID)).Take(ctx)
	if err == sqlc.ErrNotFound {
		fmt.Println("Product completely removed (as expected)")
	}

	os.Remove("test_sd.db")
}
