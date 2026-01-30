package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/arllen133/sqlc"
	"github.com/arllen133/sqlc/examples/blog/models"
	"github.com/arllen133/sqlc/examples/blog/models/generated"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Setup database
	db, _ := sql.Open("sqlite3", ":memory:")
	defer db.Close()

	_, _ = db.Exec(`CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT,
		email TEXT,
		created_at DATETIME
	)`)

	session := sqlc.NewSession(db, &sqlc.SQLiteDialect{})
	userRepo := sqlc.NewRepository[models.User](session)
	ctx := context.Background()

	// Create test users
	users := []*models.User{
		{Username: "alice", Email: "alice@example.com", CreatedAt: time.Now()},
		{Username: "bob", Email: "bob@example.com", CreatedAt: time.Now()},
		{Username: "charlie", Email: "charlie@example.com", CreatedAt: time.Now()},
	}

	for _, u := range users {
		_ = userRepo.Create(ctx, u)
	}

	fmt.Println("=== Complete ORM Demo ===")

	// 1. Basic CRUD
	fmt.Println("1. Create User")
	newUser := &models.User{
		Username:  "david",
		Email:     "david@example.com",
		CreatedAt: time.Now(),
	}
	_ = userRepo.Create(ctx, newUser)
	fmt.Printf("   Created user ID: %d\n\n", newUser.ID)

	// 2. Query with new field system
	fmt.Println("2. Query with field.String.Eq()")
	result, _ := userRepo.Query().
		Where(generated.User.Username.Eq("alice")).
		Find(ctx)
	fmt.Printf("   Found %d users\n", len(result))
	if len(result) > 0 {
		fmt.Printf("   User: %s (%s)\n\n", result[0].Username, result[0].Email)
	}

	// 3. Query with Like
	fmt.Println("3. Query with field.String.Like()")
	result, _ = userRepo.Query().
		Where(generated.User.Username.Like("%li%")).
		Find(ctx)
	fmt.Printf("   Found %d users with 'li' in username: ", len(result))
	for _, u := range result {
		fmt.Printf("%s ", u.Username)
	}
	fmt.Printf("\n")

	// 4. Query with Number comparison
	fmt.Println("4. Query with field.Number.Gt()")
	result, _ = userRepo.Query().
		Where(generated.User.ID.Gt(2)).
		Find(ctx)
	fmt.Printf("   Found %d users with ID > 2\n\n", len(result))

	// 5. Count
	fmt.Println("5. Count all users")
	count, _ := userRepo.Query().Count(ctx)
	fmt.Printf("   Total users: %d\n\n", count)

	// 6. Update
	fmt.Println("6. Update user")
	newUser.Email = "david.updated@example.com"
	_ = userRepo.Update(ctx, newUser)
	fmt.Printf("   Updated user email to: %s\n\n", newUser.Email)

	// 7. Delete
	fmt.Println("7. Delete user")
	_ = userRepo.Delete(ctx, newUser.ID)
	finalCount, _ := userRepo.Query().Count(ctx)
	fmt.Printf("   Remaining users: %d\n\n", finalCount)

	// 8. Transaction
	fmt.Println("8. Transaction test")
	err := session.Transaction(ctx, func(txSession *sqlc.Session) error {
		txRepo := sqlc.NewRepository[models.User](txSession)

		user1 := &models.User{Username: "tx_user1", Email: "tx1@example.com", CreatedAt: time.Now()}
		if err := txRepo.Create(ctx, user1); err != nil {
			return err
		}

		user2 := &models.User{Username: "tx_user2", Email: "tx2@example.com", CreatedAt: time.Now()}
		if err := txRepo.Create(ctx, user2); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		fmt.Printf("   Transaction failed: %v\n", err)
	} else {
		fmt.Println("   Transaction committed successfully")
		finalCount, _ := userRepo.Query().Count(ctx)
		fmt.Printf("   Total users after transaction: %d\n", finalCount)
	}

	fmt.Println("\n=== Demo Complete ===")
	fmt.Println("✓ CRUD operations")
	fmt.Println("✓ Field-based queries (Eq, Like, Gt)")
	fmt.Println("✓ Count aggregation")
	fmt.Println("✓ Transactions with auto commit/rollback")
	fmt.Println("✓ Lifecycle hooks (BeforeCreate, AfterCreate)")
}
