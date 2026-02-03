package sqlc_test

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

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
		if err == nil {
			_, err = db.Exec(`CREATE TABLE IF NOT EXISTS posts (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				user_id INTEGER,
				title TEXT,
				content TEXT,
				metadata TEXT,
				created_at DATETIME,
				updated_at DATETIME
			)`)
		}
	case "mysql":
		_, err = db.Exec(`CREATE TABLE IF NOT EXISTS users (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			username VARCHAR(255),
			email VARCHAR(255),
			created_at DATETIME
		)`)
		if err == nil {
			_, err = db.Exec(`CREATE TABLE IF NOT EXISTS posts (
				id BIGINT PRIMARY KEY AUTO_INCREMENT,
				user_id BIGINT,
				title VARCHAR(200),
				content TEXT,
				metadata JSON,
				created_at DATETIME,
				updated_at DATETIME
			)`)
		}
		// Truncate to ensure clean state for MySQL/PG which persist
		db.Exec("TRUNCATE TABLE users")
		db.Exec("TRUNCATE TABLE posts")
	case "postgres":
		_, err = db.Exec(`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username TEXT,
			email TEXT,
			created_at TIMESTAMP
		)`)
		if err == nil {
			_, err = db.Exec(`CREATE TABLE IF NOT EXISTS posts (
				id SERIAL PRIMARY KEY,
				user_id BIGINT,
				title TEXT,
				content TEXT,
				metadata JSONB,
				created_at TIMESTAMP,
				updated_at TIMESTAMP
			)`)
		}
		db.Exec("TRUNCATE TABLE users RESTART IDENTITY")
		db.Exec("TRUNCATE TABLE posts RESTART IDENTITY")
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
	postRepo := sqlc.NewRepository[models.Post](session)
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

	_ = postRepo.Create(ctx, &models.Post{
		UserID:    users[0].ID,
		Title:     "hello",
		Content:   "world",
		Metadata:  sqlc.NewJSON(models.PostMetadata{}),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

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

	t.Run("Take", func(t *testing.T) {
		user, err := userRepo.Query().Take(ctx)
		if err != nil {
			t.Fatalf("Take failed: %v", err)
		}
		if user == nil {
			t.Error("Expected user to be non-nil")
		}
	})

	t.Run("Last", func(t *testing.T) {
		var maxID int64
		for _, u := range users {
			if u.ID > maxID {
				maxID = u.ID
			}
		}
		user, err := userRepo.Query().Last(ctx)
		if err != nil {
			t.Fatalf("Last failed: %v", err)
		}
		if user.ID != maxID {
			t.Fatalf("Expected last ID %d, got %d", maxID, user.ID)
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
		// Select replaces columns (not cumulative), so select both in one call
		q := userRepo.Query().Select(clause.Column{Name: "username"}, clause.Column{Name: "email"})

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
			// Our Select replaced default with [username, email].
			// created_at should be excluded.
			t.Error("Expected CreatedAt to be zero (not selected)")
		}
	})

	t.Run("Join", func(t *testing.T) {
		results, err := postRepo.Query().
			Join(&generated.User,
				sqlc.On(generated.Post.UserID, generated.User.ID),
			).
			Find(ctx)
		if err != nil {
			t.Fatalf("Join failed: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("Expected 1 post, got %d", len(results))
		}
	})

	t.Run("JoinWithWhere", func(t *testing.T) {
		results, err := postRepo.Query().
			Join(&generated.User,
				sqlc.On(generated.Post.UserID, generated.User.ID),
			).
			Where(generated.Post.Title.Eq("hello")).
			Where(generated.User.Username.Eq("alice")).
			Find(ctx)
		if err != nil {
			t.Fatalf("JoinWithWhere failed: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("Expected 1 post, got %d", len(results))
		}
	})
}

func TestSQLGeneration(t *testing.T) {
	_, session := setupTestDB(t)

	userRepo := sqlc.NewRepository[models.User](session)
	postRepo := sqlc.NewRepository[models.Post](session)

	tests := []struct {
		name         string
		buildQuery   func() (string, []any, error)
		wantSQL      string
		wantArgs     []any
		wantContains []string // for partial matching when full SQL varies
	}{
		{
			name: "SimpleSelect",
			buildQuery: func() (string, []any, error) {
				return userRepo.Query().ToSQL()
			},
			wantSQL:  "SELECT id, username, email, created_at FROM users",
			wantArgs: []any{},
		},
		{
			name: "WhereEq",
			buildQuery: func() (string, []any, error) {
				return userRepo.Query().
					Where(generated.User.Username.Eq("alice")).
					ToSQL()
			},
			wantSQL:  "SELECT id, username, email, created_at FROM users WHERE username = ?",
			wantArgs: []any{"alice"},
		},
		{
			name: "WhereLike",
			buildQuery: func() (string, []any, error) {
				return userRepo.Query().
					Where(generated.User.Email.Like("%@example.com")).
					ToSQL()
			},
			wantSQL:  "SELECT id, username, email, created_at FROM users WHERE email LIKE ?",
			wantArgs: []any{"%@example.com"},
		},
		{
			name: "WhereIn",
			buildQuery: func() (string, []any, error) {
				return userRepo.Query().
					Where(generated.User.ID.In(1, 2, 3)).
					ToSQL()
			},
			wantSQL:  "SELECT id, username, email, created_at FROM users WHERE id IN (?, ?, ?)",
			wantArgs: []any{int64(1), int64(2), int64(3)},
		},
		{
			name: "WhereBetween",
			buildQuery: func() (string, []any, error) {
				return userRepo.Query().
					Where(generated.User.ID.Between(1, 10)).
					ToSQL()
			},
			wantSQL:  "SELECT id, username, email, created_at FROM users WHERE id BETWEEN ? AND ?",
			wantArgs: []any{int64(1), int64(10)},
		},
		{
			name: "WhereGtLt",
			buildQuery: func() (string, []any, error) {
				return userRepo.Query().
					Where(generated.User.ID.Gt(5)).
					Where(generated.User.ID.Lt(10)).
					ToSQL()
			},
			wantSQL:  "SELECT id, username, email, created_at FROM users WHERE id > ? AND id < ?",
			wantArgs: []any{int64(5), int64(10)},
		},
		{
			name: "LimitOffset",
			buildQuery: func() (string, []any, error) {
				return userRepo.Query().
					Limit(10).
					Offset(20).
					ToSQL()
			},
			wantSQL:  "SELECT id, username, email, created_at FROM users LIMIT 10 OFFSET 20",
			wantArgs: []any{},
		},
		{
			name: "OrderBy",
			buildQuery: func() (string, []any, error) {
				return userRepo.Query().
					OrderBy(generated.User.ID.Desc()).
					ToSQL()
			},
			wantSQL:  "SELECT id, username, email, created_at FROM users ORDER BY id DESC",
			wantArgs: []any{},
		},
		{
			name: "OrderByMultiple",
			buildQuery: func() (string, []any, error) {
				return userRepo.Query().
					OrderBy(generated.User.Username.Asc(), generated.User.ID.Desc()).
					ToSQL()
			},
			wantSQL:  "SELECT id, username, email, created_at FROM users ORDER BY username, id DESC",
			wantArgs: []any{},
		},
		{
			name: "InnerJoin",
			buildQuery: func() (string, []any, error) {
				return postRepo.Query().
					Join(&generated.User,
						sqlc.On(generated.Post.UserID, generated.User.ID),
					).
					ToSQL()
			},
			wantContains: []string{
				"SELECT posts.id, posts.user_id, posts.title, posts.content, posts.metadata, posts.created_at, posts.updated_at FROM posts",
				"JOIN users ON posts.user_id = users.id",
			},
		},
		{
			name: "LeftJoin",
			buildQuery: func() (string, []any, error) {
				return postRepo.Query().
					LeftJoin(&generated.User,
						sqlc.On(generated.Post.UserID, generated.User.ID),
					).
					ToSQL()
			},
			wantContains: []string{
				"FROM posts",
				"LEFT JOIN users ON posts.user_id = users.id",
			},
		},
		{
			name: "JoinWithWhere",
			buildQuery: func() (string, []any, error) {
				return postRepo.Query().
					Join(&generated.User,
						sqlc.On(generated.Post.UserID, generated.User.ID),
					).
					Where(generated.Post.Title.Eq("hello")).
					Where(generated.User.Username.Eq("alice")).
					ToSQL()
			},
			wantContains: []string{
				"FROM posts",
				"JOIN users ON posts.user_id = users.id",
				"WHERE title = ?",
				"AND username = ?",
			},
			wantArgs: []any{"hello", "alice"},
		},
		{
			name: "SelectSpecificColumns",
			buildQuery: func() (string, []any, error) {
				return userRepo.Query().
					Select(generated.User.ID, generated.User.Username).
					ToSQL()
			},
			wantSQL:  "SELECT id, username FROM users",
			wantArgs: []any{},
		},
		{
			name: "ComplexQuery",
			buildQuery: func() (string, []any, error) {
				return userRepo.Query().
					Where(generated.User.Username.Like("a%")).
					Where(generated.User.ID.Gt(0)).
					OrderBy(generated.User.Username.Asc()).
					Limit(5).
					Offset(10).
					ToSQL()
			},
			wantContains: []string{
				"SELECT id, username, email, created_at FROM users",
				"WHERE username LIKE ?",
				"AND id > ?",
				"ORDER BY username",
				"LIMIT 5",
				"OFFSET 10",
			},
			wantArgs: []any{"a%", int64(0)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSQL, gotArgs, err := tt.buildQuery()
			if err != nil {
				t.Fatalf("ToSQL() error = %v", err)
			}

			// Full SQL match
			if tt.wantSQL != "" {
				if gotSQL != tt.wantSQL {
					t.Errorf("SQL mismatch:\ngot:  %s\nwant: %s", gotSQL, tt.wantSQL)
				}
			}

			// Partial matching for complex queries
			for _, substr := range tt.wantContains {
				if !contains(gotSQL, substr) {
					t.Errorf("SQL should contain %q\ngot: %s", substr, gotSQL)
				}
			}

			// Args matching
			if tt.wantArgs != nil {
				if len(gotArgs) != len(tt.wantArgs) {
					t.Errorf("Args length mismatch: got %d, want %d\ngot:  %v\nwant: %v", len(gotArgs), len(tt.wantArgs), gotArgs, tt.wantArgs)
				} else {
					for i := range tt.wantArgs {
						if gotArgs[i] != tt.wantArgs[i] {
							t.Errorf("Arg[%d] mismatch: got %v (%T), want %v (%T)", i, gotArgs[i], gotArgs[i], tt.wantArgs[i], tt.wantArgs[i])
						}
					}
				}
			}
		})
	}
}

// contains checks if s contains substr (case-insensitive for SQL)
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestSubquerySQLGeneration(t *testing.T) {
	_, session := setupTestDB(t)

	userRepo := sqlc.NewRepository[models.User](session)
	postRepo := sqlc.NewRepository[models.Post](session)

	tests := []struct {
		name         string
		buildQuery   func() (string, []any, error)
		wantContains []string
		wantArgs     []any
	}{
		{
			name: "InExpr_Subquery",
			buildQuery: func() (string, []any, error) {
				// SELECT * FROM posts WHERE user_id IN (SELECT id FROM users WHERE username = 'alice')
				subquery := userRepo.Query().
					Select(generated.User.ID).
					Where(generated.User.Username.Eq("alice"))

				return postRepo.Query().
					Where(generated.Post.UserID.InExpr(subquery)).
					ToSQL()
			},
			wantContains: []string{
				"FROM posts",
				"user_id IN (SELECT id FROM users WHERE username = ?)",
			},
			wantArgs: []any{"alice"},
		},
		{
			name: "NotInExpr_Subquery",
			buildQuery: func() (string, []any, error) {
				// SELECT * FROM posts WHERE user_id NOT IN (SELECT id FROM users WHERE username = 'bob')
				subquery := userRepo.Query().
					Select(generated.User.ID).
					Where(generated.User.Username.Eq("bob"))

				return postRepo.Query().
					Where(generated.Post.UserID.NotInExpr(subquery)).
					ToSQL()
			},
			wantContains: []string{
				"FROM posts",
				"user_id NOT IN (SELECT id FROM users WHERE username = ?)",
			},
			wantArgs: []any{"bob"},
		},
		{
			name: "Exists_Subquery",
			buildQuery: func() (string, []any, error) {
				// SELECT * FROM users WHERE EXISTS (SELECT 1 FROM posts WHERE user_id > 0)
				subquery := postRepo.Query().
					Select(clause.Column{Name: "1"}).
					Where(generated.Post.UserID.Gt(0))

				return userRepo.Query().
					Where(sqlc.Exists(subquery)).
					ToSQL()
			},
			wantContains: []string{
				"FROM users",
				"EXISTS (SELECT 1 FROM posts WHERE user_id > ?)",
			},
			wantArgs: []any{int64(0)},
		},
		{
			name: "NotExists_Subquery",
			buildQuery: func() (string, []any, error) {
				// SELECT * FROM users WHERE NOT EXISTS (SELECT 1 FROM posts WHERE user_id = 999)
				subquery := postRepo.Query().
					Select(clause.Column{Name: "1"}).
					Where(generated.Post.UserID.Eq(999))

				return userRepo.Query().
					Where(sqlc.NotExists(subquery)).
					ToSQL()
			},
			wantContains: []string{
				"FROM users",
				"NOT EXISTS (SELECT 1 FROM posts WHERE user_id = ?)",
			},
			wantArgs: []any{int64(999)},
		},
		{
			name: "NestedSubquery_Complex",
			buildQuery: func() (string, []any, error) {
				// Complex: SELECT * FROM posts WHERE user_id IN (SELECT id FROM users WHERE id > 0 AND username LIKE 'a%')
				subquery := userRepo.Query().
					Select(generated.User.ID).
					Where(generated.User.ID.Gt(0)).
					Where(generated.User.Username.Like("a%"))

				return postRepo.Query().
					Where(generated.Post.UserID.InExpr(subquery)).
					Limit(10).
					ToSQL()
			},
			wantContains: []string{
				"FROM posts",
				"user_id IN (SELECT id FROM users WHERE id > ? AND username LIKE ?)",
				"LIMIT 10",
			},
			wantArgs: []any{int64(0), "a%"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSQL, gotArgs, err := tt.buildQuery()
			if err != nil {
				t.Fatalf("ToSQL() error = %v", err)
			}

			// Partial matching for subquery SQL
			for _, substr := range tt.wantContains {
				if !contains(gotSQL, substr) {
					t.Errorf("SQL should contain %q\ngot: %s", substr, gotSQL)
				}
			}

			// Args matching
			if tt.wantArgs != nil {
				if len(gotArgs) != len(tt.wantArgs) {
					t.Errorf("Args length mismatch: got %d, want %d\ngot:  %v\nwant: %v", len(gotArgs), len(tt.wantArgs), gotArgs, tt.wantArgs)
				} else {
					for i := range tt.wantArgs {
						if gotArgs[i] != tt.wantArgs[i] {
							t.Errorf("Arg[%d] mismatch: got %v (%T), want %v (%T)", i, gotArgs[i], gotArgs[i], tt.wantArgs[i], tt.wantArgs[i])
						}
					}
				}
			}
		})
	}
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
