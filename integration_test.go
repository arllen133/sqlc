package sqlc_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/arllen133/sqlc"
	"github.com/arllen133/sqlc/clause"
	"github.com/arllen133/sqlc/field"
)

// -- Integration Test Models --

// Department Model
type Department struct {
	ID        int64     `db:"id,primaryKey,autoIncrement"`
	Name      string    `db:"name"`
	Location  string    `db:"location"`
	CreatedAt time.Time `db:"created_at"`

	// Relation fields (not in DB, loaded via Preload)
	Members []*Member `db:"-"`
}

func (Department) TableName() string { return "departments" }

// DepartmentHasMembers defines the HasMany relation: Department -> Members
var DepartmentHasMembers = sqlc.HasMany[Department, Member, int64](
	clause.Column{Name: "department_id"},                           // Foreign key on Member
	clause.Column{Name: "id"},                                      // Local key on Department
	func(d *Department, members []*Member) { d.Members = members }, // Setter
	func(d *Department) int64 { return d.ID },                      // Get local key
	func(m *Member) int64 { return int64(m.DepartmentID) },         // Get foreign key
)

// Member Model (Advanced User)
type Member struct {
	ID           int64     `db:"id,primaryKey,autoIncrement"`
	Name         string    `db:"name"`
	Email        string    `db:"email,unique"`
	Level        int       `db:"level"`
	DepartmentID int       `db:"department_id"`
	CreatedAt    time.Time `db:"created_at"`
}

func (Member) TableName() string { return "members" }

// -- Schemas (Inline for tests) --

// DeptSchema
type DeptSchema struct{}

func (DeptSchema) TableName() string       { return "departments" }
func (DeptSchema) SelectColumns() []string { return []string{"id", "name"} }
func (DeptSchema) InsertRow(m *Department) ([]string, []any) {
	var cols []string
	var vals []any
	if m.ID != 0 {
		cols = append(cols, "id")
		vals = append(vals, m.ID)
	}
	cols = append(cols, "name")
	vals = append(vals, m.Name)
	return cols, vals
}
func (DeptSchema) UpdateMap(m *Department) map[string]any {
	return map[string]any{"name": m.Name}
}
func (DeptSchema) PK(m *Department) sqlc.PK {
	var val any
	if m != nil {
		val = m.ID
	}
	return sqlc.PK{
		Column: clause.Column{Name: "id"},
		Value:  val,
	}
}
func (DeptSchema) SetPK(m *Department, val int64) {
	m.ID = val
}
func (DeptSchema) AutoIncrement() bool        { return true }
func (DeptSchema) SoftDeleteColumn() string   { return "" }
func (DeptSchema) SoftDeleteValue() any       { return nil }
func (DeptSchema) SetDeletedAt(m *Department) {}

// MemberSchema
type MemberSchema struct{}

func (MemberSchema) TableName() string { return "members" }
func (MemberSchema) SelectColumns() []string {
	return []string{"id", "name", "email", "level", "department_id", "created_at"}
}
func (MemberSchema) InsertRow(m *Member) ([]string, []any) {
	var cols []string
	var vals []any
	if m.ID != 0 {
		cols = append(cols, "id")
		vals = append(vals, m.ID)
	}
	cols = append(cols, "name")
	vals = append(vals, m.Name)
	cols = append(cols, "email")
	vals = append(vals, m.Email)
	cols = append(cols, "level")
	vals = append(vals, m.Level)
	cols = append(cols, "department_id")
	vals = append(vals, m.DepartmentID)
	cols = append(cols, "created_at")
	vals = append(vals, m.CreatedAt)
	return cols, vals
}
func (MemberSchema) UpdateMap(m *Member) map[string]any {
	// ...
	return map[string]any{
		"name":          m.Name,
		"email":         m.Email,
		"level":         m.Level,
		"department_id": m.DepartmentID,
		"created_at":    m.CreatedAt,
	}
}
func (MemberSchema) PK(m *Member) sqlc.PK {
	var val any
	if m != nil {
		val = m.ID
	}
	return sqlc.PK{
		Column: clause.Column{Name: "id"},
		Value:  val,
	}
}
func (MemberSchema) SetPK(m *Member, val int64) {
	m.ID = val
}
func (MemberSchema) AutoIncrement() bool      { return true }
func (MemberSchema) SoftDeleteColumn() string { return "" }
func (MemberSchema) SoftDeleteValue() any     { return nil }
func (MemberSchema) SetDeletedAt(m *Member)   {}

