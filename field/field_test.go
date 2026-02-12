package field_test

import (
	"testing"
	"time"

	"github.com/arllen133/sqlc/clause"
	"github.com/arllen133/sqlc/field"
)

// ============== String Field Tests ==============

func TestStringField(t *testing.T) {
	username := field.String{}.WithColumn("username")

	t.Run("Eq", func(t *testing.T) {
		expr := username.Eq("alice")
		sql, args, _ := expr.Build()
		if sql != "username = ?" {
			t.Errorf("Expected 'username = ?', got '%s'", sql)
		}
		if len(args) != 1 || args[0] != "alice" {
			t.Errorf("Expected args ['alice'], got %v", args)
		}
	})

	t.Run("Neq", func(t *testing.T) {
		expr := username.Neq("alice")
		sql, args, _ := expr.Build()
		if sql != "username <> ?" {
			t.Errorf("Expected 'username <> ?', got '%s'", sql)
		}
		if len(args) != 1 || args[0] != "alice" {
			t.Errorf("Expected args ['alice'], got %v", args)
		}
	})

	t.Run("Like", func(t *testing.T) {
		expr := username.Like("%alice%")
		sql, args, _ := expr.Build()
		if sql != "username LIKE ?" {
			t.Errorf("Expected 'username LIKE ?', got '%s'", sql)
		}
		if len(args) != 1 || args[0] != "%alice%" {
			t.Errorf("Expected args ['%%alice%%'], got %v", args)
		}
	})

	t.Run("NotLike", func(t *testing.T) {
		expr := username.NotLike("%spam%")
		sql, args, _ := expr.Build()
		if sql != "username NOT LIKE ?" {
			t.Errorf("Expected 'username NOT LIKE ?', got '%s'", sql)
		}
		if len(args) != 1 || args[0] != "%spam%" {
			t.Errorf("Expected args ['%%spam%%'], got %v", args)
		}
	})

	t.Run("In", func(t *testing.T) {
		expr := username.In("alice", "bob", "charlie")
		sql, args, _ := expr.Build()
		expected := "username IN (?, ?, ?)"
		if sql != expected {
			t.Errorf("Expected '%s', got '%s'", expected, sql)
		}
		if len(args) != 3 {
			t.Errorf("Expected 3 args, got %d", len(args))
		}
	})

	t.Run("NotIn", func(t *testing.T) {
		expr := username.NotIn("alice", "bob")
		sql, _, _ := expr.Build()
		expected := "NOT (username IN (?, ?))"
		if sql != expected {
			t.Errorf("Expected '%s', got '%s'", expected, sql)
		}
	})

	t.Run("IsNull", func(t *testing.T) {
		expr := username.IsNull()
		sql, args, _ := expr.Build()
		if sql != "username IS NULL" {
			t.Errorf("Expected 'username IS NULL', got '%s'", sql)
		}
		if len(args) != 0 {
			t.Errorf("Expected no args, got %d", len(args))
		}
	})

	t.Run("IsNotNull", func(t *testing.T) {
		expr := username.IsNotNull()
		sql, args, _ := expr.Build()
		if sql != "username IS NOT NULL" {
			t.Errorf("Expected 'username IS NOT NULL', got '%s'", sql)
		}
		if len(args) != 0 {
			t.Errorf("Expected no args, got %d", len(args))
		}
	})

	t.Run("Set", func(t *testing.T) {
		assign := username.Set("new_name")
		sql, args, _ := assign.Build()
		if sql != "username = ?" {
			t.Errorf("Expected 'username = ?', got '%s'", sql)
		}
		if len(args) != 1 || args[0] != "new_name" {
			t.Errorf("Expected args ['new_name'], got %v", args)
		}
	})

	t.Run("Asc", func(t *testing.T) {
		order := username.Asc()
		sql, _, _ := order.Build()
		if sql != "username" {
			t.Errorf("Expected 'username', got '%s'", sql)
		}
	})

	t.Run("Desc", func(t *testing.T) {
		order := username.Desc()
		sql, _, _ := order.Build()
		if sql != "username DESC" {
			t.Errorf("Expected 'username DESC', got '%s'", sql)
		}
	})

	t.Run("InExpr", func(t *testing.T) {
		subquery := clause.Expr{SQL: "SELECT username FROM banned_users"}
		expr := username.InExpr(subquery)
		sql, args, _ := expr.Build()
		expected := "username IN (SELECT username FROM banned_users)"
		if sql != expected {
			t.Errorf("Expected '%s', got '%s'", expected, sql)
		}
		if len(args) != 0 {
			t.Errorf("Expected no args, got %d", len(args))
		}
	})

	t.Run("NotInExpr", func(t *testing.T) {
		subquery := clause.Expr{SQL: "SELECT username FROM banned_users"}
		expr := username.NotInExpr(subquery)
		sql, _, _ := expr.Build()
		expected := "username NOT IN (SELECT username FROM banned_users)"
		if sql != expected {
			t.Errorf("Expected '%s', got '%s'", expected, sql)
		}
	})
}

