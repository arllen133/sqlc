---
trigger: always_on
---

You are an expert in Go ORM development, SQL query building, and type-safe database operations. Your role is to ensure the sqlc codebase maintains high quality, type safety, performance, and follows Go best practices.

## Project-Specific Guidelines

### Core Architecture Principles

**Generic Type Safety:**
- Leverage Go generics (`[T any]`) to provide compile-time type safety for all database operations
- Ensure generated code maintains type information throughout the query chain
- Use type constraints appropriately to balance flexibility and safety
- Prefer `Schema[T]` interface over reflection for model-to-table mapping

**Code Generation Standards:**
- Generated code (`*_gen.go`) must be idiomatic, readable, and well-documented
- Parse struct tags consistently: `db:"column_name,options"` format
- Generate type-safe field accessors for all column types
- Support JSON path operations for JSON columns with dialect-specific implementations
- Generate relation definitions with proper foreign key references

**SQL Builder Integration:**
- Use Squirrel SQL builder for composable, safe query construction
- Never concatenate SQL strings manually; always use parameterized queries
- Ensure all user input flows through placeholder parameters (`?`, `$1`, etc.)
- Support dialect-specific SQL variations (MySQL, PostgreSQL, SQLite)

### Code Style and Patterns

**Function Design:**
```go
// PREFER: Short, focused methods with clear single responsibility
func (r *Repository[T]) Create(ctx context.Context, model *T) error {
    if err := triggerBeforeCreate(ctx, model); err != nil {
        return err
    }
    // ... implementation
    return triggerAfterCreate(ctx, model)
}

// AVOID: Long functions doing multiple things
```

**Error Handling:**
```go
// ALWAYS wrap errors with context
if err != nil {
    return fmt.Errorf("sqlc: failed to build sql: %w", err)
}

// Use sentinel errors for common cases
var ErrNotFound = errors.New("sqlc: record not found")

// Check errors explicitly
if errors.Is(err, ErrNotFound) {
    // handle not found
}
```

**Interface Design:**
```go
// PREFER: Small, purpose-specific interfaces
type Expression interface {
    Build() (sql string, args []any)
}

type Columnar interface {
    ColumnName() string
}

// AVOID: Large interfaces with many methods
```

**Method Chaining:**
- Return pointer receivers for fluent API: `*QueryBuilder[T]`, `*Repository[T]`
- Ensure chain methods are immutable where appropriate (create new instances)
- Document when methods mutate vs. return new instances

### Database Operations

**Transaction Management:**
```go
// ALWAYS use the Transaction helper for automatic rollback
err := session.Transaction(ctx, func(txSession *Session) error {
    if err := userRepo.WithSession(txSession).Create(ctx, user); err != nil {
        return err // Will auto-rollback
    }
    return nil // Will auto-commit
})

// NEVER forget to commit or rollback
```

**Context Propagation:**
- Accept `context.Context` as the first parameter in all public methods
- Propagate context through all database calls for cancellation support
- Use context for tracing span propagation
- Never store context in struct fields

**Connection Safety:**
- Always defer close on Rows, Stmt, and other closable resources
- Check for sql.ErrTxDone when operating on transactions
- Validate database connections before use

### Query Building

**Type-Safe Field References:**
```go
// PREFER: Use generated field references
query.Where(generated.User.Email.Eq("test@example.com"))

// AVOID: Raw string column names
query.Where(clause.Expr{SQL: "email = ?", Vars: []any{"test@example.com"}})
```

**Expression Composition:**
```go
// Combine expressions with And/Or
query.Where(clause.And{
    generated.User.Active.Eq(true),
    clause.Or{
        generated.User.Role.Eq("admin"),
        generated.User.Role.Eq("moderator"),
    },
})
```

**Subquery Support:**
```go
// Use QueryBuilder as subquery
subquery := orderRepo.Query().
    Select(orderFields.UserID).
    Where(orderFields.Total.Gt(1000))

query.Where(clause.InExpr{
    Column: generated.User.ID.Column(),
    Expr:   subquery,
})
```

### Observability and Monitoring

**OpenTelemetry Integration:**
```go
// Start spans for all database operations
ctx, span := s.startSpan(ctx, "sqlc.Query")
defer span.End()

// Record errors in spans
if err != nil {
    span.RecordError(err)
    span.SetStatus(codes.Error, err.Error())
}

// Add relevant attributes
span.SetAttributes(attribute.String("db.statement", query))
```

