package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/arllen133/sqlc"
	"github.com/arllen133/sqlc/clause"
	"github.com/arllen133/sqlc/field"
	_ "github.com/mattn/go-sqlite3"
)

// User model
type User struct {
	ID        int64     `db:"id,primaryKey,autoIncrement"`
	Username  string    `db:"username"`
	Email     string    `db:"email"`
	Age       int       `db:"age"`
	Status    string    `db:"status"`
	CreatedAt time.Time `db:"created_at"`
}

// TableName implements the TableNamer interface
func (User) TableName() string {
	return "users"
}

// Define field struct for User
var UserFields = struct {
	ID        field.Number[int64]
	Username  field.String
	Email     field.String
	Age       field.Number[int]
	Status    field.String
	CreatedAt field.Time
}{
	ID:        field.Number[int64]{}.WithColumn("id"),
	Username:  field.String{}.WithColumn("username"),
	Email:     field.String{}.WithColumn("email"),
	Age:       field.Number[int]{}.WithColumn("age"),
	Status:    field.String{}.WithColumn("status"),
	CreatedAt: field.Time{}.WithColumn("created_at"),
}

// Implement UserSchema
type UserSchema struct{}

func (UserSchema) TableName() string { return "users" }

func (UserSchema) PrimaryKey() string { return "id" }

func (UserSchema) AutoIncrement() bool { return true }

func (UserSchema) SelectColumns() []string {
	return []string{"id", "username", "email", "age", "status", "created_at"}
}

func (UserSchema) InsertRow(u *User) ([]string, []any) {
	var cols []string
	var vals []any
	if u.ID != 0 {
		cols = append(cols, "id")
		vals = append(vals, u.ID)
	}
	cols = append(cols, "username")
	vals = append(vals, u.Username)
	cols = append(cols, "email")
	vals = append(vals, u.Email)
	cols = append(cols, "age")
	vals = append(vals, u.Age)
	cols = append(cols, "status")
	vals = append(vals, u.Status)
	cols = append(cols, "created_at")
	vals = append(vals, u.CreatedAt)
	return cols, vals
}

func (UserSchema) UpdateMap(u *User) map[string]any {
	return map[string]any{
		"username":   u.Username,
		"email":      u.Email,
		"age":        u.Age,
		"status":     u.Status,
		"created_at": u.CreatedAt,
	}
}

func (UserSchema) PK(u *User) sqlc.PK {
	var val any
	if u != nil {
		val = u.ID
	}
	return sqlc.PK{
		Column: clause.Column{Name: "id"},
		Value:  val,
	}
}

func (UserSchema) SetPK(u *User, val int64) {
	u.ID = val
}

func init() {
	sqlc.RegisterSchema(UserSchema{})
}

