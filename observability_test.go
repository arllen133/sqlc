package sqlc_test

import (
	"bytes"
	"context"
	"database/sql"
	"log/slog"
	"testing"
	"time"

	"github.com/arllen133/sqlc"
	"github.com/arllen133/sqlc/clause"
	_ "github.com/mattn/go-sqlite3"
)

// ObsTestModel is a simple model for observability tests
type ObsTestModel struct {
	ID   int64  `db:"id,primaryKey,autoIncrement"`
	Name string `db:"name"`
}

// ObsTestSchema implements Schema for ObsTestModel
type obsTestSchema struct{}

func (s *obsTestSchema) TableName() string { return "obs_test" }
func (s *obsTestSchema) SelectColumns() []string {
	return []string{"id", "name"}
}
func (s *obsTestSchema) InsertRow(m *ObsTestModel) ([]string, []any) {
	if m.ID != 0 {
		return []string{"id", "name"}, []any{m.ID, m.Name}
	}
	return []string{"name"}, []any{m.Name}
}
func (s *obsTestSchema) UpdateMap(m *ObsTestModel) map[string]any {
	return map[string]any{"name": m.Name}
}
func (s *obsTestSchema) PK(m *ObsTestModel) sqlc.PK {
	var val any
	if m != nil {
		val = m.ID
	}
	return sqlc.PK{Column: clause.Column{Name: "id"}, Value: val}
}
func (s *obsTestSchema) SetPK(m *ObsTestModel, val int64) { m.ID = val }
func (s *obsTestSchema) AutoIncrement() bool              { return true }

var ObsTest = obsTestSchema{}

func init() {
	sqlc.RegisterSchema(&ObsTest)
}

func setupObsTestDB(t *testing.T) (*sql.DB, func()) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS obs_test (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	return db, func() { db.Close() }
}

func TestWithLogger(t *testing.T) {
	db, cleanup := setupObsTestDB(t)
	defer cleanup()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	sess := sqlc.NewSession(db, &sqlc.SQLiteDialect{},
		sqlc.WithLogger(logger),
		sqlc.WithQueryLogging(true),
	)

	repo := sqlc.NewRepository[ObsTestModel](sess)
	ctx := context.Background()

	// Create a record
	m := &ObsTestModel{Name: "Test"}
	err := repo.Create(ctx, m)
	if err != nil {
		t.Fatalf("failed to create: %v", err)
	}

	// Check log output
	logOutput := buf.String()
	if logOutput == "" {
		t.Error("expected log output, got empty")
	}
}

func TestWithSlowQueryThreshold(t *testing.T) {
	db, cleanup := setupObsTestDB(t)
	defer cleanup()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	sess := sqlc.NewSession(db, &sqlc.SQLiteDialect{},
		sqlc.WithLogger(logger),
		sqlc.WithSlowQueryThreshold(1*time.Nanosecond), // Very low threshold to trigger warning
	)

	repo := sqlc.NewRepository[ObsTestModel](sess)
	ctx := context.Background()

	// Create a record
	m := &ObsTestModel{Name: "Test"}
	_ = repo.Create(ctx, m)

	// Check for slow query warning
	logOutput := buf.String()
	if !bytes.Contains(buf.Bytes(), []byte("slow query")) {
		t.Errorf("expected 'slow query' warning in log, got: %s", logOutput)
	}
}

func TestWithDefaultTracer(t *testing.T) {
	db, cleanup := setupObsTestDB(t)
	defer cleanup()

	// Just test that it doesn't panic
	sess := sqlc.NewSession(db, &sqlc.SQLiteDialect{},
		sqlc.WithDefaultTracer(),
	)

	repo := sqlc.NewRepository[ObsTestModel](sess)
	ctx := context.Background()

	m := &ObsTestModel{Name: "Test"}
	err := repo.Create(ctx, m)
	if err != nil {
		t.Fatalf("failed to create with tracer: %v", err)
	}

	// Verify the record was created
	found, err := repo.FindOne(ctx, m.ID)
	if err != nil {
		t.Fatalf("failed to find: %v", err)
	}
	if found.Name != "Test" {
		t.Errorf("expected name 'Test', got '%s'", found.Name)
	}
}

func TestWithDefaultMeter(t *testing.T) {
	db, cleanup := setupObsTestDB(t)
	defer cleanup()

	// Just test that it doesn't panic
	sess := sqlc.NewSession(db, &sqlc.SQLiteDialect{},
		sqlc.WithDefaultMeter(),
	)

	repo := sqlc.NewRepository[ObsTestModel](sess)
	ctx := context.Background()

	m := &ObsTestModel{Name: "Test"}
	err := repo.Create(ctx, m)
	if err != nil {
		t.Fatalf("failed to create with meter: %v", err)
	}

	// Verify the record was created
	found, err := repo.FindOne(ctx, m.ID)
	if err != nil {
		t.Fatalf("failed to find: %v", err)
	}
	if found.Name != "Test" {
		t.Errorf("expected name 'Test', got '%s'", found.Name)
	}
}

func TestCombinedObservability(t *testing.T) {
	db, cleanup := setupObsTestDB(t)
	defer cleanup()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	sess := sqlc.NewSession(db, &sqlc.SQLiteDialect{},
		sqlc.WithLogger(logger),
		sqlc.WithQueryLogging(true),
		sqlc.WithDefaultTracer(),
		sqlc.WithDefaultMeter(),
		sqlc.WithSlowQueryThreshold(100*time.Millisecond),
	)

	repo := sqlc.NewRepository[ObsTestModel](sess)
	ctx := context.Background()

	// Perform CRUD operations
	m := &ObsTestModel{Name: "Combined Test"}
	_ = repo.Create(ctx, m)

	_, _ = repo.FindOne(ctx, m.ID)

	m.Name = "Updated Combined Test"
	_ = repo.Update(ctx, m)

	_ = repo.Delete(ctx, m.ID)

	// Just verify no panics and some logging occurred
	if buf.Len() == 0 {
		t.Error("expected some log output")
	}
}