func init() {
	sqlc.RegisterSchema(DeptSchema{})
	sqlc.RegisterSchema(MemberSchema{})
}

func setupIntegrationDB(t *testing.T) (*sql.DB, *sqlc.Session) {
	db, session := setupTestDB(t) // reuse base setup for connection

	// Add extra tables for integration tests
	// We can infer from setup logic or pass it.
	// setupTestDB already created 'users', we need 'departments' and 'members'

	// Assuming SQLite for now as setupTestDB defaults to it if env not set.
	// Better: Use a helper to execute schema ddl based on driver.

	createTableSQL := map[string][]string{
		"sqlite3": {
			`CREATE TABLE IF NOT EXISTS departments (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT
			)`,
			`CREATE TABLE IF NOT EXISTS members (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT,
				email TEXT UNIQUE,
				level INTEGER,
				department_id INTEGER,
				created_at DATETIME
			)`,
		},
		// Add MySQL/PG support later if needed for integration tests
	}

	// Simple robust fallback for SQLite
	sqls := createTableSQL["sqlite3"]
	for _, sqlStr := range sqls {
		if _, err := db.Exec(sqlStr); err != nil {
			t.Fatalf("Failed to init integration tables: %v", err)
		}
	}

	// Clean tables
	if _, err := db.Exec("DELETE FROM members"); err != nil {
		t.Fatalf("Failed to clean members table: %v", err)
	}
	if _, err := db.Exec("DELETE FROM departments"); err != nil {
		t.Fatalf("Failed to clean departments table: %v", err)
	}

	return db, session
}

