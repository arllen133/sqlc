package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/arllen133/sqlc"
	"github.com/arllen133/sqlc/examples/01_basic_crud/models"
	"github.com/arllen133/sqlc/examples/01_basic_crud/models/generated"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// 1. Initialize DB
	dsn := "file:test_basic.db?cache=shared&mode=rwc"

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		email TEXT,
		age INTEGER
	);`

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(createTableSQL); err != nil {
		log.Fatal(err)
	}

	// 2. Initialize Session
	sess := sqlc.NewSession(db, &sqlc.SQLiteDialect{})

	ctx := context.Background()
	repo := sqlc.NewRepository[models.User](sess)

	// 3. Create
	fmt.Println("--- Create ---")
	user := &models.User{
		Name:  "Alice",
		Email: "alice@example.com",
		Age:   30,
	}
	if err := repo.Create(ctx, user); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created User: ID=%d, Name=%s\n", user.ID, user.Name)

	// 4. Read (FindOne)
	fmt.Println("\n--- FindOne ---")
	fetchedUser, err := repo.FindOne(ctx, user.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Fetched: %v\n", fetchedUser)

	// 5. Update
	fmt.Println("\n--- Update ---")
	fetchedUser.Age = 31
	if err := repo.Update(ctx, fetchedUser); err != nil {
		log.Fatal(err)
	}

	// Verify update
	updatedUser, _ := repo.FindOne(ctx, user.ID)
	fmt.Printf("Updated Age: %d\n", updatedUser.Age)

	// 6. Query with Conditions
	fmt.Println("\n--- Query ---")
	users, err := repo.Query().
		Where(generated.User.Age.Gt(20)).
		Where(generated.User.Name.Eq("Alice")).
		Find(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found %d users > 20\n", len(users))

	// 7. Delete
	fmt.Println("\n--- Delete ---")
	if err := repo.Delete(ctx, user.ID); err != nil {
		log.Fatal(err)
	}
	fmt.Println("User deleted")

	// Verify delete
	_, err = repo.FindOne(ctx, user.ID)
	if err == sqlc.ErrNotFound {
		fmt.Println("User not found as expected")
	}

	// Clean up
	os.Remove("test_basic.db")
}
