# sqlc - High Performance Golang ORM Framework

A modern ORM framework based on **Go Generics + Code Generation + Squirrel SQL Builder**.

## Core Features

- ✅ **Generics Driven** - Type-safe API using Go 1.18+ generics, zero runtime reflection overhead.
- ✅ **Code Generation** - Auto-generate model metadata, only ~50 lines of code per model.
- ✅ **SQL Builder** - Integrated Squirrel for SQL construction, auto-adapts to MySQL/PostgreSQL/SQLite.
- ✅ **Fluent Query API** - Chainable query builder.
- ✅ **Transaction Support** - Auto-commit/rollback transaction management.
- ✅ **Lifecycle Hooks** - Model hooks like BeforeCreate/AfterCreate.
- ✅ **JSON Support** - Rich JSON operations including querying, updating, and deep merging (RFC 7396).
- ✅ **Observability** - Built-in support for `slog` logging and OpenTelemetry tracing.
- ✅ **High Performance** - High-performance ORM designed for speed and efficiency.

## Quick Start

### Installation

```bash
go get github.com/arllen133/sqlc
```

### Define Model

```go
package models

import "time"

type User struct {
    ID        int64     `db:"id,primaryKey,autoIncrement"`
    Username  string    `db:"username,size:100,unique"`
    Email     string    `db:"email,size:255"`
    CreatedAt time.Time `db:"created_at"`
}
```

### Generate Code

```bash
go install github.com/arllen133/sqlc/cmd/sqlcli@latest
sqlcli -i ./models
```

This generates `models/generated/user_gen.go`, containing:

- `generated.User` - Schema instance with type-safe field definitions.
- `generated.UserMetadata` - JSON path accessors (if JSON fields exist).

### Declarative Configuration (Optional)

Create a `config.go` file in your model directory to customize code generation:

```go
// models/config.go
package models

import "github.com/arllen133/sqlc/gen"

var _ = gen.Config{
    OutPath:        "../generated",              // Output directory (relative to model dir)
    IncludeStructs: []any{"User", Post{}},       // Supports strings and type literals
    ExcludeStructs: []any{BaseModel{}, "Draft"}, // Skip these structs
}
```

When `sqlcli` runs, it automatically detects and applies this configuration.

### Usage

```go
package main

import (
    "context"
    "database/sql"
    "log/slog"
    "time"

    _ "github.com/mattn/go-sqlite3"
    "github.com/arllen133/sqlc"
    "yourapp/models"
    "yourapp/models/generated" // Import generated code
)

func main() {
    // 1. Connect to Database with Observability
    db, _ := sql.Open("sqlite3", "app.db")
    session := sqlc.NewSession(db, &sqlc.SQLiteDialect{},
        sqlc.WithLogger(slog.Default()), // Enable logging
        sqlc.WithDefaultTracer(),        // Enable OpenTelemetry tracing
    )

    // 2. Create Repository
    userRepo := sqlc.NewRepository[models.User](session)
    ctx := context.Background()

    // 3. Create Record
    user := &models.User{
        Username: "alice",
        Email:    "alice@example.com",
    }
    userRepo.Create(ctx, user)
    // user.ID is auto-filled

    // 4. Type-Safe Query
    users, _ := userRepo.Query().
        Where(generated.User.Username.Eq("alice")).
        OrderBy(generated.User.CreatedAt.Asc()).
        Limit(10).
        Find(ctx)

    // 5. Update
    user.Email = "new@example.com"
    userRepo.Update(ctx, user)

    // 6. Delete
    userRepo.Delete(ctx, user.ID)
}
```

## Advanced Features

### Transactions

```go
err := session.Transaction(ctx, func(txSession *sqlc.Session) error {
    txRepo := sqlc.NewRepository[models.User](txSession)

    user1 := &models.User{Username: "user1"}
    if err := txRepo.Create(ctx, user1); err != nil {
        return err // Auto-rollback
    }

    return nil // Auto-commit
})
```

### JSON Operations

Rich support for JSON columns with dialect-specific optimizations (MySQL, PostgreSQL, SQLite).

#### Define JSON Field

```go
type Metadata struct {
    Tags []string `json:"tags"`
    Info struct {
        Age int `json:"age"`
    } `json:"info"`
}

type Post struct {
    ID   int64                 `db:"id"`
    Meta sqlc.JSON[Metadata]   `db:"meta,type:json"`
}
```

#### JSON Querying

```go
import "yourapp/models/generated"

// Query by JSON path
repo.Query().
    Where(generated.Metadata.Age.Gt(18)). // Metadata is generated from Metadata struct name
    Where(generated.Metadata.Tags.Contains("golang")).
    Find(ctx)
```