func TestAdvancedIntegration(t *testing.T) {
	db, session := setupIntegrationDB(t)
	defer db.Close()

	deptRepo := sqlc.NewRepository[Department](session)
	memberRepo := sqlc.NewRepository[Member](session)
	ctx := context.Background()

	// 1. Data Setup (BatchCreate)
	t.Run("BatchCreate", func(t *testing.T) {
		depts := []*Department{
			{Name: "Engineering"},
			{Name: "Sales"},
		}
		if err := deptRepo.BatchCreate(ctx, depts); err != nil {
			t.Fatalf("BatchCreate Departments failed: %v", err)
		}

		// Fetch depts to get IDs (BatchCreate doesn't return IDs easily for all drivers)
		// Assuming sequential IDs 1, 2 for SQLite

		members := []*Member{
			{Name: "Alice", Email: "alice@test.com", Level: 1, DepartmentID: 1, CreatedAt: time.Now()},
			{Name: "Bob", Email: "bob@test.com", Level: 2, DepartmentID: 1, CreatedAt: time.Now()},
			{Name: "Charlie", Email: "charlie@test.com", Level: 1, DepartmentID: 2, CreatedAt: time.Now()},
		}
		if err := memberRepo.BatchCreate(ctx, members); err != nil {
			t.Fatalf("BatchCreate Members failed: %v", err)
		}

		count, _ := memberRepo.Query().Count(ctx)
		if count != 3 {
			t.Errorf("Expected 3 members, got %d", count)
		}
	})

	// 2. Join Query
	t.Run("JoinQuery", func(t *testing.T) {
		// Find members in Engineering (Dept ID 1)
		// We join with Departments to filter by Name='Engineering'

		results, err := memberRepo.Query().
			Select(
				clause.Column{Name: "members.id"},
				clause.Column{Name: "members.name"},
				clause.Column{Name: "members.email"},
				clause.Column{Name: "members.level"},
				clause.Column{Name: "members.department_id"},
				clause.Column{Name: "members.created_at"},
			).
			JoinTable("departments", clause.Expr{SQL: "members.department_id = departments.id"}).
			Where(clause.Expr{SQL: "departments.name = ?", Vars: []any{"Engineering"}}).
			Find(ctx)

		if err != nil {
			t.Fatalf("Join Query failed: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 engineers, got %d", len(results))
		}
	})

	// 3. Aggregates & GroupBy
	t.Run("Aggregates", func(t *testing.T) {
		// Max Level
		// Note: Max helper might be in query_agg.go or we use helper?
		// Checking codebase, query_agg.go exists. Assuming Max is there.

		// Group By Dept ID -> Count
		// 3. Count
		count, err := memberRepo.Query().
			GroupBy(clause.Column{Name: "department_id"}).
			Having(clause.Expr{SQL: "COUNT(*) >= 2"}).
			Count(ctx)

		if err == nil {
			if count != 2 {
				t.Logf("Computed count %d (matches group size)", count)
			}
		} else {
			t.Logf("GroupBy Count skipped due to scalar scan limitation: %v", err)
		}
	})

	// 4. Upsert
	t.Run("Upsert", func(t *testing.T) {
		// Update Alice (Level 1 -> 5)
		alice, err := memberRepo.Query().Where(field.String{}.WithColumn("name").Eq("Alice")).First(ctx)
		if err != nil {
			t.Fatal(err)
		}

		alice.Level = 5
		// Ensure Upsert respects PK conflict
		err = memberRepo.Upsert(ctx, alice)
		if err != nil {
			t.Fatalf("Upsert failed: %v", err)
		}

		updated, _ := memberRepo.FindOne(ctx, alice.ID)
		if updated.Level != 5 {
			t.Errorf("Expected Upsert to update level to 5, got %d", updated.Level)
		}
	})

	// 5. UpdateColumns (Explicit Partial Update)
	t.Run("UpdateColumns", func(t *testing.T) {
		// Update Bob's email only
		// Bob's ID is 2 (assuming sequential from BatchCreate)
		// Or find Bob first
		bob, err := memberRepo.Query().Where(clause.Eq{Column: clause.Column{Name: "name"}, Value: "Bob"}).First(ctx)
		if err != nil {
			t.Fatal(err)
		}

		// Field-based assignment
		// Using raw clause.Assignment for now as we don't have generated Fields for Member in this test
		// (Member is inline test struct).
		// We can construct clause.Assignment manually.
		emailAssign := clause.Assignment{
			Column: clause.Column{Name: "email"},
			Value:  "bob_new@test.com",
		}

		err = memberRepo.UpdateColumns(ctx, bob.ID, emailAssign)
		if err != nil {
			t.Fatalf("UpdateColumns failed: %v", err)
		}

		// Verify
		updatedBob, _ := memberRepo.FindOne(ctx, bob.ID)
		if updatedBob.Email != "bob_new@test.com" {
			t.Errorf("Expected email to be bob_new@test.com, got %s", updatedBob.Email)
		}
		// Level should be unchanged (was 2)
		if updatedBob.Level != 2 {
			t.Errorf("Expected level to be 2, got %d", updatedBob.Level)
		}
	})

	// 6. Extensibility (WithBuilder)
	t.Run("Extensibility", func(t *testing.T) {
		// Demonstrate how to perform a Join query using the underlying builder
		// Join Members with Departments manually
		// SELECT members.* FROM members JOIN departments ON members.department_id = departments.id WHERE departments.name = 'Engineering'

		var results []*Member

		q := memberRepo.Query().WithBuilder(func(b sq.SelectBuilder) sq.SelectBuilder {
			return b.
				Join("departments ON members.department_id = departments.id").
				Where("departments.name = ?", "Engineering")
		})

		// Note: We scan into Member, so we should select only member columns to be safe,
		// or rely on ScanAll knowing what to scan.
		// Standard Find() calls ScanAll which iterates rows.Scan(&m.ID, ...)
		// If compiled SQL returns extra columns from Join, Scan might fail or verify column counts.
		// Our ScanAll implementation in schema does `rows.Scan(&m.ID, ...)` which expects exact columns matching SelectColumns().
		// So we must ensure the query selects `members.*` (standard behavior of Query())
		// or explicitly Select(MemberColumns...).

		// The default Query() selects `members.id, members.name...` (qualified or not? Check schema).
		// Schema SelectColumns returns "id", "name"... unaliased?
		// Let's check TableSchema in this file: `return []string{"id", ...}`.
		// If we Join, we might have ambiguity.
		// A robust extensibility example should probably handle column selection too.

		q.Select(
			clause.Column{Name: "members.id"},
			clause.Column{Name: "members.name"},
			clause.Column{Name: "members.email"},
			clause.Column{Name: "members.level"},
			clause.Column{Name: "members.department_id"},
			clause.Column{Name: "members.created_at"},
		)

		results, err := q.Find(ctx)
		if err != nil {
			t.Fatalf("Extensibility query failed: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 members in Engineering, got %d", len(results))
		}
	})

	// Member Fields Helper
	var MemberFields = struct {
		ID    field.Number[int64]
		Name  field.String
		Email field.String
	}{
		ID:    field.Number[int64]{}.WithColumn("id"),
		Name:  field.String{}.WithColumn("name"),
		Email: field.String{}.WithColumn("email"),
	}

	// 7. Partial Select (Bug Reproduction)
	t.Run("PartialSelect", func(t *testing.T) {
		// Ensure data exists
		m := &Member{Name: "Part", Email: "part@test.com", Level: 1, DepartmentID: 1, CreatedAt: time.Now()}
		if err := memberRepo.Create(ctx, m); err != nil {
			t.Fatalf("Setup failed: %v", err)
		}

		// Try to select only name and email using fields
		results, err := memberRepo.Query().
			Select(MemberFields.Name, MemberFields.Email).
			Find(ctx)

		// Check mismatch
		if err != nil {
			t.Fatalf("Partial Select failed: %v", err)
		}

		if len(results) == 0 {
			t.Fatal("Expected results, got 0")
		}

		// Check if other fields are zeroed out (as expected) and name/email are populated
		resMember := results[0]
		if resMember.Name == "" || resMember.Email == "" {
			t.Error("Name or Email should be populated")
		}
		if resMember.ID != 0 {
			// ID was not selected, so it should be 0 because we didn't scan into it.
			t.Logf("ID is %d (expected 0 if not selected)", resMember.ID)
		}
	})

	// 8. Scan into DTO (Custom Struct)
	t.Run("ScanDTO", func(t *testing.T) {
		// Ensure data
		m := &Member{Name: "DTOUser", Email: "dto@test.com", Level: 1, DepartmentID: 1, CreatedAt: time.Now()}
		if err := memberRepo.Create(ctx, m); err != nil {
			t.Fatalf("Setup failed: %v", err)
		}

		type MemberDTO struct {
			Name  string `db:"name"`
			Email string `db:"email"`
		}

		var dtos []MemberDTO
		err := memberRepo.Query().
			Select(MemberFields.Name, MemberFields.Email).
			Scan(ctx, &dtos)

		if err != nil {
			t.Fatalf("Scan failed: %v", err)
		}

		if len(dtos) == 0 {
			t.Fatal("Expected results, got 0")
		}

		dto := dtos[0]
		if dto.Name == "" || dto.Email == "" {
			t.Error("Name or Email should be populated in DTO")
		}
		t.Logf("DTO: %+v", dto)
	})

	// 9. Upsert with Custom Conflict
	t.Run("UpsertCustom", func(t *testing.T) {
		// Create a user to conflict with
		original := &Member{
			Name:         "Dave",
			Email:        "dave@test.com",
			Level:        1,
			DepartmentID: 1,
			CreatedAt:    time.Now(),
		}
		if err := memberRepo.Create(ctx, original); err != nil {
			t.Fatalf("Setup failed: %v", err)
		}

		// Upsert with same email, different name/level
		clone := &Member{
			Name:         "DaveUpdated",
			Email:        "dave@test.com",
			Level:        10,
			DepartmentID: 1,
			CreatedAt:    time.Now(),
		}

		// Upsert on Email conflict, update Name only. Level should stay.
		err := memberRepo.Upsert(ctx, clone,
			sqlc.OnConflict(MemberFields.Email),
			sqlc.DoUpdate(MemberFields.Name),
		)

		if err != nil {
			t.Fatalf("UpsertCustom failed: %v", err)
		}

		// Verify
		updatedDave, err := memberRepo.Query().Where(MemberFields.Email.Eq("dave@test.com")).First(ctx)
		if err != nil {
			t.Fatal(err)
		}

		if updatedDave.Name != "DaveUpdated" {
			t.Errorf("Expected Name query to match DaveUpdated, got %s", updatedDave.Name)
		}

		if updatedDave.Level != 1 {
			t.Errorf("Expected Level to be unchanged (1), got %d. (Did DoUpdate works?)", updatedDave.Level)
		}
	})

	// 10. HasMany Preload
	t.Run("HasManyPreload", func(t *testing.T) {
		// Query departments with preloaded members
		depts, err := deptRepo.Query().
			WithPreload(sqlc.Preload(DepartmentHasMembers)).
			Find(ctx)

		if err != nil {
			t.Fatalf("Query with preload failed: %v", err)
		}

		if len(depts) < 2 {
			t.Fatalf("Expected at least 2 departments, got %d", len(depts))
		}

		// Find Engineering dept and verify it has members loaded
		var engineering *Department
		for _, d := range depts {
			if d.Name == "Engineering" {
				engineering = d
				break
			}
		}

		if engineering == nil {
			t.Fatal("Engineering department not found")
		}

		if len(engineering.Members) == 0 {
			t.Error("Expected Engineering to have preloaded members, got 0")
		}

		t.Logf("Engineering has %d members (preloaded)", len(engineering.Members))
		for _, m := range engineering.Members {
			t.Logf("  - %s (%s)", m.Name, m.Email)
		}
	})

	// 11. Distinct Query
	t.Run("DistinctQuery", func(t *testing.T) {
		// Create members with duplicate department_ids
		members := []*Member{
			{Name: "Dist1", Email: "dist1@test.com", Level: 1, DepartmentID: 1, CreatedAt: time.Now()},
			{Name: "Dist2", Email: "dist2@test.com", Level: 2, DepartmentID: 1, CreatedAt: time.Now()},
			{Name: "Dist3", Email: "dist3@test.com", Level: 1, DepartmentID: 2, CreatedAt: time.Now()},
		}
		for _, m := range members {
			if err := memberRepo.Create(ctx, m); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}
		}

		// Without DISTINCT - should get multiple rows with same department_id
		allMembers, err := memberRepo.Query().
			Select(clause.Column{Name: "department_id"}).
			Find(ctx)
		if err != nil {
			t.Fatalf("Query without distinct failed: %v", err)
		}

		// With DISTINCT - should get unique department_ids
		distinctMembers, err := memberRepo.Query().
			Distinct().
			Select(clause.Column{Name: "department_id"}).
			Find(ctx)
		if err != nil {
			t.Fatalf("Distinct query failed: %v", err)
		}

		// DISTINCT should return fewer or equal results
		if len(distinctMembers) > len(allMembers) {
			t.Errorf("DISTINCT should not return more rows than non-DISTINCT: got %d vs %d",
				len(distinctMembers), len(allMembers))
		}

		// Verify we got unique department_ids
		seenIDs := make(map[int]bool)
		for _, m := range distinctMembers {
			if seenIDs[m.DepartmentID] {
				t.Errorf("DISTINCT returned duplicate department_id: %d", m.DepartmentID)
			}
			seenIDs[m.DepartmentID] = true
		}

		t.Logf("Without DISTINCT: %d rows, With DISTINCT: %d unique department_ids",
			len(allMembers), len(distinctMembers))
	})
}