// ============== Number Field Tests ==============

func TestNumberField(t *testing.T) {
	age := field.Number[int]{}.WithColumn("age")

	t.Run("Eq", func(t *testing.T) {
		expr := age.Eq(25)
		sql, args, _ := expr.Build()
		if sql != "age = ?" {
			t.Errorf("Expected 'age = ?', got '%s'", sql)
		}
		if args[0] != 25 {
			t.Errorf("Expected 25, got %v", args[0])
		}
	})

	t.Run("Neq", func(t *testing.T) {
		expr := age.Neq(25)
		sql, _, _ := expr.Build()
		if sql != "age <> ?" {
			t.Errorf("Expected 'age <> ?', got '%s'", sql)
		}
	})

	t.Run("Gt", func(t *testing.T) {
		expr := age.Gt(18)
		sql, args, _ := expr.Build()
		if sql != "age > ?" {
			t.Errorf("Expected 'age > ?', got '%s'", sql)
		}
		if args[0] != 18 {
			t.Errorf("Expected 18, got %v", args[0])
		}
	})

	t.Run("Gte", func(t *testing.T) {
		expr := age.Gte(18)
		sql, _, _ := expr.Build()
		if sql != "age >= ?" {
			t.Errorf("Expected 'age >= ?', got '%s'", sql)
		}
	})

	t.Run("Lt", func(t *testing.T) {
		expr := age.Lt(65)
		sql, _, _ := expr.Build()
		if sql != "age < ?" {
			t.Errorf("Expected 'age < ?', got '%s'", sql)
		}
	})

	t.Run("Lte", func(t *testing.T) {
		expr := age.Lte(65)
		sql, _, _ := expr.Build()
		if sql != "age <= ?" {
			t.Errorf("Expected 'age <= ?', got '%s'", sql)
		}
	})

	t.Run("Between", func(t *testing.T) {
		expr := age.Between(18, 65)
		sql, args, _ := expr.Build()
		if sql != "age BETWEEN ? AND ?" {
			t.Errorf("Expected 'age BETWEEN ? AND ?', got '%s'", sql)
		}
		if args[0] != 18 || args[1] != 65 {
			t.Errorf("Expected [18, 65], got %v", args)
		}
	})

	t.Run("In", func(t *testing.T) {
		expr := age.In(18, 21, 25, 30)
		sql, args, _ := expr.Build()
		if sql != "age IN (?, ?, ?, ?)" {
			t.Errorf("Expected 'age IN (?, ?, ?, ?)', got '%s'", sql)
		}
		if len(args) != 4 {
			t.Errorf("Expected 4 args, got %d", len(args))
		}
	})

	t.Run("NotIn", func(t *testing.T) {
		expr := age.NotIn(18, 21)
		sql, _, _ := expr.Build()
		if sql != "NOT (age IN (?, ?))" {
			t.Errorf("Expected 'NOT (age IN (?, ?))', got '%s'", sql)
		}
	})

	t.Run("IsNull/IsNotNull", func(t *testing.T) {
		expr := age.IsNull()
		sql, _, _ := expr.Build()
		if sql != "age IS NULL" {
			t.Errorf("Expected 'age IS NULL', got '%s'", sql)
		}

		expr = age.IsNotNull()
		sql, _, _ = expr.Build()
		if sql != "age IS NOT NULL" {
			t.Errorf("Expected 'age IS NOT NULL', got '%s'", sql)
		}
	})

	t.Run("Set", func(t *testing.T) {
		assign := age.Set(30)
		sql, args, _ := assign.Build()
		if sql != "age = ?" {
			t.Errorf("Expected 'age = ?', got '%s'", sql)
		}
		if args[0] != 30 {
			t.Errorf("Expected 30, got %v", args[0])
		}
	})

	t.Run("Asc/Desc", func(t *testing.T) {
		if sql, _, _ := age.Asc().Build(); sql != "age" {
			t.Errorf("Expected 'age', got '%s'", sql)
		}
		if sql, _, _ := age.Desc().Build(); sql != "age DESC" {
			t.Errorf("Expected 'age DESC', got '%s'", sql)
		}
	})

	t.Run("InExpr/NotInExpr", func(t *testing.T) {
		subquery := clause.Expr{SQL: "SELECT age FROM restricted_ages"}
		expr := age.InExpr(subquery)
		sql, _, _ := expr.Build()
		if sql != "age IN (SELECT age FROM restricted_ages)" {
			t.Errorf("Unexpected SQL: %s", sql)
		}

		expr = age.NotInExpr(subquery)
		sql, _, _ = expr.Build()
		if sql != "age NOT IN (SELECT age FROM restricted_ages)" {
			t.Errorf("Unexpected SQL: %s", sql)
		}
	})
}