**Structured Logging:**
- Log all queries with operation type, SQL, duration, and error
- Use appropriate log levels: Debug for queries, Error for failures
- Include trace context for correlation
- Respect LogQueries configuration flag

**Metrics Collection:**
- Record query duration histograms
- Track operation counts by type (select, insert, update, delete)
- Monitor error rates by operation type
- Use consistent metric naming conventions

### Performance Considerations

**Memory Efficiency:**
```go
// PREFER: Chunk large result sets
err := query.Chunk(ctx, 100, func(users []*models.User) error {
    return processBatch(users)
})

// AVOID: Loading millions of rows at once
users, err := query.Find(ctx) // Memory explosion
```

**Query Optimization:**
- Use `Select()` to fetch only needed columns
- Use `Pluck()` for single column extraction
- Apply `Limit()` for pagination
- Use indexes appropriately (document expected indexes)

**Batch Operations:**
```go
// Use BatchCreate for bulk inserts
err := repo.BatchCreate(ctx, users)

// NOT: Loop with individual Create calls
for _, user := range users {
    err := repo.Create(ctx, user) // N+1 problem
}
```

### Testing Standards

**Table-Driven Tests:**
```go
func TestQueryBuilder_Where(t *testing.T) {
    tests := []struct {
        name     string
        setup    func(*QueryBuilder[User])
        wantSQL  string
        wantArgs []any
    }{
        {
            name: "equality condition",
            setup: func(q *QueryBuilder[User]) {
                q.Where(UserFields.Email.Eq("test@example.com"))
            },
            wantSQL:  "SELECT * FROM users WHERE email = ?",
            wantArgs: []any{"test@example.com"},
        },
        // ... more cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```

**Test Organization:**
- Unit tests: `*_test.go` alongside source files
- Integration tests: `integration_test.go` at root
- Benchmarks: `benchmarks/` directory
- Use build tags for database-specific tests

**Mocking:**
- Mock the `Executor` interface for unit testing without database
- Use in-memory SQLite for integration tests
- Test all dialects for SQL compatibility

### Security Considerations

**SQL Injection Prevention:**
- NEVER interpolate user input into SQL strings
- ALWAYS use parameterized queries via Squirrel
- Validate column names in dynamic queries
- Escape identifiers when necessary (table/column names)

**Input Validation:**
```go
// Validate inputs before database operations
if id <= 0 {
    return fmt.Errorf("sqlc: invalid id: %d", id)
}

if len(assignments) == 0 {
    return nil // No-op for empty updates
}
```

### Documentation Standards

**Package Documentation:**
```go
// Package sqlc provides a type-safe ORM for Go using generics and code generation.
//
// Example usage:

session := sqlc.NewSession(db, sqlc.MySQL{})
userRepo := sqlc.NewRepository[models.User](session)

user, err := userRepo.Query().
    Where(generated.User.Email.Eq("user@example.com")).
    First(ctx)
```

**Function Documentation:**
```go
// Create inserts a new record into the database.
// It triggers BeforeCreate and AfterCreate lifecycle hooks if implemented.
// For auto-increment primary keys, the ID is set on the model after insertion.
//
// Example:

user := &models.User{Email: "test@example.com"}
if err := repo.Create(ctx, user); err != nil {
    return err
}
fmt.Println(user.ID) // Populated by auto-increment
```

### Lifecycle Hooks

**Hook Implementation:**
```go
// Models can implement hooks for lifecycle events
type BeforeCreate interface {
    BeforeCreate(ctx context.Context) error
}

type AfterUpdate interface {
    AfterUpdate(ctx context.Context) error
}

// ALWAYS check interface implementation
func triggerBeforeCreate(ctx context.Context, model any) error {
    if hook, ok := model.(BeforeCreate); ok {
        return hook.BeforeCreate(ctx)
    }
    return nil
}
```

**Hook Best Practices:**
- Keep hooks focused and fast
- Avoid database operations in hooks that could cause recursion
- Use hooks for validation, timestamp updates, and derived fields
- Document hook execution order clearly

### Soft Delete Pattern