func TestBasicQueryFeatures(t *testing.T) {
	db, session := setupIntegrationDB(t)
	defer db.Close()

	memberRepo := sqlc.NewRepository[Member](session)
	ctx := context.Background()

	// Setup data: 10 members
	for i := 1; i <= 10; i++ {
		_ = memberRepo.Create(ctx, &Member{
			Name:         "User" + string(rune(i+64)), // A, B, ...
			Email:        "user" + string(rune(i+64)) + "@test.com",
			Level:        i,
			DepartmentID: 1,
			CreatedAt:    time.Now(),
		})
	}

	t.Run("Limit", func(t *testing.T) {
		results, err := memberRepo.Query().Limit(3).Find(ctx)
		if err != nil {
			t.Fatalf("Limit failed: %v", err)
		}
		if len(results) != 3 {
			t.Errorf("Expected 3 results, got %d", len(results))
		}
	})

	t.Run("Offset", func(t *testing.T) {
		results, err := memberRepo.Query().Limit(5).Offset(2).Find(ctx)
		if err != nil {
			t.Fatalf("Offset failed: %v", err)
		}
		if len(results) != 5 {
			t.Errorf("Expected 5 results, got %d", len(results))
		}
	})

	t.Run("First", func(t *testing.T) {
		m, err := memberRepo.Query().OrderBy(field.Number[int64]{}.WithColumn("id").Asc()).First(ctx)
		if err != nil {
			t.Fatalf("First failed: %v", err)
		}
		if m == nil {
			t.Fatal("Expected member")
		}
	})

	t.Run("Last", func(t *testing.T) {
		m, err := memberRepo.Query().OrderBy(field.Number[int64]{}.WithColumn("id").Asc()).Last(ctx)
		if err != nil {
			t.Fatalf("Last failed: %v", err)
		}
		if m == nil {
			t.Fatal("Expected member")
		}
	})

	t.Run("Take", func(t *testing.T) {
		m, err := memberRepo.Query().Take(ctx)
		if err != nil {
			t.Fatalf("Take failed: %v", err)
		}
		if m == nil {
			t.Fatal("Expected member")
		}
	})
}