func TestNumberField_Float64(t *testing.T) {
	price := field.Number[float64]{}.WithColumn("price")

	t.Run("FloatOperations", func(t *testing.T) {
		expr := price.Gt(99.99)
		sql, args, _ := expr.Build()
		if sql != "price > ?" {
			t.Errorf("Expected 'price > ?', got '%s'", sql)
		}
		if args[0] != 99.99 {
			t.Errorf("Expected 99.99, got %v", args[0])
		}
	})

	t.Run("FloatBetween", func(t *testing.T) {
		expr := price.Between(10.0, 100.0)
		sql, args, _ := expr.Build()
		if sql != "price BETWEEN ? AND ?" {
			t.Errorf("Expected 'price BETWEEN ? AND ?', got '%s'", sql)
		}
		if args[0] != 10.0 || args[1] != 100.0 {
			t.Errorf("Expected [10.0, 100.0], got %v", args)
		}
	})
}

func TestNumberField_Int64(t *testing.T) {
	id := field.Number[int64]{}.WithColumn("id")

	t.Run("Int64Operations", func(t *testing.T) {
		expr := id.Eq(int64(12345678901234))
		sql, args, _ := expr.Build()
		if sql != "id = ?" {
			t.Errorf("Expected 'id = ?', got '%s'", sql)
		}
		if args[0] != int64(12345678901234) {
			t.Errorf("Expected int64 value, got %v", args[0])
		}
	})
}

// ============== Bool Field Tests ==============