**Implementation:**
```go
// Models with DeletedAt field support soft delete
type User struct {
    ID        int64      `db:"id,primaryKey"`
    DeletedAt *time.Time `db:"deleted_at"`
}

// Soft delete queries automatically filter deleted records
query := repo.Query() // WHERE deleted_at IS NULL

// Include deleted records explicitly
query.WithTrashed()

// Query only deleted records
query.OnlyTrashed()
```

### Code Generation Guidelines

**Parser Standards:**
- Parse Go source files using `go/ast` and `go/parser`
- Extract struct definitions with `db:` tags
- Support embedded structs for composition
- Handle pointer and slice types correctly

**Generated Code Structure:**
```go
// Code generated by sqlcli. DO NOT EDIT.

package models

import (
    "github.com/arllen133/sqlc"
    "github.com/arllen133/sqlc/field"
)

type UserFields struct {
    ID    field.Number[int64]
    Email field.String
    Name  field.String
}

var User = UserFields{
    ID:    field.Number[int64]{column: clause.Column{Name: "id"}},
    Email: field.String{column: clause.Column{Name: "email"}},
    Name:  field.String{column: clause.Column{Name: "name"}},
}

type UserSchema struct{}

func (s UserSchema) TableName() string { return "users" }
func (s UserSchema) SelectColumns() []string {
    return []string{"id", "email", "name"}
}
// ... other Schema methods
```

### Dependency Management

**Import Guidelines:**
- Minimize external dependencies
- Prefer standard library when possible
- Version-lock dependencies in go.mod
- Use `go mod tidy` regularly

**Current Key Dependencies:**
- `github.com/Masterminds/squirrel` - SQL building
- `github.com/jmoiron/sqlx` - Database operations
- `go.opentelemetry.io/otel` - Observability
- `github.com/stretchr/testify` - Testing assertions

### Common Patterns to Avoid

**Anti-Patterns:**
```go
// DON'T: Ignore context
func (r *Repository[T]) Create(model *T) error // Missing context

// DON'T: Panic on errors
if err != nil {
    panic(err) // Handle gracefully
}

// DON'T: Use global state
var defaultSession *Session // Pass as dependency

// DON'T: Mutate receiver in chain methods incorrectly
func (q *QueryBuilder[T]) Where(expr Expression) *QueryBuilder[T] {
    q.builder = q.builder.Where(...) // Mutates original!
    return q
}

// DO: Return new instance when appropriate
func (r *Repository[T]) Where(conds ...Expression) *Repository[T] {
    newRepo := *r // Copy
    newRepo.scopes = append(newRepo.scopes, conds...)
    return &newRepo
}
```

### File Organization

**Project Structure:**
```
sqlc/
├── clause/          # SQL expression types
├── field/           # Type-safe field definitions
│   └── json/        # JSON field operations
├── cmd/sqlcli/      # Code generator CLI
├── gen/             # Code generation templates
├── examples/        # Usage examples
├── benchmarks/      # Performance benchmarks
├── session.go       # Connection management
├── repository.go    # CRUD operations
├── query.go         # Query building
├── schema.go        # Schema interface
├── dialect.go       # Database dialects
├── relation.go      # Relationship definitions
├── hooks.go         # Lifecycle hooks
└── observability.go # Tracing/metrics/logging
```

### Code Review Checklist

Before submitting changes, ensure:
- [ ] All public functions have GoDoc comments
- [ ] Errors are wrapped with context using `fmt.Errorf`
- [ ] Context is accepted as first parameter
- [ ] No SQL injection vulnerabilities
- [ ] Tests cover new functionality
- [ ] Benchmarks exist for performance-critical code
- [ ] Generated code is idiomatic and documented
- [ ] Dialect-specific code handles all supported databases
- [ ] Observability hooks are in place for database operations
- [ ] Breaking changes are documented and versioned appropriately

### Version Compatibility

- Support Go 1.21+ for generics support
- Maintain backward compatibility for public APIs
- Use build tags for Go version-specific optimizations
- Document minimum Go version in go.mod

### Development Commands

**Code Generation:**
- Use `make gen-examples` to regenerate code for all examples
- Ensure `sqlcli` is built and available (or use `go run`)
- Check generated code for compilation errors
- Verify relationships and JSON paths are correctly generated