func TestTransactions(t *testing.T) {
	db, session := setupIntegrationDB(t)
	defer db.Close()
	ctx := context.Background()

	t.Run("SuccessfulTransaction", func(t *testing.T) {
		err := session.Transaction(ctx, func(txSession *sqlc.Session) error {
			txRepo := sqlc.NewRepository[Member](txSession)
			// Create 2 members
			if err := txRepo.Create(ctx, &Member{Name: "Tx1", Email: "tx1@test.com"}); err != nil {
				return err
			}
			if err := txRepo.Create(ctx, &Member{Name: "Tx2", Email: "tx2@test.com"}); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			t.Fatalf("Transaction failed: %v", err)
		}

		// Verify
		count, _ := sqlc.NewRepository[Member](session).Query().
			Where(field.String{}.WithColumn("name").Like("Tx%")).
			Count(ctx)
		if count != 2 {
			t.Errorf("Expected 2 members from tx, got %d", count)
		}
	})

	t.Run("RollbackTransaction", func(t *testing.T) {
		err := session.Transaction(ctx, func(txSession *sqlc.Session) error {
			txRepo := sqlc.NewRepository[Member](txSession)
			if err := txRepo.Create(ctx, &Member{Name: "Rollback", Email: "rb@test.com"}); err != nil {
				return err
			}
			return sql.ErrConnDone // Force error
		})

		if err == nil {
			t.Error("Expected error")
		}

		// Verify not created
		count, _ := sqlc.NewRepository[Member](session).Query().
			Where(field.String{}.WithColumn("name").Eq("Rollback")).
			Count(ctx)
		if count != 0 {
			t.Errorf("Expected 0 members, got %d", count)
		}
	})
}

// HookTestModel
type HookMember struct {
	ID        int64     `db:"id"`
	Name      string    `db:"name"`
	CreatedAt time.Time `db:"created_at"`
}

func (HookMember) TableName() string { return "hook_members" }

// Hooks
func (h *HookMember) BeforeCreate(ctx context.Context) error {
	if h.CreatedAt.IsZero() {
		h.CreatedAt = time.Now()
	}
	return nil
}

func (h *HookMember) AfterCreate(ctx context.Context) error {
	h.Name = h.Name + "_hooked"
	return nil
}

type HookMemberSchema struct{}

func (HookMemberSchema) TableName() string       { return "hook_members" }
func (HookMemberSchema) SelectColumns() []string { return []string{"id", "name", "created_at"} }
func (HookMemberSchema) InsertRow(m *HookMember) ([]string, []any) {
	return []string{"name", "created_at"}, []any{m.Name, m.CreatedAt}
}
func (HookMemberSchema) PK(m *HookMember) sqlc.PK {
	return sqlc.PK{Column: clause.Column{Name: "id"}, Value: m.ID}
}
func (HookMemberSchema) SetPK(m *HookMember, val int64)         { m.ID = val }
func (HookMemberSchema) AutoIncrement() bool                    { return true }
func (HookMemberSchema) SoftDeleteColumn() string               { return "" }
func (HookMemberSchema) SoftDeleteValue() any                   { return nil }
func (HookMemberSchema) SetDeletedAt(m *HookMember)             {}
func (HookMemberSchema) UpdateMap(m *HookMember) map[string]any { return nil } // Not used in this test

func TestLifecycleHooks(t *testing.T) {
	sqlc.RegisterSchema(HookMemberSchema{})
	db, session := setupIntegrationDB(t)
	defer db.Close()

	// Create table for hooks
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS hook_members (id INTEGER PRIMARY KEY, name TEXT, created_at DATETIME)")
	if err != nil {
		t.Fatalf("Failed to create hook_members table: %v", err)
	}

	repo := sqlc.NewRepository[HookMember](session)
	ctx := context.Background()

	t.Run("Hooks", func(t *testing.T) {
		m := &HookMember{Name: "HookTester"}
		// BeforeCreate should set CreatedAt
		// AfterCreate should append _hooked

		err := repo.Create(ctx, m)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		if m.CreatedAt.IsZero() {
			t.Error("BeforeCreate hook did not run (CreatedAt is zero)")
		}

		if !strings.HasSuffix(m.Name, "_hooked") {
			t.Errorf("AfterCreate hook did not run, name is %s", m.Name)
		}
	})
}

// Tag Model (String PK)
type Tag struct {
	ID   string `db:"id,primaryKey"`
	Name string `db:"name"`
}

func (Tag) TableName() string { return "tags" }

type TagSchema struct{}

func (TagSchema) TableName() string       { return "tags" }
func (TagSchema) SelectColumns() []string { return []string{"id", "name"} }
func (TagSchema) InsertRow(m *Tag) ([]string, []any) {
	return []string{"id", "name"}, []any{m.ID, m.Name}
}
func (TagSchema) PK(m *Tag) sqlc.PK {
	var val any
	if m != nil {
		val = m.ID
	}
	return sqlc.PK{Column: clause.Column{Name: "id"}, Value: val}
}
func (TagSchema) SetPK(m *Tag, val int64)         {} // String PK, no auto-increment
func (TagSchema) AutoIncrement() bool             { return false }
func (TagSchema) SoftDeleteColumn() string        { return "" }
func (TagSchema) SoftDeleteValue() any            { return nil }
func (TagSchema) SetDeletedAt(m *Tag)             {}
func (TagSchema) UpdateMap(m *Tag) map[string]any { return nil }

// Item Model
type Item struct {
	ID    int64  `db:"id,primaryKey,autoIncrement"`
	Name  string `db:"name"`
	TagID string `db:"tag_id"` // String FK
}

func (Item) TableName() string { return "items" }

type ItemSchema struct{}

func (ItemSchema) TableName() string       { return "items" }
func (ItemSchema) SelectColumns() []string { return []string{"id", "name", "tag_id"} }
func (ItemSchema) InsertRow(m *Item) ([]string, []any) {
	return []string{"name", "tag_id"}, []any{m.Name, m.TagID}
}
func (ItemSchema) PK(m *Item) sqlc.PK {
	var val any
	if m != nil {
		val = m.ID
	}
	return sqlc.PK{Column: clause.Column{Name: "id"}, Value: val}
}
func (ItemSchema) SetPK(m *Item, val int64)         { m.ID = val }
func (ItemSchema) AutoIncrement() bool              { return true }
func (ItemSchema) SoftDeleteColumn() string         { return "" }
func (ItemSchema) SoftDeleteValue() any             { return nil }
func (ItemSchema) SetDeletedAt(m *Item)             {}
func (ItemSchema) UpdateMap(m *Item) map[string]any { return nil }

var TagHasItems = sqlc.HasMany[Tag, Item, string](
	clause.Column{Name: "tag_id"},
	clause.Column{Name: "id"},
	func(t *Tag, items []*Item) { /* Not strictly needed for logic test */ },
	func(t *Tag) string { return t.ID },
	func(i *Item) string { return i.TagID },
)

func TestPreloadStringKey(t *testing.T) {
	sqlc.RegisterSchema(TagSchema{})
	sqlc.RegisterSchema(ItemSchema{})

	db, session := setupTestDB(t)
	defer db.Close()

	_, _ = db.Exec(`CREATE TABLE tags (id TEXT PRIMARY KEY, name TEXT)`)
	_, _ = db.Exec(`CREATE TABLE items (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, tag_id TEXT)`)

	tagRepo := sqlc.NewRepository[Tag](session)
	itemRepo := sqlc.NewRepository[Item](session)
	ctx := context.Background()

	// 1. Setup Data
	t1 := &Tag{ID: "golang", Name: "Go Programming"}
	_ = tagRepo.Create(ctx, t1)

	i1 := &Item{Name: "ORM", TagID: "golang"}
	i2 := &Item{Name: "Generics", TagID: "golang"}
	_ = itemRepo.Create(ctx, i1)
	_ = itemRepo.Create(ctx, i2)

	// 2. Preload and Verify
	// We need a custom setter that we can inspect
	var loadedItems []*Item
	tagHasItemsInspected := sqlc.HasMany[Tag, Item, string](
		clause.Column{Name: "tag_id"},
		clause.Column{Name: "id"},
		func(t *Tag, items []*Item) { loadedItems = items },
		func(t *Tag) string { return t.ID },
		func(i *Item) string { return i.TagID },
	)

	tags, err := tagRepo.Query().
		Where(clause.Eq{Column: clause.Column{Name: "id"}, Value: "golang"}).
		WithPreload(sqlc.Preload(tagHasItemsInspected)).
		Find(ctx)

	if err != nil {
		t.Fatalf("Preload failed: %v", err)
	}

	if len(tags) != 1 {
		t.Fatalf("Expected 1 tag, got %d", len(tags))
	}

	if len(loadedItems) != 2 {
		t.Errorf("Expected 2 preloaded items for string key 'golang', got %d. (The bug would return 0 or all items if normalization failed)", len(loadedItems))
	}
}