func TestBoolField(t *testing.T) {
	active := field.Bool{}.WithColumn("is_active")

	t.Run("Eq", func(t *testing.T) {
		expr := active.Eq(true)
		sql, args, _ := expr.Build()
		if sql != "is_active = ?" {
			t.Errorf("Expected 'is_active = ?', got '%s'", sql)
		}
		if args[0] != true {
			t.Errorf("Expected true, got %v", args[0])
		}
	})

	t.Run("Neq", func(t *testing.T) {
		expr := active.Neq(false)
		sql, args, _ := expr.Build()
		if sql != "is_active <> ?" {
			t.Errorf("Expected 'is_active <> ?', got '%s'", sql)
		}
		if args[0] != false {
			t.Errorf("Expected false, got %v", args[0])
		}
	})

	t.Run("IsTrue", func(t *testing.T) {
		expr := active.IsTrue()
		sql, args, _ := expr.Build()
		if sql != "is_active = ?" {
			t.Errorf("Expected 'is_active = ?', got '%s'", sql)
		}
		if args[0] != true {
			t.Errorf("Expected true, got %v", args[0])
		}
	})

	t.Run("IsFalse", func(t *testing.T) {
		expr := active.IsFalse()
		sql, args, _ := expr.Build()
		if sql != "is_active = ?" {
			t.Errorf("Expected 'is_active = ?', got '%s'", sql)
		}
		if args[0] != false {
			t.Errorf("Expected false, got %v", args[0])
		}
	})

	t.Run("IsNull/IsNotNull", func(t *testing.T) {
		expr := active.IsNull()
		sql, _, _ := expr.Build()
		if sql != "is_active IS NULL" {
			t.Errorf("Expected 'is_active IS NULL', got '%s'", sql)
		}

		expr = active.IsNotNull()
		sql, _, _ = expr.Build()
		if sql != "is_active IS NOT NULL" {
			t.Errorf("Expected 'is_active IS NOT NULL', got '%s'", sql)
		}
	})

	t.Run("Set", func(t *testing.T) {
		assign := active.Set(true)
		sql, args, _ := assign.Build()
		if sql != "is_active = ?" {
			t.Errorf("Expected 'is_active = ?', got '%s'", sql)
		}
		if args[0] != true {
			t.Errorf("Expected true, got %v", args[0])
		}
	})

	t.Run("Asc/Desc", func(t *testing.T) {
		if sql, _, _ := active.Asc().Build(); sql != "is_active" {
			t.Errorf("Expected 'is_active', got '%s'", sql)
		}
		if sql, _, _ := active.Desc().Build(); sql != "is_active DESC" {
			t.Errorf("Expected 'is_active DESC', got '%s'", sql)
		}
	})

	t.Run("InExpr/NotInExpr", func(t *testing.T) {
		subquery := clause.Expr{SQL: "SELECT is_active FROM config"}
		expr := active.InExpr(subquery)
		sql, _, _ := expr.Build()
		if sql != "is_active IN (SELECT is_active FROM config)" {
			t.Errorf("Unexpected SQL: %s", sql)
		}
	})
}

// ============== Time Field Tests ==============

func TestTimeField(t *testing.T) {
	createdAt := field.Time{}.WithColumn("created_at")
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	later := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)

	t.Run("Eq", func(t *testing.T) {
		expr := createdAt.Eq(now)
		sql, args, _ := expr.Build()
		if sql != "created_at = ?" {
			t.Errorf("Expected 'created_at = ?', got '%s'", sql)
		}
		if args[0] != now {
			t.Errorf("Expected %v, got %v", now, args[0])
		}
	})

	t.Run("Neq", func(t *testing.T) {
		expr := createdAt.Neq(now)
		sql, args, _ := expr.Build()
		if sql != "created_at <> ?" {
			t.Errorf("Expected 'created_at <> ?', got '%s'", sql)
		}
		if args[0] != now {
			t.Errorf("Expected %v, got %v", now, args[0])
		}
	})

	t.Run("Gt", func(t *testing.T) {
		expr := createdAt.Gt(now)
		sql, _, _ := expr.Build()
		if sql != "created_at > ?" {
			t.Errorf("Expected 'created_at > ?', got '%s'", sql)
		}
	})

	t.Run("Gte", func(t *testing.T) {
		expr := createdAt.Gte(now)
		sql, _, _ := expr.Build()
		if sql != "created_at >= ?" {
			t.Errorf("Expected 'created_at >= ?', got '%s'", sql)
		}
	})

	t.Run("Lt", func(t *testing.T) {
		expr := createdAt.Lt(now)
		sql, _, _ := expr.Build()
		if sql != "created_at < ?" {
			t.Errorf("Expected 'created_at < ?', got '%s'", sql)
		}
	})

	t.Run("Lte", func(t *testing.T) {
		expr := createdAt.Lte(now)
		sql, _, _ := expr.Build()
		if sql != "created_at <= ?" {
			t.Errorf("Expected 'created_at <= ?', got '%s'", sql)
		}
	})

	t.Run("Between", func(t *testing.T) {
		expr := createdAt.Between(now, later)
		sql, args, _ := expr.Build()
		if sql != "created_at BETWEEN ? AND ?" {
			t.Errorf("Expected 'created_at BETWEEN ? AND ?', got '%s'", sql)
		}
		if args[0] != now || args[1] != later {
			t.Errorf("Expected [%v, %v], got %v", now, later, args)
		}
	})

	t.Run("IsNull/IsNotNull", func(t *testing.T) {
		expr := createdAt.IsNull()
		sql, _, _ := expr.Build()
		if sql != "created_at IS NULL" {
			t.Errorf("Expected 'created_at IS NULL', got '%s'", sql)
		}

		expr = createdAt.IsNotNull()
		sql, _, _ = expr.Build()
		if sql != "created_at IS NOT NULL" {
			t.Errorf("Expected 'created_at IS NOT NULL', got '%s'", sql)
		}
	})

	t.Run("Set", func(t *testing.T) {
		assign := createdAt.Set(now)
		sql, args, _ := assign.Build()
		if sql != "created_at = ?" {
			t.Errorf("Expected 'created_at = ?', got '%s'", sql)
		}
		if args[0] != now {
			t.Errorf("Expected %v, got %v", now, args[0])
		}
	})

	t.Run("Asc/Desc", func(t *testing.T) {
		if sql, _, _ := createdAt.Asc().Build(); sql != "created_at" {
			t.Errorf("Expected 'created_at', got '%s'", sql)
		}
		if sql, _, _ := createdAt.Desc().Build(); sql != "created_at DESC" {
			t.Errorf("Expected 'created_at DESC', got '%s'", sql)
		}
	})

	t.Run("InExpr/NotInExpr", func(t *testing.T) {
		subquery := clause.Expr{SQL: "SELECT created_at FROM audit_log"}
		expr := createdAt.InExpr(subquery)
		sql, _, _ := expr.Build()
		if sql != "created_at IN (SELECT created_at FROM audit_log)" {
			t.Errorf("Unexpected SQL: %s", sql)
		}

		expr = createdAt.NotInExpr(subquery)
		sql, _, _ = expr.Build()
		if sql != "created_at NOT IN (SELECT created_at FROM audit_log)" {
			t.Errorf("Unexpected SQL: %s", sql)
		}
	})
}

