package sqlc_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"os"

	"github.com/arllen133/sqlc"
	"github.com/arllen133/sqlc/clause"
	"github.com/arllen133/sqlc/examples/blog/models"
	"github.com/arllen133/sqlc/examples/blog/models/generated"
	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) (*sql.DB, *sqlc.Session) {
	driver := os.Getenv("TEST_DRIVER")
	dsn := os.Getenv("TEST_DSN")

	if driver == "" {
		driver = "sqlite3"
		dsn = ":memory:"
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	var dialect sqlc.Dialect
	switch driver {
	case "mysql":
		dialect = &sqlc.MySQLDialect{}
	case "postgres":
		dialect = &sqlc.PostgreSQLDialect{}
	default:
		dialect = &sqlc.SQLiteDialect{}
	}

	// Create table if not exists (Basic schema for simple tests)
	switch driver {
	case "sqlite3":
		_, err = db.Exec(`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT,
			email TEXT,
			created_at DATETIME
		)`)
	case "mysql":
		_, err = db.Exec(`CREATE TABLE IF NOT EXISTS users (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			username VARCHAR(255),
			email VARCHAR(255),
			created_at DATETIME
		)`)
		// Truncate to ensure clean state for MySQL/PG which persist
		db.Exec("TRUNCATE TABLE users")
	case "postgres":
		_, err = db.Exec(`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username TEXT,
			email TEXT,
			created_at TIMESTAMP
		)`)
		db.Exec("TRUNCATE TABLE users RESTART IDENTITY")
	}

	if err != nil {
		t.Fatalf("Failed to create/init table: %v", err)
	}

	session := sqlc.NewSession(db, dialect)
	return db, session
}

func TestCRUDOperations(t *testing.T) {
	db, session := setupTestDB(t)
	defer db.Close()

	userRepo := sqlc.NewRepository[models.User](session)
	ctx := context.Background()

	t.Run("Create", func(t *testing.T) {
		user := &models.User{
			Username:  "alice",
			Email:     "alice@example.com",
			CreatedAt: time.Now(),
		}

		err := userRepo.Create(ctx, user)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		if user.ID == 0 {
			t.Error("Expected user ID to be set after create")
		}
	})

	t.Run("FindOne", func(t *testing.T) {
		user, err := userRepo.FindOne(ctx, 1)
		if err != nil {
			t.Fatalf("FindOne failed: %v", err)
		}

		if user.Username != "alice" {
			t.Errorf("Expected username 'alice', got '%s'", user.Username)
		}
	})

	t.Run("Update", func(t *testing.T) {
		user, err := userRepo.FindOne(ctx, 1)
		if err != nil {
			t.Fatalf("FindOne for Update failed: %v", err)
		}
		user.Email = "alice.updated@example.com"

		err = userRepo.Update(ctx, user)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		updated, _ := userRepo.FindOne(ctx, 1)
		if updated.Email != "alice.updated@example.com" {
			t.Errorf("Expected email to be updated, got '%s'", updated.Email)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		err := userRepo.Delete(ctx, 1)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		_, err = userRepo.FindOne(ctx, 1)
		if err == nil {
			t.Error("Expected error when finding deleted user")
		}
	})
}

func TestQueryBuilder(t *testing.T) {
	db, session := setupTestDB(t)
	defer db.Close()

	userRepo := sqlc.NewRepository[models.User](session)
	ctx := context.Background()

	// Create test data
	users := []*models.User{
		{Username: "alice", Email: "alice@example.com", CreatedAt: time.Now()},
		{Username: "bob", Email: "bob@example.com", CreatedAt: time.Now()},
		{Username: "charlie", Email: "charlie@example.com", CreatedAt: time.Now()},
	}

	for _, u := range users {
		_ = userRepo.Create(ctx, u)
	}

	t.Run("Count", func(t *testing.T) {
		count, err := userRepo.Query().Count(ctx)
		if err != nil {
			t.Fatalf("Count failed: %v", err)
		}

		if count != 3 {
			t.Errorf("Expected count 3, got %d", count)
		}
	})

	t.Run("Find", func(t *testing.T) {
		results, err := userRepo.Query().Find(ctx)
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}

		if len(results) != 3 {
			t.Errorf("Expected 3 results, got %d", len(results))
		}
	})

	t.Run("First", func(t *testing.T) {
		user, err := userRepo.Query().First(ctx)
		if err != nil {
			t.Fatalf("First failed: %v", err)
		}

		if user == nil {
			t.Error("Expected user to be non-nil")
		}
	})

	t.Run("Limit", func(t *testing.T) {
		results, err := userRepo.Query().Limit(2).Find(ctx)
		if err != nil {
			t.Fatalf("Limit failed: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}
	})

	t.Run("Offset", func(t *testing.T) {
		results, err := userRepo.Query().Limit(10).Offset(1).Find(ctx)
		if err != nil {
			t.Fatalf("Offset failed: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 results after offset, got %d", len(results))
		}
	})

	t.Run("SelectCumulative", func(t *testing.T) {
		// First Select should replace default columns
		// Second Select should append
		q := userRepo.Query().Select(clause.Column{Name: "username"})
		q.Select(clause.Column{Name: "email"})

		// We can't easily inspect columns on private struct, but we can execute query
		// If both selected, we can access them. If not selected, they might be empty string/zero?
		// Actually if we Select into slice of pointers, we need to check if unselected fields are zero.
		// NOTE: sqlx Scan might error if we scan into struct but query has missing columns?
		// No, sqlx maps columns to fields. Missing columns = fields stay zero.

		results, err := q.Find(ctx)
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}

		if len(results) == 0 {
			t.Fatal("Expected results")
		}

		user := results[0]
		if user.Username == "" || user.Email == "" {
			t.Errorf("Expected Username and Email to be populated. Got: %v", user)
		}
		if !user.CreatedAt.IsZero() {
			// CreatedAt was NOT selected, so it should be zero (time.Time zero value)
			// UNLESS schema.SelectColumns() was used which includes created_at.
			// Default columns includes ALL.
			// Our cumulative select should have REPLACED default with [username].
			// Then appended [email].
			// So [username, email].
			// created_at should be excluded.
			t.Error("Expected CreatedAt to be zero (not selected)")
		}
	})
}

func TestFieldExpressions(t *testing.T) {
	db, session := setupTestDB(t)
	defer db.Close()

	userRepo := sqlc.NewRepository[models.User](session)
	ctx := context.Background()

	// Create test data
	users := []*models.User{
		{Username: "alice", Email: "alice@example.com", CreatedAt: time.Now()},
		{Username: "bob", Email: "bob@example.com", CreatedAt: time.Now()},
		{Username: "charlie", Email: "charlie@example.com", CreatedAt: time.Now()},
	}

	for _, u := range users {
		_ = userRepo.Create(ctx, u)
	}

	t.Run("StringField.Eq", func(t *testing.T) {
		expr := generated.User.Username.Eq("alice")
		if expr == nil {
			t.Error("Expected expression to be non-nil")
		}
	})

	t.Run("StringField.Like", func(t *testing.T) {
		expr := generated.User.Username.Like("%li%")
		if expr == nil {
			t.Error("Expected expression to be non-nil")
		}
	})

	t.Run("NumberField.Gt", func(t *testing.T) {
		expr := generated.User.ID.Gt(1)
		if expr == nil {
			t.Error("Expected expression to be non-nil")
		}
	})

	t.Run("NumberField.Between", func(t *testing.T) {
		expr := generated.User.ID.Between(1, 3)
		if expr == nil {
			t.Error("Expected expression to be non-nil")
		}
	})

	t.Run("NumberField.In", func(t *testing.T) {
		expr := generated.User.ID.In(1, 2, 3)
		if expr == nil {
			t.Error("Expected expression to be non-nil")
		}
	})
}

func TestFieldConfiguration(t *testing.T) {
	t.Run("WithColumn", func(t *testing.T) {
		field := generated.User.ID.WithColumn("user_id")
		col := field.Column()

		if col.Name != "user_id" {
			t.Errorf("Expected column name 'user_id', got '%s'", col.Name)
		}
	})

	t.Run("WithTable", func(t *testing.T) {
		field := generated.User.ID.WithTable("users")
		col := field.Column()

		if col.Table != "users" {
			t.Errorf("Expected table name 'users', got '%s'", col.Table)
		}
	})

	t.Run("ChainedConfiguration", func(t *testing.T) {
		field := generated.User.ID.WithTable("users").WithColumn("user_id")
		col := field.Column()

		if col.Table != "users" || col.Name != "user_id" {
			t.Errorf("Expected table 'users' and column 'user_id', got table '%s' and column '%s'",
				col.Table, col.Name)
		}
	})
}

func TestTransactions(t *testing.T) {
	db, session := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	t.Run("SuccessfulTransaction", func(t *testing.T) {
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
			t.Fatalf("Transaction failed: %v", err)
		}

		// Verify both users were created
		userRepo := sqlc.NewRepository[models.User](session)
		count, _ := userRepo.Query().Count(ctx)
		if count != 2 {
			t.Errorf("Expected 2 users after transaction, got %d", count)
		}
	})

	t.Run("RollbackTransaction", func(t *testing.T) {
		// Clear database
		db.Exec("DELETE FROM users")

		initialCount, _ := sqlc.NewRepository[models.User](session).Query().Count(ctx)

		err := session.Transaction(ctx, func(txSession *sqlc.Session) error {
			txRepo := sqlc.NewRepository[models.User](txSession)

			user := &models.User{Username: "rollback_user", Email: "rollback@example.com", CreatedAt: time.Now()}
			if err := txRepo.Create(ctx, user); err != nil {
				return err
			}

			// Force rollback by returning error
			return sql.ErrConnDone
		})

		if err == nil {
			t.Error("Expected transaction to fail")
		}

		// Verify user was not created
		userRepo := sqlc.NewRepository[models.User](session)
		finalCount, _ := userRepo.Query().Count(ctx)
		if finalCount != initialCount {
			t.Errorf("Expected count to remain %d after rollback, got %d", initialCount, finalCount)
		}
	})
}

func TestLifecycleHooks(t *testing.T) {
	db, session := setupTestDB(t)
	defer db.Close()

	userRepo := sqlc.NewRepository[models.User](session)
	ctx := context.Background()

	t.Run("BeforeCreate", func(t *testing.T) {
		user := &models.User{
			Username: "hook_test",
			Email:    "hook@example.com",
			// CreatedAt will be set by BeforeCreate hook
		}

		err := userRepo.Create(ctx, user)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		if user.CreatedAt.IsZero() {
			t.Error("Expected CreatedAt to be set by BeforeCreate hook")
		}
	})

	t.Run("AfterCreate", func(t *testing.T) {
		user := &models.User{
			Username:  "after_hook_test",
			Email:     "after@example.com",
			CreatedAt: time.Now(),
		}

		// AfterCreate hook logs the creation
		err := userRepo.Create(ctx, user)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		// Verify user was created successfully
		if user.ID == 0 {
			t.Error("Expected user ID to be set")
		}
	})
}
