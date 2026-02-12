package benchmarks

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/arllen133/sqlc"
	"github.com/arllen133/sqlc/clause"
	_ "github.com/mattn/go-sqlite3"
)

// -- Benchmark Models --

type BenchUser struct {
	ID        int64     `db:"id,primaryKey,autoIncrement"`
	Username  string    `db:"username"`
	Email     string    `db:"email"`
	CreatedAt time.Time `db:"created_at"`
}

func (BenchUser) TableName() string { return "bench_users" }

type BenchUserSchema struct{}

func (BenchUserSchema) TableName() string { return "bench_users" }
func (BenchUserSchema) SelectColumns() []string {
	return []string{"id", "username", "email", "created_at"}
}
func (BenchUserSchema) InsertRow(m *BenchUser) ([]string, []any) {
	var cols []string
	var vals []any
	if m.ID != 0 {
		cols = append(cols, "id")
		vals = append(vals, m.ID)
	}
	cols = append(cols, "username")
	vals = append(vals, m.Username)
	cols = append(cols, "email")
	vals = append(vals, m.Email)
	cols = append(cols, "created_at")
	vals = append(vals, m.CreatedAt)
	return cols, vals
}
func (BenchUserSchema) UpdateMap(m *BenchUser) map[string]any {
	return map[string]any{"username": m.Username, "email": m.Email, "created_at": m.CreatedAt}
}
func (BenchUserSchema) PK(m *BenchUser) sqlc.PK {
	var val any
	if m != nil {
		val = m.ID
	}
	return sqlc.PK{
		Column: clause.Column{Name: "id"},
		Value:  val,
	}
}
func (BenchUserSchema) SetPK(m *BenchUser, val int64) {
	m.ID = val
}
func (BenchUserSchema) AutoIncrement() bool       { return true }
func (BenchUserSchema) SoftDeleteColumn() string  { return "" }
func (BenchUserSchema) SoftDeleteValue() any      { return nil }
func (BenchUserSchema) SetDeletedAt(m *BenchUser) {}

func init() {
	sqlc.RegisterSchema(BenchUserSchema{})
}

func setupBenchDB(b *testing.B) (*sql.DB, *sqlc.Session) {
	// Check Env
	driver := os.Getenv("TEST_DRIVER")
	dsn := os.Getenv("TEST_DSN")

	if driver == "" {
		driver = "sqlite3"
		dsn = ":memory:"
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}

	// Create Table
	// Simplified schema creation
	query := `CREATE TABLE IF NOT EXISTS bench_users (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            username TEXT,
            email TEXT,
            created_at DATETIME
        )`
	if driver == "mysql" {
		query = `CREATE TABLE IF NOT EXISTS bench_users (
            id BIGINT PRIMARY KEY AUTO_INCREMENT,
            username VARCHAR(255),
            email VARCHAR(255),
            created_at DATETIME
        )`
	} else if driver == "postgres" {
		query = `CREATE TABLE IF NOT EXISTS bench_users (
            id SERIAL PRIMARY KEY,
            username TEXT,
            email TEXT,
            created_at TIMESTAMP
        )`
	}

	if _, err := db.Exec(query); err != nil {
		b.Fatalf("Failed to create table: %v", err)
	}

	// Clear data
	if _, err := db.Exec("DELETE FROM bench_users"); err != nil {
		b.Fatalf("Failed to clear table: %v", err)
	}
	if driver == "postgres" {
		if _, err := db.Exec("TRUNCATE TABLE bench_users RESTART IDENTITY"); err != nil {
			b.Fatalf("Failed to truncate table: %v", err)
		}
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

	return db, sqlc.NewSession(db, dialect)
}

func BenchmarkInsert(b *testing.B) {
	db, session := setupBenchDB(b)
	defer db.Close()

	repo := sqlc.NewRepository[BenchUser](session)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		user := &BenchUser{
			Username:  "bench",
			Email:     fmt.Sprintf("bench%d@test.com", i),
			CreatedAt: time.Now(),
		}
		if err := repo.Create(ctx, user); err != nil {
			b.Fatalf("Create failed: %v", err)
		}
	}
}

func BenchmarkBatchInsert100(b *testing.B) {
	db, session := setupBenchDB(b)
	defer db.Close()

	repo := sqlc.NewRepository[BenchUser](session)
	ctx := context.Background()

	batchSize := 100
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		var users []*BenchUser
		for j := 0; j < batchSize; j++ {
			users = append(users, &BenchUser{
				Username:  "batch",
				Email:     fmt.Sprintf("batch%d_%d@test.com", i, j),
				CreatedAt: time.Now(),
			})
		}
		b.StartTimer()

		if err := repo.BatchCreate(ctx, users); err != nil {
			b.Fatalf("BatchCreate failed: %v", err)
		}
	}
}

func BenchmarkFindID(b *testing.B) {
	db, session := setupBenchDB(b)
	defer db.Close()

	repo := sqlc.NewRepository[BenchUser](session)
	ctx := context.Background()

	// Pre-seed
	user := &BenchUser{Username: "find_me", Email: "find@test.com"}
	if err := repo.Create(ctx, user); err != nil {
		b.Fatalf("Failed to seed user: %v", err)
	}
	id := user.ID

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := repo.FindOne(ctx, id)
		if err != nil {
			b.Fatalf("FindOne failed: %v", err)
		}
	}
}