// ============== Bytes Field Tests ==============

func TestBytesField(t *testing.T) {
	data := field.Bytes{}.WithColumn("binary_data")
	sampleData := []byte("hello world")

	t.Run("Eq", func(t *testing.T) {
		expr := data.Eq(sampleData)
		sql, args, _ := expr.Build()
		if sql != "binary_data = ?" {
			t.Errorf("Expected 'binary_data = ?', got '%s'", sql)
		}
		if string(args[0].([]byte)) != "hello world" {
			t.Errorf("Expected 'hello world', got %v", args[0])
		}
	})

	t.Run("Neq", func(t *testing.T) {
		expr := data.Neq(sampleData)
		sql, args, _ := expr.Build()
		if sql != "binary_data <> ?" {
			t.Errorf("Expected 'binary_data <> ?', got '%s'", sql)
		}
		if string(args[0].([]byte)) != "hello world" {
			t.Errorf("Expected 'hello world', got %v", args[0])
		}
	})

	t.Run("IsNull/IsNotNull", func(t *testing.T) {
		expr := data.IsNull()
		sql, _, _ := expr.Build()
		if sql != "binary_data IS NULL" {
			t.Errorf("Expected 'binary_data IS NULL', got '%s'", sql)
		}

		expr = data.IsNotNull()
		sql, _, _ = expr.Build()
		if sql != "binary_data IS NOT NULL" {
			t.Errorf("Expected 'binary_data IS NOT NULL', got '%s'", sql)
		}
	})

	t.Run("Set", func(t *testing.T) {
		assign := data.Set(sampleData)
		sql, args, _ := assign.Build()
		if sql != "binary_data = ?" {
			t.Errorf("Expected 'binary_data = ?', got '%s'", sql)
		}
		if string(args[0].([]byte)) != "hello world" {
			t.Errorf("Expected 'hello world', got %v", args[0])
		}
	})

	t.Run("InExpr/NotInExpr", func(t *testing.T) {
		subquery := clause.Expr{SQL: "SELECT hash FROM banned_hashes"}
		expr := data.InExpr(subquery)
		sql, _, _ := expr.Build()
		if sql != "binary_data IN (SELECT hash FROM banned_hashes)" {
			t.Errorf("Unexpected SQL: %s", sql)
		}

		expr = data.NotInExpr(subquery)
		sql, _, _ = expr.Build()
		if sql != "binary_data NOT IN (SELECT hash FROM banned_hashes)" {
			t.Errorf("Unexpected SQL: %s", sql)
		}
	})
}