#### JSON Updates

```go
// Partial update (JSON_SET)
repo.UpdateColumns(ctx, id,
    generated.Metadata.Age.Set(25),
)

// Remove field (JSON_REMOVE)
repo.UpdateColumns(ctx, id,
    generated.Metadata.Info.Remove(), // Assuming Info itself can be removed or a field inside it
    // Or for a custom path:
    field.JSONRemove(generated.Post.Meta, "$.deprecated_field"),
)
```

#### JSON Merge (RFC 7396)

Efficiently merge JSON objects using DB-native functions (`JSON_MERGE_PATCH` in MySQL, `||` in Postgres, `json_patch` in SQLite).

```go
// Merge Patch
repo.UpdateColumns(ctx, id,
    generated.Post.Meta.MergePatch(map[string]any{
        "view_count": 100,
        "tags": []string{"updated"},
    }),
)
```

### Relations & Eager Loading

Support for defining and eagerly loading relationships (HasOne, HasMany, BelongsTo) to avoid N+1 query problems.

#### Define Relations

Use the `relation` tag on your struct fields. These fields are typically not stored in the database column matching the field name, so use `db:"-"`.

```go
type User struct {
    ID    int64   `db:"id,primaryKey,autoIncrement"`
    Posts []*Post `db:"-" relation:"hasMany,foreignKey:user_id"`
}

type Post struct {
    ID     int64 `db:"id,primaryKey"`
    UserID int64 `db:"user_id"`
    // BelongsTo: FK (user_id) is on the Post struct
    Author *User `db:"-" relation:"belongsTo,foreignKey:user_id"`
}
```

#### Generate Code

Running `sqlcli` will automatically generate relation metadata in your `*_gen.go` files, e.g., `generated.User_Posts`.

```bash
sqlcli -i ./models
```

#### Eager Loading (Preload)

Use `WithPreload` to load relationships efficiently (usually via logical IN queries).

```go
// Load Users with their Posts
users, _ := userRepo.Query().
    WithPreload(sqlc.Preload(generated.User_Posts)).
    Find(ctx)

for _, u := range users {
    fmt.Printf("User %d has %d posts\n", u.ID, len(u.Posts))
}

// Load Posts with their Author
posts, _ := postRepo.Query().
    WithPreload(sqlc.Preload(generated.Post_Author)).
    Find(ctx)
```

### Observability

#### Logging

Built-in support for `log/slog`.

```go
sess := sqlc.NewSession(db, dialect,
    sqlc.WithLogger(slog.Default()),
    sqlc.WithSlowQueryThreshold(200*time.Millisecond), // Alert on slow queries
    sqlc.WithQueryLogging(true),                       // Log all queries (debug)
)
```

#### Tracing

Built-in integration with OpenTelemetry.

```go
import "go.opentelemetry.io/otel"

sess := sqlc.NewSession(db, dialect,
    sqlc.WithTracer(otel.Tracer("my-service")),
)
```

Spans include attributes like `db.statement`, `db.system`, and `db.table`.

### Fluent Expressions

```go
models.UserFields.Username.Eq("alice")      // username = 'alice'
models.UserFields.Age.Ne(18)                // age != 18
models.UserFields.Age.Gt(18)                // age > 18
models.UserFields.Age.Gte(18)               // age >= 18
models.UserFields.Age.Lt(30)                // age < 30
models.UserFields.Status.In("a", "b")       // status IN ('a', 'b')
models.UserFields.Username.Like("%alice%")  // username LIKE '%alice%'
models.UserFields.Email.IsNull()            // email IS NULL
```

### Joins and Aggregations

```go
// JOIN
repo.Query().
    Join("departments", clause.Expr{SQL: "users.dept_id = departments.id"}).
    Find(ctx)

// Aggregation
count, _ := repo.Query().
    Where(models.UserFields.Status.Eq("active")).
    Count(ctx)
```

### Upsert

Support `INSERT ... ON CONFLICT/DUPLICATE KEY UPDATE` across databases.

```go
// Default: Update non-PK columns on conflict
repo.Upsert(ctx, user)

// Custom: Specific conflict target and update columns
repo.Upsert(ctx, user,
    sqlc.OnConflict(models.UserFields.Email),
    sqlc.DoUpdate(models.UserFields.Username),
)
```

## Database Support

- ✅ **SQLite** (Modern JSON support)
- ✅ **MySQL** (5.7+, 8.0+)
- ✅ **PostgreSQL** (JSONB support)

## License

MIT License
