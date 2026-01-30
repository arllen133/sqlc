# sqlc - 高性能 Golang ORM 框架

基于 **Go 泛型 + 代码生成 + Squirrel SQL Builder** 的现代化 ORM 框架。

## 核心特性

- ✅ **泛型驱动** - 使用 Go 1.18+ 泛型实现类型安全的 API，零反射运行时开销
- ✅ **代码生成** - 自动生成模型元数据，每个模型仅需 ~50 行生成代码
- ✅ **SQL 构建** - 集成 Squirrel 处理 SQL 拼接，自动适配 MySQL/PostgreSQL/SQLite
- ✅ **链式查询** - 流畅的查询构建器 API
- ✅ **事务支持** - 自动提交/回滚的事务管理
- ✅ **生命周期钩子** - BeforeCreate/AfterCreate 等模型钩子
- ✅ **性能极致** - 目标比 GORM 快 3-5 倍

## 快速开始

### 安装

```bash
go get github.com/arllen133/sqlc
```

### 定义模型

```go
package models

import "time"

type User struct {
    ID        int64     `orm:"column:id;primaryKey;autoIncrement"`
    Username  string    `orm:"column:username"`
    Email     string    `orm:"column:email"`
    CreatedAt time.Time `orm:"column:created_at"`
}
```

### 生成代码

```bash
go install github.com/arllen133/sqlc/cmd/orm-gen@latest
orm-gen -model ./models -output ./models
```

这将生成 `user_gen.go`，包含：

- `UserFields` - 类型安全的字段定义
- `UserSchema` - 表元数据实现

### 使用 ORM

```go
package main

import (
    "context"
    "database/sql"

    _ "github.com/mattn/go-sqlite3"
    "github.com/arllen133/sqlc"
    "yourapp/models"
)

func main() {
    // 1. 连接数据库
    db, _ := sql.Open("sqlite3", "app.db")
    session := sqlc.NewSession(db, &sqlc.SQLiteDialect{})

    // 2. 创建仓储
    userRepo := sqlc.NewRepository[models.User](session)
    ctx := context.Background()

    // 3. 创建记录
    user := &models.User{
        Username: "alice",
        Email:    "alice@example.com",
    }
    userRepo.Create(ctx, user)
    // user.ID 自动回填

    // 4. 类型安全查询
    users, _ := userRepo.Query().
        Where(models.UserFields.Username.Eq("alice")).
        OrderBy(models.UserFields.CreatedAt.Asc()).
        Limit(10).
        Find(ctx)

    // 5. 更新
    user.Email = "new@example.com"
    userRepo.Update(ctx, user)

    // 6. 删除
    userRepo.Delete(ctx, user.ID)
}
```

## 高级功能

### 事务

```go
err := session.Transaction(ctx, func(txSession *sqlc.Session) error {
    txRepo := sqlc.NewRepository[models.User](txSession)

    user1 := &models.User{Username: "user1"}
    if err := txRepo.Create(ctx, user1); err != nil {
        return err // 自动回滚
    }

    user2 := &models.User{Username: "user2"}
    if err := txRepo.Create(ctx, user2); err != nil {
        return err // 自动回滚
    }

    return nil // 自动提交
})
```

### 生命周期钩子

```go
func (u *User) BeforeCreate(ctx context.Context) error {
    if u.CreatedAt.IsZero() {
        u.CreatedAt = time.Now()
    }
    return nil
}

func (u *User) AfterCreate(ctx context.Context) error {
    log.Printf("User %s created with ID %d", u.Username, u.ID)
    return nil
}
```

支持的钩子接口：

- `BeforeCreate` / `AfterCreate`
- `BeforeUpdate` / `AfterUpdate`
- `BeforeDelete` / `AfterDelete`

注意：删除钩子仅在调用 `repo.DeleteModel(ctx, model)` 时触发。

### 复杂查询

#### 传统方式

```go
// WHERE 条件
users, _ := userRepo.Query().
    Where(models.UserFields.Age.Gt(18)).
    Where(models.UserFields.Status.Eq("active")).
    Find(ctx)
```

#### 流畅表达式 API（推荐）

```go
// 使用流畅的表达式 API
users, _ := userRepo.Query().
    Where(models.UserFields.Age.Gt(18)).
    Where(models.UserFields.Status.Eq("active")).
    Find(ctx)

// 支持的表达式方法
models.UserFields.Username.Eq("alice")      // username = 'alice'
models.UserFields.Age.Ne(18)                // age != 18
models.UserFields.Age.Gt(18)                // age > 18
models.UserFields.Age.Gte(18)               // age >= 18
models.UserFields.Age.Lt(30)                // age < 30
models.UserFields.Age.Lte(30)               // age <= 30
models.UserFields.Username.Like("%alice%")  // username LIKE '%alice%'
models.UserFields.Status.In("active", "pending")  // status IN ('active', 'pending')
models.UserFields.Email.IsNull()            // email IS NULL
models.UserFields.Email.IsNotNull()         // email IS NOT NULL
```