// ============== Generic Field[T] Tests ==============

func TestGenericField(t *testing.T) {
	// Test with a custom type
	type Status string
	status := field.Field[Status]{}.WithColumn("status")

	t.Run("Eq", func(t *testing.T) {
		expr := status.Eq(Status("active"))
		sql, args, _ := expr.Build()
		if sql != "status = ?" {
			t.Errorf("Expected 'status = ?', got '%s'", sql)
		}
		if args[0] != Status("active") {
			t.Errorf("Expected 'active', got %v", args[0])
		}
	})

	t.Run("Neq", func(t *testing.T) {
		expr := status.Neq(Status("inactive"))
		sql, _, _ := expr.Build()
		if sql != "status <> ?" {
			t.Errorf("Expected 'status <> ?', got '%s'", sql)
		}
	})

	t.Run("In", func(t *testing.T) {
		expr := status.In(Status("active"), Status("pending"))
		sql, args, _ := expr.Build()
		if sql != "status IN (?, ?)" {
			t.Errorf("Expected 'status IN (?, ?)', got '%s'", sql)
		}
		if len(args) != 2 {
			t.Errorf("Expected 2 args, got %d", len(args))
		}
	})

	t.Run("NotIn", func(t *testing.T) {
		// Use multiple values to avoid single-value optimization (IN -> =)
		expr := status.NotIn(Status("banned"), Status("deleted"))
		sql, _, _ := expr.Build()
		if sql != "NOT (status IN (?, ?))" {
			t.Errorf("Expected 'NOT (status IN (?, ?))', got '%s'", sql)
		}
	})

	t.Run("IsNull/IsNotNull", func(t *testing.T) {
		expr := status.IsNull()
		sql, _, _ := expr.Build()
		if sql != "status IS NULL" {
			t.Errorf("Expected 'status IS NULL', got '%s'", sql)
		}

		expr = status.IsNotNull()
		sql, _, _ = expr.Build()
		if sql != "status IS NOT NULL" {
			t.Errorf("Expected 'status IS NOT NULL', got '%s'", sql)
		}
	})

	t.Run("Set", func(t *testing.T) {
		assign := status.Set(Status("active"))
		sql, args, _ := assign.Build()
		if sql != "status = ?" {
			t.Errorf("Expected 'status = ?', got '%s'", sql)
		}
		if args[0] != Status("active") {
			t.Errorf("Expected 'active', got %v", args[0])
		}
	})

	t.Run("Asc/Desc", func(t *testing.T) {
		if sql, _, _ := status.Asc().Build(); sql != "status" {
			t.Errorf("Expected 'status', got '%s'", sql)
		}
		if sql, _, _ := status.Desc().Build(); sql != "status DESC" {
			t.Errorf("Expected 'status DESC', got '%s'", sql)
		}
	})
}

// ============== WithTable Tests ==============

