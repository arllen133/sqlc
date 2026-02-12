package sqlc_test

import (
	"database/sql"
	"os"
	"testing"

	"github.com/arllen133/sqlc"
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

	session := sqlc.NewSession(db, dialect)
	return db, session
}