#### 排序和分页

```go
users, _ := userRepo.Query().
    OrderBy(models.UserFields.CreatedAt.Desc()).
    Limit(20).
    Offset(40).
    Find(ctx)
```

#### 统计

```go
count, _ := userRepo.Query().
    Where(models.UserFields.Status.Eq("active")).
    Count(ctx)
```

### 支持的操作符 (Internal)

- `Eq` - 等于 (=)
- `Neq` - 不等于 (<>)
- `Gt` - 大于 (>)
- `Gte` - 大于等于 (>=)
- `Lt` - 小于 (<)
- `Lte` - 小于等于 (<=)
- `In` - IN
- `Like` - LIKE
- `IsNull` - IS NULL
- `IsNotNull` - IS NOT NULL

## ORM 标签

```go
type User struct {
    ID   int64  `orm:"column:id;primaryKey;autoIncrement"`
    Name string `orm:"column:name;size:100;unique"`
    Age  int    `orm:"column:age;index"`
}
```

支持的标签：

- `unique` - 唯一约束（用于文档）
- `index` - 索引（用于文档）

## 高级查询功能

### 1. 关联查询 (JOIN)

```go
// SELECT users.*, departments.name FROM users JOIN departments ...
repo.Query().
    Join("departments", clause.Expr{SQL: "users.dept_id = departments.id"}).
    Find(ctx)
```

支持 `Join` (Inner), `LeftJoin`, `RightJoin`。

### 2. 分组与聚合 (Group By & Aggregates)

```go
// SELECT dept_id, COUNT(*) FROM users GROUP BY dept_id HAVING COUNT(*) > 5
repo.Query().
    GroupBy(models.UserFields.DeptID).
    Having(clause.Expr{SQL: "COUNT(*) > 5"}).
    Count(ctx)

// 聚合函数
maxScore, _ := repo.Query().Max(ctx, models.UserFields.Score)
avgAge, _ := repo.Query().Avg(ctx, models.UserFields.Age)
```

### 3. 自定义列选择

```go
// 默认 SELECT *，可覆盖为特定列
repo.Query().
    Select(models.UserFields.ID, models.UserFields.Username).
    Find(ctx)
```

## 批量操作与 Upsert

### 1. BatchCreate

高效插入多条记录：

```go
users := []*models.User{{...}, {...}}
err := repo.BatchCreate(ctx, users)
```

### 2. Upsert

支持 `INSERT ... ON CONFLICT/DUPLICATE KEY UPDATE`：

```go
// 使用默认行为：冲突时更新非主键列
err := repo.Upsert(ctx, user)

// 自定义冲突列和更新列
err := repo.Upsert(ctx, user,
    sqlc.OnConflict(models.UserFields.Email), // 指定冲突列
    sqlc.DoUpdate(models.UserFields.Username), // 指定更新列
)
```

自动适配 MySQL, SQLite, PostgreSQL 语法差异。

## 数据库支持

- ✅ SQLite
- ✅ MySQL
- ✅ PostgreSQL

通过 Squirrel 自动处理占位符差异（`?` vs `$1`）。

## 性能对比

| ORM      | 查询性能   | 内存分配   | 类型安全   |
| -------- | ---------- | ---------- | ---------- |
| **sqlc** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| GORM     | ⭐⭐⭐     | ⭐⭐⭐     | ⭐⭐       |
| sqlx     | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐         |

## 项目结构

```
sqlc/
├── core.go           # Session 和事务管理
├── query.go          # 泛型查询构建器
├── repository.go     # 泛型仓储
├── schema.go         # Schema 接口和注册表
├── hooks.go          # 生命周期钩子
├── field.go          # 类型安全字段
├── dialect.go        # 数据库方言
├── operator.go       # SQL 操作符
└── cmd/orm-gen/      # 代码生成器
    ├── main.go
    └── generator/
        ├── parser.go
        └── generator.go
```

## 设计理念

1. **类型安全优先** - 编译期发现错误，而非运行时
2. **零反射运行时** - 泛型实现核心逻辑，性能接近手写 SQL
3. **最小化生成** - 仅生成必要的元数据（~50 行/模型）
4. **职责分离** - ORM 做抽象，Squirrel 做 SQL 拼接
5. **渐进增强** - 从简单 CRUD 到复杂查询，API 一致

## 示例项目

查看 `examples/blog` 目录获取完整示例。

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！
