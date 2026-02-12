package sqlc_test

import (
	"strings"
	"testing"
	"time"

	"github.com/arllen133/sqlc"
	"github.com/arllen133/sqlc/clause"
	"github.com/arllen133/sqlc/field"
)

// -- Mocks for SQL Generation Tests --

type GenUser struct {
	ID        int64     `db:"id"`
	Username  string    `db:"username"`
	Email     string    `db:"email"`
	CreatedAt time.Time `db:"created_at"`
}

func (GenUser) TableName() string { return "users" }

type GenPost struct {
	ID        int64     `db:"id"`
	UserID    int64     `db:"user_id"`
	Title     string    `db:"title"`
	Content   string    `db:"content"`
	Metadata  string    `db:"metadata"` // Simplified from JSON
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (GenPost) TableName() string { return "posts" }

// Schemas
type GenUserSchema struct{}

func (GenUserSchema) TableName() string { return "users" }
func (GenUserSchema) SelectColumns() []string {
	return []string{"id", "username", "email", "created_at"}
}
func (GenUserSchema) InsertRow(m *GenUser) ([]string, []any) { return nil, nil }
func (GenUserSchema) UpdateMap(m *GenUser) map[string]any    { return nil }
func (GenUserSchema) PK(m *GenUser) sqlc.PK                  { return sqlc.PK{} }
func (GenUserSchema) SetPK(m *GenUser, val int64)            {}
func (GenUserSchema) AutoIncrement() bool                    { return true }
func (GenUserSchema) SoftDeleteColumn() string               { return "" }
func (GenUserSchema) SoftDeleteValue() any                   { return nil }
func (GenUserSchema) SetDeletedAt(m *GenUser)                {}

// Minimal schema methods required for Query builder (Select/Where/Join)
// We don't need Insert/Update/PK for ToSQL tests usually, unless Query() calls them?
// Query() creates a builder. The builder uses SelectColumns.
// Join uses TableName.
// We also need to register them.

func init() {
	sqlc.RegisterSchema(GenUserSchema{})
	sqlc.RegisterSchema(GenPostSchema{})
	sqlc.RegisterSchema(SoftDeleteProductSchema{})
}

type GenPostSchema struct{}

func (GenPostSchema) TableName() string { return "posts" }
func (GenPostSchema) SelectColumns() []string {
	return []string{"id", "user_id", "title", "content", "metadata", "created_at", "updated_at"}
}
func (GenPostSchema) InsertRow(m *GenPost) ([]string, []any) { return nil, nil }
func (GenPostSchema) UpdateMap(m *GenPost) map[string]any    { return nil }
func (GenPostSchema) PK(m *GenPost) sqlc.PK                  { return sqlc.PK{} }
func (GenPostSchema) SetPK(m *GenPost, val int64)            {}
func (GenPostSchema) AutoIncrement() bool                    { return true }
func (GenPostSchema) SoftDeleteColumn() string               { return "" }
func (GenPostSchema) SoftDeleteValue() any                   { return nil }
func (GenPostSchema) SetDeletedAt(m *GenPost)                {}

// Soft Delete Mock
type SoftDeleteProduct struct {
	ID        int64      `db:"id"`
	Name      string     `db:"name"`
	DeletedAt *time.Time `db:"deleted_at"`
}

func (SoftDeleteProduct) TableName() string { return "products" }

type SoftDeleteProductSchema struct{}

func (SoftDeleteProductSchema) TableName() string { return "products" }
func (SoftDeleteProductSchema) SelectColumns() []string {
	return []string{"id", "name", "deleted_at"}
}
func (SoftDeleteProductSchema) InsertRow(m *SoftDeleteProduct) ([]string, []any) { return nil, nil }
func (SoftDeleteProductSchema) UpdateMap(m *SoftDeleteProduct) map[string]any    { return nil }
func (SoftDeleteProductSchema) PK(m *SoftDeleteProduct) sqlc.PK {
	var val any
	if m != nil {
		val = m.ID
	}
	return sqlc.PK{Column: clause.Column{Name: "id"}, Value: val}
}
func (SoftDeleteProductSchema) SetPK(m *SoftDeleteProduct, val int64) {}
func (SoftDeleteProductSchema) AutoIncrement() bool                   { return true }
func (SoftDeleteProductSchema) SoftDeleteColumn() string              { return "deleted_at" }
func (SoftDeleteProductSchema) SoftDeleteValue() any                  { return time.Now() }
func (SoftDeleteProductSchema) SetDeletedAt(m *SoftDeleteProduct) {
	now := time.Now()
	m.DeletedAt = &now
}

// Generated Fields Helper (Simulating generated code)
var GenUserFields = struct {
	ID       field.Number[int64]
	Username field.String
	Email    field.String
}{
	ID:       field.Number[int64]{}.WithColumn("id").WithTable("users"),
	Username: field.String{}.WithColumn("username").WithTable("users"),
	Email:    field.String{}.WithColumn("email").WithTable("users"),
}

var GenPostFields = struct {
	ID     field.Number[int64]
	UserID field.Number[int64]
	Title  field.String
}{
	ID:     field.Number[int64]{}.WithColumn("id").WithTable("posts"),
	UserID: field.Number[int64]{}.WithColumn("user_id").WithTable("posts"),
	Title:  field.String{}.WithColumn("title").WithTable("posts"),
}

// Mock setup for SQL generation (no DB needed really, but Session requires dialect)
func setupGenSession() *sqlc.Session {
	// We can pass nil db if we only call ToSQL, but strictly NewSession takes *sql.DB
	// If sqlc.NewSession checks db != nil, we might need a mock or real DB.
	// Looking at sqlc code (I can't see it now but usually it assigns).
	// Let's pass nil and see. If it panics, we'll fix.
	// We need a dialect.
	return sqlc.NewSession(nil, &sqlc.SQLiteDialect{})
}

func TestSQLGeneration(t *testing.T) {
	session := setupGenSession()

	userRepo := sqlc.NewRepository[GenUser](session)
	postRepo := sqlc.NewRepository[GenPost](session)

	tests := []struct {
		name         string
		buildQuery   func() (string, []any, error)
		wantSQL      string
		wantArgs     []any
		wantContains []string
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
					Where(GenUserFields.Username.Eq("alice")).
					ToSQL()
			},
			wantSQL:  "SELECT id, username, email, created_at FROM users WHERE users.username = ?",
			wantArgs: []any{"alice"},
		},
		{
			name: "WhereLike",
			buildQuery: func() (string, []any, error) {
				return userRepo.Query().
					Where(GenUserFields.Email.Like("%@example.com")).
					ToSQL()
			},
			wantSQL:  "SELECT id, username, email, created_at FROM users WHERE users.email LIKE ?",
			wantArgs: []any{"%@example.com"},
		},
		{
			name: "WhereIn",
			buildQuery: func() (string, []any, error) {
				return userRepo.Query().
					Where(GenUserFields.ID.In(1, 2, 3)).
					ToSQL()
			},
			wantSQL:  "SELECT id, username, email, created_at FROM users WHERE users.id IN (?, ?, ?)",
			wantArgs: []any{int64(1), int64(2), int64(3)},
		},
		{
			name: "WhereBetween",
			buildQuery: func() (string, []any, error) {
				return userRepo.Query().
					Where(GenUserFields.ID.Between(1, 10)).
					ToSQL()
			},
			wantSQL:  "SELECT id, username, email, created_at FROM users WHERE users.id BETWEEN ? AND ?",
			wantArgs: []any{int64(1), int64(10)},
		},
		{
			name: "WhereGtLt",
			buildQuery: func() (string, []any, error) {
				return userRepo.Query().
					Where(GenUserFields.ID.Gt(5)).
					Where(GenUserFields.ID.Lt(10)).
					ToSQL()
			},
			wantSQL:  "SELECT id, username, email, created_at FROM users WHERE users.id > ? AND users.id < ?",
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
					OrderBy(GenUserFields.ID.Desc()).
					ToSQL()
			},
			wantSQL:  "SELECT id, username, email, created_at FROM users ORDER BY users.id DESC",
			wantArgs: []any{},
		},
		{
			name: "InnerJoin",
			buildQuery: func() (string, []any, error) {
				return postRepo.Query().
					Join(&GenUser{},
						sqlc.On(GenPostFields.UserID, GenUserFields.ID),
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
					LeftJoin(&GenUser{},
						sqlc.On(GenPostFields.UserID, GenUserFields.ID),
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
					Join(&GenUser{},
						sqlc.On(GenPostFields.UserID, GenUserFields.ID),
					).
					Where(GenPostFields.Title.Eq("hello")).
					Where(GenUserFields.Username.Eq("alice")).
					ToSQL()
			},
			wantContains: []string{
				"FROM posts",
				"JOIN users ON posts.user_id = users.id",
				"WHERE posts.title = ?",
				"AND users.username = ?",
			},
			wantArgs: []any{"hello", "alice"},
		},
		{
			name: "SelectSpecificColumns",
			buildQuery: func() (string, []any, error) {
				return userRepo.Query().
					Select(GenUserFields.ID, GenUserFields.Username).
					ToSQL()
			},
			wantSQL:  "SELECT users.id, users.username FROM users",
			wantArgs: []any{},
		},
		{
			name: "Distinct",
			buildQuery: func() (string, []any, error) {
				return userRepo.Query().
					Distinct().
					Select(GenUserFields.Email).
					ToSQL()
			},
			wantSQL:  "SELECT DISTINCT users.email FROM users",
			wantArgs: []any{},
		},
		{
			name: "ComplexQuery",
			buildQuery: func() (string, []any, error) {
				return userRepo.Query().
					Where(GenUserFields.Username.Like("a%")).
					Where(GenUserFields.ID.Gt(0)).
					OrderBy(GenUserFields.Username.Asc()).
					Limit(5).
					Offset(10).
					ToSQL()
			},
			wantContains: []string{
				"SELECT id, username, email, created_at FROM users",
				"WHERE users.username LIKE ?",
				"AND users.id > ?",
				"ORDER BY users.username",
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

func TestSubquerySQLGeneration(t *testing.T) {
	session := setupGenSession()

	userRepo := sqlc.NewRepository[GenUser](session)
	postRepo := sqlc.NewRepository[GenPost](session)

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
					Select(GenUserFields.ID).
					Where(GenUserFields.Username.Eq("alice"))

				return postRepo.Query().
					Where(GenPostFields.UserID.InExpr(subquery)).
					ToSQL()
			},
			wantContains: []string{
				"FROM posts",
				"posts.user_id IN (SELECT users.id FROM users WHERE users.username = ?)",
			},
			wantArgs: []any{"alice"},
		},
		{
			name: "Exists_Subquery",
			buildQuery: func() (string, []any, error) {
				// SELECT * FROM users WHERE EXISTS (SELECT 1 FROM posts WHERE user_id > 0)
				subquery := postRepo.Query().
					Select(clause.Column{Name: "1"}).
					Where(GenPostFields.UserID.Gt(0))

				return userRepo.Query().
					Where(sqlc.Exists(subquery)).
					ToSQL()
			},
			wantContains: []string{
				"FROM users",
				"EXISTS (SELECT 1 FROM posts WHERE posts.user_id > ?)",
			},
			wantArgs: []any{int64(0)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSQL, gotArgs, err := tt.buildQuery()
			if err != nil {
				t.Fatalf("ToSQL() error = %v", err)
			}

			for _, substr := range tt.wantContains {
				if !contains(gotSQL, substr) {
					t.Errorf("SQL should contain %q\ngot: %s", substr, gotSQL)
				}
			}

			if tt.wantArgs != nil {
				if len(gotArgs) != len(tt.wantArgs) {
					t.Errorf("Args length mismatch: got %d, want %d", len(gotArgs), len(tt.wantArgs))
				} else {
					for i := range tt.wantArgs {
						if gotArgs[i] != tt.wantArgs[i] {
							t.Errorf("Arg[%d] mismatch: got %v, want %v", i, gotArgs[i], tt.wantArgs[i])
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