func TestFieldWithTable(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		email := field.String{}.WithTable("users").WithColumn("email")
		expr := email.Eq("test@example.com")
		sql, _, _ := expr.Build()
		if sql != "users.email = ?" {
			t.Errorf("Expected 'users.email = ?', got '%s'", sql)
		}
	})

	t.Run("Number", func(t *testing.T) {
		age := field.Number[int]{}.WithTable("users").WithColumn("age")
		expr := age.Gt(18)
		sql, _, _ := expr.Build()
		if sql != "users.age > ?" {
			t.Errorf("Expected 'users.age > ?', got '%s'", sql)
		}
	})

	t.Run("Time", func(t *testing.T) {
		createdAt := field.Time{}.WithTable("users").WithColumn("created_at")
		expr := createdAt.IsNull()
		sql, _, _ := expr.Build()
		if sql != "users.created_at IS NULL" {
			t.Errorf("Expected 'users.created_at IS NULL', got '%s'", sql)
		}
	})

	t.Run("Bool", func(t *testing.T) {
		active := field.Bool{}.WithTable("users").WithColumn("is_active")
		expr := active.IsTrue()
		sql, _, _ := expr.Build()
		if sql != "users.is_active = ?" {
			t.Errorf("Expected 'users.is_active = ?', got '%s'", sql)
		}
	})

	t.Run("Bytes", func(t *testing.T) {
		data := field.Bytes{}.WithTable("files").WithColumn("content")
		expr := data.IsNull()
		sql, _, _ := expr.Build()
		if sql != "files.content IS NULL" {
			t.Errorf("Expected 'files.content IS NULL', got '%s'", sql)
		}
	})

	t.Run("Generic", func(t *testing.T) {
		type Custom string
		custom := field.Field[Custom]{}.WithTable("my_table").WithColumn("my_column")
		expr := custom.Eq(Custom("value"))
		sql, _, _ := expr.Build()
		if sql != "my_table.my_column = ?" {
			t.Errorf("Expected 'my_table.my_column = ?', got '%s'", sql)
		}
	})
}

// ============== Column and ColumnName Tests ==============

func TestColumnMethods(t *testing.T) {
	t.Run("Column", func(t *testing.T) {
		email := field.String{}.WithColumn("email").WithTable("users")
		col := email.Column()
		if col.Name != "email" {
			t.Errorf("Expected column name 'email', got '%s'", col.Name)
		}
		if col.Table != "users" {
			t.Errorf("Expected table 'users', got '%s'", col.Table)
		}
	})

	t.Run("ColumnName without table", func(t *testing.T) {
		email := field.String{}.WithColumn("email")
		if email.ColumnName() != "email" {
			t.Errorf("Expected 'email', got '%s'", email.ColumnName())
		}
	})

	t.Run("ColumnName with table", func(t *testing.T) {
		email := field.String{}.WithTable("users").WithColumn("email")
		if email.ColumnName() != "users.email" {
			t.Errorf("Expected 'users.email', got '%s'", email.ColumnName())
		}
	})
}

// ============== Complex Expression Tests ==============

func TestComplexExpression(t *testing.T) {
	age := field.Number[int]{}.WithColumn("age")
	status := field.String{}.WithColumn("status")
	role := field.String{}.WithColumn("role")

	t.Run("OrAndCombination", func(t *testing.T) {
		// (age > 18 AND status = 'active') OR role = 'admin'
		expr := clause.Or{
			clause.And{
				age.Gt(18),
				status.Eq("active"),
			},
			role.Eq("admin"),
		}

		sql, args, _ := expr.Build()
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
	})

	t.Run("NotExpression", func(t *testing.T) {
		expr := clause.Not{Expr: status.Eq("banned")}
		sql, args, _ := expr.Build()
		if sql != "NOT (status = ?)" {
			t.Errorf("Expected 'NOT (status = ?)', got '%s'", sql)
		}
		if args[0] != "banned" {
			t.Errorf("Expected 'banned', got %v", args[0])
		}
	})
}

// ============== OrderBy Tests ==============

func TestOrderBy(t *testing.T) {
	createdAt := field.Time{}.WithColumn("created_at")

	t.Run("Asc", func(t *testing.T) {
		order := createdAt.Asc()
		sql, args, _ := order.Build()
		if sql != "created_at" {
			t.Errorf("Expected 'created_at', got '%s'", sql)
		}
		if len(args) != 0 {
			t.Errorf("Expected no args, got %d", len(args))
		}
	})

	t.Run("Desc", func(t *testing.T) {
		order := createdAt.Desc()
		sql, args, _ := order.Build()
		if sql != "created_at DESC" {
			t.Errorf("Expected 'created_at DESC', got '%s'", sql)
		}
		if len(args) != 0 {
			t.Errorf("Expected no args, got %d", len(args))
		}
	})

	t.Run("WithTable", func(t *testing.T) {
		order := field.String{}.WithTable("users").WithColumn("name").Desc()
		sql, _, _ := order.Build()
		if sql != "users.name DESC" {
			t.Errorf("Expected 'users.name DESC', got '%s'", sql)
		}
	})
}

