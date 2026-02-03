package field_test

import (
	"testing"

	"github.com/arllen133/sqlc/clause"
	"github.com/arllen133/sqlc/field"
)

func TestStringField(t *testing.T) {
	username := field.String{}.WithColumn("username")

	// Test Eq
	expr := username.Eq("alice")
	sql, args := expr.Build()
	if sql != "username = ?" {
		t.Errorf("Expected 'username = ?', got '%s'", sql)
	}
	if len(args) != 1 || args[0] != "alice" {
		t.Errorf("Expected args ['alice'], got %v", args)
	}

	// Test Like
	expr = username.Like("%alice%")
	sql, _ = expr.Build()
	if sql != "username LIKE ?" {
		t.Errorf("Expected 'username LIKE ?', got '%s'", sql)
	}

	// Test In
	expr = username.In("alice", "bob", "charlie")
	sql, args = expr.Build()
	expected := "username IN (?, ?, ?)"
	if sql != expected {
		t.Errorf("Expected '%s', got '%s'", expected, sql)
	}
	if len(args) != 3 {
		t.Errorf("Expected 3 args, got %d", len(args))
	}
}

func TestNumberField(t *testing.T) {
	age := field.Number[int]{}.WithColumn("age")

	// Test Gt
	expr := age.Gt(18)
	sql, args := expr.Build()
	if sql != "age > ?" {
		t.Errorf("Expected 'age > ?', got '%s'", sql)
	}
	if len(args) != 1 || args[0] != 18 {
		t.Errorf("Expected args [18], got %v", args)
	}

	// Test Between
	expr = age.Between(18, 65)
	sql, args = expr.Build()
	expected := "age BETWEEN ? AND ?"
	if sql != expected {
		t.Errorf("Expected '%s', got '%s'", expected, sql)
	}
	if len(args) != 2 || args[0] != 18 || args[1] != 65 {
		t.Errorf("Expected args [18, 65], got %v", args)
	}
}

func TestBoolField(t *testing.T) {
	active := field.Bool{}.WithColumn("is_active")

	// Test IsTrue
	expr := active.IsTrue()
	sql, args := expr.Build()
	if sql != "is_active = ?" {
		t.Errorf("Expected 'is_active = ?', got '%s'", sql)
	}
	if len(args) != 1 || args[0] != true {
		t.Errorf("Expected args [true], got %v", args)
	}

	// Test IsFalse
	expr = active.IsFalse()
	sql, args = expr.Build()
	if sql != "is_active = ?" {
		t.Errorf("Expected 'is_active = ?', got '%s'", sql)
	}
	if len(args) != 1 || args[0] != false {
		t.Errorf("Expected args [false], got %v", args)
	}
}

func TestFieldWithTable(t *testing.T) {
	email := field.String{}.WithTable("users").WithColumn("email")

	expr := email.Eq("test@example.com")
	sql, _ := expr.Build()

	expected := "users.email = ?"
	if sql != expected {
		t.Errorf("Expected '%s', got '%s'", expected, sql)
	}
}

func TestFieldIsNull(t *testing.T) {
	email := field.String{}.WithColumn("email")

	// Test IsNull
	expr := email.IsNull()
	sql, _ := expr.Build()
	if sql != "email IS NULL" {
		t.Errorf("Expected 'email IS NULL', got '%s'", sql)
	}

	// Test IsNotNull
	expr = email.IsNotNull()
	sql, _ = expr.Build()
	if sql != "email IS NOT NULL" {
		t.Errorf("Expected 'email IS NOT NULL', got '%s'", sql)
	}
}

func TestComplexExpression(t *testing.T) {
	age := field.Number[int]{}.WithColumn("age")
	status := field.String{}.WithColumn("status")
	role := field.String{}.WithColumn("role")

	// Build complex expression: (age > 18 AND status = 'active') OR role = 'admin'
	expr := clause.Or{
		clause.And{
			age.Gt(18),
			status.Eq("active"),
		},
		role.Eq("admin"),
	}

	sql, args := expr.Build()

	expected := "((age > ?) AND (status = ?)) OR (role = ?)"
	if sql != expected {
		t.Errorf("Expected '%s', got '%s'", expected, sql)
	}
	if len(args) != 3 {
		t.Errorf("Expected 3 args, got %d: %v", len(args), args)
	}
	if args[0] != 18 || args[1] != "active" || args[2] != "admin" {
		t.Errorf("Args mismatch, got %v", args)
	}
}

func TestOrderBy(t *testing.T) {
	createdAt := field.String{}.WithColumn("created_at")

	// Test Asc
	orderAsc := createdAt.Asc()
	if sql, _ := orderAsc.Build(); sql != "created_at" {
		t.Errorf("Expected 'created_at', got '%s'", sql)
	}

	// Test Desc
	orderDesc := createdAt.Desc()
	if sql, _ := orderDesc.Build(); sql != "created_at DESC" {
		t.Errorf("Expected 'created_at DESC', got '%s'", sql)
	}
}

func TestAssignment(t *testing.T) {
	email := field.String{}.WithColumn("email")

	assign := email.Set("new@example.com")
	sql, args := assign.Build()

	expected := "email = ?"
	if sql != expected {
		t.Errorf("Expected '%s', got '%s'", expected, sql)
	}
	if len(args) != 1 || args[0] != "new@example.com" {
		t.Errorf("Expected args ['new@example.com'], got %v", args)
	}
}