func main() {
	// Create database connection
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create table
	createTable := `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL,
			email TEXT NOT NULL,
			age INTEGER NOT NULL,
			status TEXT NOT NULL,
			created_at DATETIME NOT NULL
		)
	`
	if _, err := db.Exec(createTable); err != nil {
		log.Fatal(err)
	}

	// Create session and repository
	session := sqlc.NewSession(db, &sqlc.SQLiteDialect{})
	userRepo := sqlc.NewRepository[User](session)
	ctx := context.Background()

	fmt.Println("=== Expression System Demo ===")
	fmt.Println()

	// Insert test data
	users := []*User{
		{Username: "alice", Email: "alice@example.com", Age: 25, Status: "active", CreatedAt: time.Now()},
		{Username: "bob", Email: "bob@example.com", Age: 30, Status: "active", CreatedAt: time.Now()},
		{Username: "charlie", Email: "charlie@example.com", Age: 17, Status: "inactive", CreatedAt: time.Now()},
		{Username: "david", Email: "david@example.com", Age: 35, Status: "active", CreatedAt: time.Now()},
		{Username: "eve", Email: "eve@example.com", Age: 22, Status: "suspended", CreatedAt: time.Now()},
	}

	for _, user := range users {
		if err := userRepo.Create(ctx, user); err != nil {
			log.Fatal(err)
		}
	}
	fmt.Println("✓ Created 5 test users")
	fmt.Println()

	// Demo 1: Simple equality
	fmt.Println("1. Find user with username = 'alice':")
	results, _ := userRepo.Query().
		Where(UserFields.Username.Eq("alice")).
		Find(ctx)
	for _, u := range results {
		fmt.Printf("   → %s (age: %d, status: %s)\n", u.Username, u.Age, u.Status)
	}
	fmt.Println()

	// Demo 2: Greater than
	fmt.Println("2. Find users with age > 25:")
	results, _ = userRepo.Query().
		Where(UserFields.Age.Gt(25)).
		Find(ctx)
	for _, u := range results {
		fmt.Printf("   → %s (age: %d)\n", u.Username, u.Age)
	}
	fmt.Println()

	// Demo 3: IN clause
	fmt.Println("3. Find users with username IN ('alice', 'bob', 'charlie'):")
	results, _ = userRepo.Query().
		Where(UserFields.Username.In("alice", "bob", "charlie")).
		Find(ctx)
	for _, u := range results {
		fmt.Printf("   → %s\n", u.Username)
	}
	fmt.Println()

	// Demo 4: BETWEEN
	fmt.Println("4. Find users with age BETWEEN 20 AND 30:")
	results, _ = userRepo.Query().
		Where(UserFields.Age.Between(20, 30)).
		Find(ctx)
	for _, u := range results {
		fmt.Printf("   → %s (age: %d)\n", u.Username, u.Age)
	}
	fmt.Println()

	// Demo 5: LIKE
	fmt.Println("5. Find users with email ending in 'example.com':")
	results, _ = userRepo.Query().
		Where(UserFields.Email.Like("%example.com")).
		Find(ctx)
	for _, u := range results {
		fmt.Printf("   → %s (%s)\n", u.Username, u.Email)
	}
	fmt.Println()

	// Demo 6: AND logic
	fmt.Println("6. Find users with age > 18 AND status = 'active':")
	results, _ = userRepo.Query().
		Where(clause.And{
			UserFields.Age.Gt(18),
			UserFields.Status.Eq("active"),
		}).
		Find(ctx)
	for _, u := range results {
		fmt.Printf("   → %s (age: %d, status: %s)\n", u.Username, u.Age, u.Status)
	}
	fmt.Println()

	// Demo 7: OR logic
	fmt.Println("7. Find users with status = 'inactive' OR status = 'suspended':")
	results, _ = userRepo.Query().
		Where(clause.Or{
			UserFields.Status.Eq("inactive"),
			UserFields.Status.Eq("suspended"),
		}).
		Find(ctx)
	for _, u := range results {
		fmt.Printf("   → %s (status: %s)\n", u.Username, u.Status)
	}
	fmt.Println()

	// Demo 8: Complex nested logic
	fmt.Println("8. Complex: (age > 18 AND status = 'active') OR username = 'charlie':")
	results, _ = userRepo.Query().
		Where(clause.Or{
			clause.And{
				UserFields.Age.Gt(18),
				UserFields.Status.Eq("active"),
			},
			UserFields.Username.Eq("charlie"),
		}).
		Find(ctx)
	for _, u := range results {
		fmt.Printf("   → %s (age: %d, status: %s)\n", u.Username, u.Age, u.Status)
	}
	fmt.Println()

	// Demo 9: NOT logic
	fmt.Println("9. Find users NOT with status = 'active':")
	results, _ = userRepo.Query().
		Where(clause.Not{
			Expr: UserFields.Status.Eq("active"),
		}).
		Find(ctx)
	for _, u := range results {
		fmt.Printf("   → %s (status: %s)\n", u.Username, u.Status)
	}
	fmt.Println()

	// Demo 10: Count with expression
	fmt.Println("10. Count users with age >= 25:")
	count, _ := userRepo.Query().
		Where(UserFields.Age.Gte(25)).
		Count(ctx)
	fmt.Printf("   → %d users\n", count)
	fmt.Println()

	fmt.Println("=== All Expression Features Working! ===")
}