// ============== Assignment Tests ==============

func TestAssignment(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		email := field.String{}.WithColumn("email")
		assign := email.Set("new@example.com")
		sql, args, _ := assign.Build()
		if sql != "email = ?" {
			t.Errorf("Expected 'email = ?', got '%s'", sql)
		}
		if args[0] != "new@example.com" {
			t.Errorf("Expected 'new@example.com', got %v", args[0])
		}
	})

	t.Run("Number", func(t *testing.T) {
		age := field.Number[int]{}.WithColumn("age")
		assign := age.Set(25)
		sql, args, _ := assign.Build()
		if sql != "age = ?" {
			t.Errorf("Expected 'age = ?', got '%s'", sql)
		}
		if args[0] != 25 {
			t.Errorf("Expected 25, got %v", args[0])
		}
	})

	t.Run("Bool", func(t *testing.T) {
		active := field.Bool{}.WithColumn("is_active")
		assign := active.Set(true)
		sql, args, _ := assign.Build()
		if sql != "is_active = ?" {
			t.Errorf("Expected 'is_active = ?', got '%s'", sql)
		}
		if args[0] != true {
			t.Errorf("Expected true, got %v", args[0])
		}
	})

	t.Run("Time", func(t *testing.T) {
		now := time.Now()
		updatedAt := field.Time{}.WithColumn("updated_at")
		assign := updatedAt.Set(now)
		sql, args, _ := assign.Build()
		if sql != "updated_at = ?" {
			t.Errorf("Expected 'updated_at = ?', got '%s'", sql)
		}
		if args[0] != now {
			t.Errorf("Expected %v, got %v", now, args[0])
		}
	})

	t.Run("Bytes", func(t *testing.T) {
		data := field.Bytes{}.WithColumn("data")
		assign := data.Set([]byte("test"))
		sql, args, _ := assign.Build()
		if sql != "data = ?" {
			t.Errorf("Expected 'data = ?', got '%s'", sql)
		}
		if string(args[0].([]byte)) != "test" {
			t.Errorf("Expected 'test', got %v", args[0])
		}
	})
}

// ============== Interface Implementation Tests ==============

func TestColumnarInterface(t *testing.T) {
	// Ensure all field types implement clause.Columnar
	var _ clause.Columnar = field.String{}
	var _ clause.Columnar = field.Number[int]{}
	var _ clause.Columnar = field.Number[float64]{}
	var _ clause.Columnar = field.Bool{}
	var _ clause.Columnar = field.Time{}
	var _ clause.Columnar = field.Bytes{}
	var _ clause.Columnar = field.Field[string]{}
}

// ============== Edge Cases Tests ==============

func TestEdgeCases(t *testing.T) {
	t.Run("EmptyInValues", func(t *testing.T) {
		// Empty IN should return a false condition
		status := field.String{}.WithColumn("status")
		expr := status.In()
		sql, args, _ := expr.Build()
		if sql != "1 = 0" {
			t.Errorf("Expected '1 = 0' for empty IN, got '%s'", sql)
		}
		if len(args) != 0 {
			t.Errorf("Expected no args, got %d", len(args))
		}
	})

	t.Run("SingleInValue", func(t *testing.T) {
		// Single value IN should be optimized to =
		status := field.String{}.WithColumn("status")
		expr := status.In("active")
		sql, args, _ := expr.Build()
		if sql != "status = ?" {
			t.Errorf("Expected 'status = ?' for single IN, got '%s'", sql)
		}
		if len(args) != 1 || args[0] != "active" {
			t.Errorf("Expected ['active'], got %v", args)
		}
	})

	t.Run("EmptyAnd", func(t *testing.T) {
		expr := clause.And{}
		sql, _, _ := expr.Build()
		if sql != "1 = 1" {
			t.Errorf("Expected '1 = 1' for empty AND, got '%s'", sql)
		}
	})

	t.Run("EmptyOr", func(t *testing.T) {
		expr := clause.Or{}
		sql, _, _ := expr.Build()
		if sql != "1 = 0" {
			t.Errorf("Expected '1 = 0' for empty OR, got '%s'", sql)
		}
	})
}
