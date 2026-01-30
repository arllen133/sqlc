package main

import (
	"context"
	"database/sql"
	"fmt"

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
		{Username: "alice", Email: "alice@example.com"},
		{Username: "bob", Email: "bob@example.com"},
		{Username: "charlie", Email: "charlie@example.com"},
	}

	for _, u := range users {
		_ = userRepo.Create(ctx, u)
	}

	fmt.Println("=== New Field System Demo ===")

	// Example 1: Using field.Number with type-specific operations
	fmt.Println("1. Number field - ID > 1")
	fmt.Printf("   Field type: %T\n", generated.User.ID)
	fmt.Printf("   Column: %+v\n\n", generated.User.ID.Column())

	// Example 2: Using field.String with string operations
	fmt.Println("2. String field - Username LIKE '%li%'")
	fmt.Printf("   Field type: %T\n", generated.User.Username)
	fmt.Printf("   Column: %+v\n\n", generated.User.Username)

	// Example 3: WithColumn and WithTable methods
	fmt.Println("3. Field configuration methods")
	userID := generated.User.ID.WithTable("users")
	fmt.Printf("   ID with table: %+v\n", userID.Column())

	username := generated.User.Username.WithColumn("user_name")
	fmt.Printf("   Username with new column: %+v\n\n", username.Column())

	// Example 4: Expression building
	fmt.Println("4. Building expressions")
	eqExpr := generated.User.Username.Eq("alice")
	fmt.Printf("   Eq expression type: %T\n", eqExpr)

	likeExpr := generated.User.Username.Like("%li%")
	fmt.Printf("   Like expression type: %T\n", likeExpr)

	gtExpr := generated.User.ID.Gt(1)
	fmt.Printf("   Gt expression type: %T\n", gtExpr)

	betweenExpr := generated.User.ID.Between(1, 3)
	fmt.Printf("   Between expression type: %T\n\n", betweenExpr)

	fmt.Println("=== Field System Architecture ===")
	fmt.Println("✓ Fields are independent of models")
	fmt.Println("✓ Type-specific operations (Number, String, Time, Bool)")
	fmt.Println("✓ Fluent API with WithColumn() and WithTable()")
	fmt.Println("✓ Rich expression system via clause package")
	fmt.Println("✓ Ready for advanced query building")
}
