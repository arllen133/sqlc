package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/arllen133/sqlc"
	"github.com/arllen133/sqlc/examples/02_relationships/models"
	"github.com/arllen133/sqlc/examples/02_relationships/models/generated"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	dsn := "file:test_rel.db?cache=shared&mode=rwc"

	// Setup DB
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, name TEXT);
	CREATE TABLE IF NOT EXISTS posts (id INTEGER PRIMARY KEY, user_id INTEGER, title TEXT);
	`); err != nil {
		log.Fatal(err)
	}

	sess := sqlc.NewSession(db, &sqlc.SQLiteDialect{})
	ctx := context.Background()

	userRepo := sqlc.NewRepository[models.User](sess)
	postRepo := sqlc.NewRepository[models.Post](sess)

	// Create User
	alice := &models.User{Name: "Alice"}
	userRepo.Create(ctx, alice)

	// Create Posts
	post1 := &models.Post{UserID: alice.ID, Title: "Post 1"}
	post2 := &models.Post{UserID: alice.ID, Title: "Post 2"}
	postRepo.Create(ctx, post1)
	postRepo.Create(ctx, post2)

	// Query User with Posts (Preload)
	fmt.Println("--- Preload Posts ---")
	u, err := userRepo.Query().
		Where(generated.User.ID.Eq(alice.ID)).
		WithPreload(sqlc.Preload(generated.User_Posts)).
		Take(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("User: %s\n", u.Name)
	for _, p := range u.Posts {
		fmt.Printf("  - Post: %s\n", p.Title)
	}

	// Query Post with Author
	fmt.Println("\n--- Preload Author ---")
	p, err := postRepo.Query().
		Where(generated.Post.ID.Eq(post1.ID)).
		WithPreload(sqlc.Preload(generated.Post_Author)).
		Take(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Post: %s, Author: %s\n", p.Title, p.Author.Name)

	os.Remove("test_rel.db")
}
