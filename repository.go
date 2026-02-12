// Package sqlc provides a type-safe ORM library using generics and code generation.
// This file implements the Repository type, providing complete CRUD operation support.
//
// Repository is the core component of sqlc ORM, providing type-safe database operations for model T.
// It encapsulates all common database operations, including:
//   - Create (Create, BatchCreate, Upsert)
//   - Read (FindOne, Query)
//   - Update (Update, UpdateColumns)
//   - Delete (Delete, DeleteModel, SoftDelete, ForceDelete)
//   - Soft delete support (SoftDelete, Restore)
//   - Conditional scoping (Where)
package sqlc

import (
	"context"
	"errors"
	"fmt"
	"slices"

	sq "github.com/Masterminds/squirrel"
	"github.com/arllen133/sqlc/clause"
)

// Repository manages all CRUD operations for model T.
// It is type-safe, leveraging Go generics for compile-time type checking.
//
// Repository design principles:
//   - Immutability: Where() and other methods return new Repository instances, not modifying original instances
//   - Composability: Build complex queries through method chaining
//   - Lifecycle hooks: Automatically triggers BeforeCreate/AfterCreate and other hooks
//   - Soft delete support: Automatically detects and supports soft delete functionality
//
// Usage example:
//
//	// Create Repository
//	userRepo := sqlc.NewRepository[models.User](session)
//
//	// Create record
//	user := &models.User{Email: "test@example.com", Name: "Test User"}
//	if err := userRepo.Create(ctx, user); err != nil {
//	    return err
//	}
//	fmt.Println("Created user ID:", user.ID) // Auto-increment ID backfilled
//
//	// Query record
//	user, err := userRepo.FindOne(ctx, 1)
//
//	// Update record
//	user.Name = "New Name"
//	if err := userRepo.Update(ctx, user); err != nil {
//	    return err
//	}
//
//	// Delete record
//	if err := userRepo.Delete(ctx, 1); err != nil {
//	    return err
//	}
type Repository[T any] struct {
	session  *Session            // Database session
	schema   Schema[T]           // Model's Schema implementation
	scopes   []clause.Expression // Query condition scopes
	unscoped bool                // Whether to bypass soft delete
}

// NewRepository creates a new Repository instance.
// This is the entry point for using Repository.
//
// Parameters:
//   - session: Database session, can be regular session or transaction session
//
// Type parameter:
//   - T: Model type, must be registered via RegisterSchema
//
// Returns:
//   - *Repository[T]: Initialized Repository instance
//
// Note:
//   - Model T must be registered via RegisterSchema[T]()
//   - If not registered, LoadSchema[T]() will panic
//
// Example:
//
//	// Basic usage
//	userRepo := sqlc.NewRepository[models.User](session)
//
//	// Use in transaction
//	err := session.Transaction(ctx, func(txSession *Session) error {
//	    txUserRepo := sqlc.NewRepository[models.User](txSession)
//	    return txUserRepo.Create(ctx, user)
//	})
func NewRepository[T any](session *Session) *Repository[T] {
	return &Repository[T]{
		session: session,
		schema:  LoadSchema[T](),
		scopes:  make([]clause.Expression, 0),
	}
}

// Where returns a new Repository instance with appended conditions.
// This allows method chaining, e.g., repo.Where(cond).Update(...)
//
// Important: This method returns a new Repository instance, not modifying the original instance.
// This ensures Repository immutability and thread safety.
//
// Parameters:
//   - conds: Query condition expressions (variadic)
//
// Returns:
//   - *Repository[T]: New Repository instance with appended conditions
//
// Use cases:
//   - Conditional update: repo.Where(active.Eq(true)).Update(ctx, user)
//   - Conditional delete: repo.Where(old.Eq(true)).Delete(ctx, id)
//   - Batch operations: repo.Where(status.Eq("pending")).UpdateColumns(ctx, nil, ...)
//
// Example:
//
//	// Conditional update
//	affected, err := userRepo.
//	    Where(generated.User.Status.Eq("inactive")).
//	    Where(generated.User.LastLoginAt.Lt(time.Now().AddDate(0, -6, 0))).
//	    UpdateColumns(ctx, nil,
//	        clause.Assignment{Column: generated.User.Status.Column(), Value: "archived"},
//	    )
//
//	// Conditional delete
//	err := orderRepo.
//	    Where(generated.Order.Status.Eq("cancelled")).
//	    Delete(ctx, orderID)
func (r *Repository[T]) Where(conds ...clause.Expression) *Repository[T] {
	// Create new Repository instance (value copy)
	newRepo := *r
	// Copy scopes slice, avoid sharing underlying array
	newRepo.scopes = append(newRepo.scopes, conds...)
	return &newRepo
}

// Unscoped returns a new Repository instance that bypasses soft delete.
// When unscoped is set to true, Delete() and DeleteModel() will perform hard delete
// even if the model supports soft delete.
//
// Example:
//
//	err := userRepo.Unscoped().Delete(ctx, userID)
func (r *Repository[T]) Unscoped() *Repository[T] {
	newRepo := *r
	newRepo.unscoped = true
	return &newRepo
}

// Create inserts a new record into the database.
// This is the recommended way to create a single record.
//
// Operation flow:
//  1. Trigger BeforeCreate hook (if model implements BeforeCreateInterface)
//  2. Extract insert data from model (via schema.InsertRow)
//  3. Execute INSERT statement
//  4. If auto-increment primary key, backfill ID to model
//  5. Trigger AfterCreate hook (if model implements AfterCreateInterface)
//
// Parameters:
//   - ctx: Context, supports cancellation and timeout
//   - model: Model instance pointer, may be modified after insertion (auto-increment ID backfilled)
//
// Returns:
//   - error: Insertion error or hook error
//
// Hook support:
//   - BeforeCreate: Called before insertion, can be used for validation or setting default values
//   - AfterCreate: Called after insertion, can be used for logging or cascade operations
//
// Example:
//
//	user := &models.User{
//	    Email: "test@example.com",
//	    Name:  "Test User",
//	}
//
//	if err := userRepo.Create(ctx, user); err != nil {
//	    return err
//	}
//
//	fmt.Println("Created user ID:", user.ID) // Auto-increment ID backfilled
func (r *Repository[T]) Create(ctx context.Context, model *T) error {
	// Trigger BeforeCreate hook
	if err := triggerBeforeCreate(ctx, model); err != nil {
		return err
	}

	// Extract insert data from model
	cols, vals := r.schema.InsertRow(model)

	// Build INSERT statement
	builder := sq.Insert(r.schema.TableName()).
		Columns(cols...).
		Values(vals...).
		PlaceholderFormat(r.session.dialect.PlaceholderFormat())

	// Generate SQL
	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	// Execute insertion
	result, err := r.session.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	// If auto-increment primary key, backfill ID
	if r.schema.AutoIncrement() {
		id, err := result.LastInsertId()
		if err == nil {
			r.schema.SetPK(model, id)
		}
	}

	// Trigger AfterCreate hook
	return triggerAfterCreate(ctx, model)
}

// BatchCreate inserts multiple records in a single SQL statement.
// This is more efficient than calling Create() in a loop, suitable for batch import scenarios.
//
// Operation flow:
//  1. Trigger BeforeCreate hook for each model
//  2. Build batch INSERT statement (single SQL, multiple VALUES)
//  3. Execute batch insertion
//  4. Trigger AfterCreate hook for each model
//
// Parameters:
//   - ctx: Context, supports cancellation and timeout
//   - models: Model instance pointer slice
//
// Returns:
//   - error: Insertion error or hook error
//
// Note:
//   - Empty slice will immediately return nil (no-op)
//   - Auto-increment IDs will not be backfilled to models (database limitation)
//   - If any hook fails, entire operation aborts
//   - Does not support partial rollback within transaction (should be called outside transaction)
//
// Performance suggestions:
//   - For large amounts of data (>1000 records), consider calling in batches
//   - Database may have single SQL size limits
//
// Example:
//
//	users := []*models.User{
//	    {Email: "user1@example.com", Name: "User 1"},
//	    {Email: "user2@example.com", Name: "User 2"},
//	    {Email: "user3@example.com", Name: "User 3"},
//	}
//
//	if err := userRepo.BatchCreate(ctx, users); err != nil {
//	    return err
//	}
//
//	// Note: users[i].ID will not be set
func (r *Repository[T]) BatchCreate(ctx context.Context, models []*T) error {
	// Empty slice fast return
	if len(models) == 0 {
		return nil
	}

	// Trigger BeforeCreate hook for all models
	for _, model := range models {
		if err := triggerBeforeCreate(ctx, model); err != nil {
			return err
		}
	}

	// Build batch INSERT statement
	builder := sq.Insert(r.schema.TableName()).
		PlaceholderFormat(r.session.dialect.PlaceholderFormat())

	// Add each row of data
	for i, model := range models {
		cols, vals := r.schema.InsertRow(model)
		if i == 0 {
			// First row sets column names
			builder = builder.Columns(cols...)
		}
		// Add values
		builder = builder.Values(vals...)
	}

	// Generate and execute SQL
	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	_, err = r.session.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	// Note: Cannot easily get all auto-increment IDs for batch insert across all databases
	// For MVP version, we skip updating model IDs

	// Trigger AfterCreate hook for all models
	for _, model := range models {
		if err := triggerAfterCreate(ctx, model); err != nil {
			return err
		}
	}
	return nil
}

// Upsert Options
type upsertConfig struct {
	conflictCols []string // Conflict detection columns (unique constraint or primary key)
	updateCols   []string // Columns to update when conflict occurs
}

// UpsertOption defines configuration function for Upsert operation.
// Uses functional options pattern to provide flexible configuration.
type UpsertOption func(*upsertConfig)

// OnConflict specifies columns to check for conflict (e.g., PRIMARY KEY or UNIQUE constraints)
//
// Parameters:
//   - columns: Conflict detection columns (implements clause.Columnar interface)
//
// Returns:
//   - UpsertOption: Configuration function
//
// Default behavior:
//   - If this option is not called, primary key column is used by default
//
// Example:
//
//	// Use email unique constraint to detect conflict
//	err := userRepo.Upsert(ctx, user,
//	    sqlc.OnConflict(generated.User.Email),
//	)
//
//	// Use composite unique constraint
//	err := orderRepo.Upsert(ctx, order,
//	    sqlc.OnConflict(generated.Order.UserID, generated.Order.ProductID),
//	)
func OnConflict(columns ...clause.Columnar) UpsertOption {
	return func(c *upsertConfig) {
		c.conflictCols = ResolveColumnNames(columns)
	}
}

// DoUpdate specifies which columns to update when a conflict occurs.
// If not specified, all model columns (except conflict columns) are updated.
//
// Parameters:
//   - columns: Columns to update (implements clause.Columnar interface)
//
// Returns:
//   - UpsertOption: Configuration function
//
// Default behavior:
//   - If this option is not called, all non-conflict columns are updated
//
// Example:
//
//	// Only update name and updated_at when conflict occurs
//	err := userRepo.Upsert(ctx, user,
//	    sqlc.OnConflict(generated.User.Email),
//	    sqlc.DoUpdate(generated.User.Name, generated.User.UpdatedAt),
//	)
//
//	// Don't update any columns when conflict occurs (DO NOTHING)
//	err := userRepo.Upsert(ctx, user,
//	    sqlc.OnConflict(generated.User.Email),
//	    sqlc.DoUpdate(), // Empty parameters
//	)
func DoUpdate(columns ...clause.Columnar) UpsertOption {
	return func(c *upsertConfig) {
		c.updateCols = ResolveColumnNames(columns)
	}
}

// Upsert inserts or updates a record.
// By default, it uses the Primary Key as the conflict target and updates all other columns.
// You can customize this utilizing OnConflict() and DoUpdate() options.
//
// Database dialect differences:
//   - MySQL: ON DUPLICATE KEY UPDATE
//   - PostgreSQL: ON CONFLICT (...) DO UPDATE SET
//   - SQLite: ON CONFLICT (...) DO UPDATE SET
//
// Operation flow:
//  1. Trigger BeforeCreate hook
//  2. Determine conflict columns (default is primary key)
//  3. Determine update columns (default is all non-conflict columns)
//  4. Build INSERT ... ON CONFLICT statement
//  5. Execute statement
//  6. Trigger AfterCreate hook
//
// Parameters:
//   - ctx: Context, supports cancellation and timeout
//   - model: Model instance pointer
//   - opts: Optional configuration (OnConflict, DoUpdate)
//
// Returns:
//   - error: Insert/update error or hook error
//
// Example:
//
//	// Basic usage (use primary key to detect conflict)
//	err := userRepo.Upsert(ctx, user)
//
//	// Use unique constraint to detect conflict
//	err := userRepo.Upsert(ctx, user,
//	    sqlc.OnConflict(generated.User.Email),
//	)
//
//	// Specify columns to update
//	err := userRepo.Upsert(ctx, user,
//	    sqlc.OnConflict(generated.User.Email),
//	    sqlc.DoUpdate(generated.User.Name, generated.User.LastLoginAt),
//	)
func (r *Repository[T]) Upsert(ctx context.Context, model *T, opts ...UpsertOption) error {
	// Apply configuration options
	config := &upsertConfig{}
	for _, opt := range opts {
		opt(config)
	}

	// Trigger BeforeCreate hook
	if err := triggerBeforeCreate(ctx, model); err != nil {
		return err
	}

	// Extract data from model
	cols, vals := r.schema.InsertRow(model)

	// Determine Conflict Columns (Default: PK Column)
	conflictCols := config.conflictCols
	if len(conflictCols) == 0 {
		pk := r.schema.PK(nil)
		conflictCols = []string{pk.Column.Name}
	}

	// Determine Update Columns (Default: All Cols - Conflict Cols)
	updateCols := config.updateCols
	if len(updateCols) == 0 {
		// Filter out conflict columns from all columns
		for _, col := range cols {
			if !slices.Contains(conflictCols, col) {
				updateCols = append(updateCols, col)
			}
		}
	}

	// Get dialect-specific Upsert clause
	upsertClause := r.session.dialect.UpsertClause(r.schema.TableName(), conflictCols, updateCols)

	// Build INSERT ... ON CONFLICT statement
	builder := sq.Insert(r.schema.TableName()).
		Columns(cols...).
		Values(vals...).
		Suffix(upsertClause).
		PlaceholderFormat(r.session.dialect.PlaceholderFormat())

	// Generate and execute SQL
	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	_, err = r.session.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	// Trigger AfterCreate hook
	return triggerAfterCreate(ctx, model)
}

// Update updates a record in the database.
// Locates record by model's primary key, updates all updatable fields.
//
// Operation flow:
//  1. Trigger BeforeUpdate hook (if model implements BeforeUpdateInterface)
//  2. Extract update data from model (via schema.UpdateMap)
//  3. Build UPDATE statement with primary key condition
//  4. Apply all scope conditions (set via Where)
//  5. Execute update
//  6. Trigger AfterUpdate hook (if model implements AfterUpdateInterface)
//
// Parameters:
//   - ctx: Context, supports cancellation and timeout
//   - model: Model instance pointer, must contain valid primary key value
//
// Returns:
//   - error: Update error or hook error
//
// Note:
//   - Model must have valid primary key value
//   - Scope conditions (Where) will be combined with primary key condition
//   - Empty UpdateMap will result in UPDATE with no actual changes
//
// Example:
//
//	// Basic update
//	user.Name = "New Name"
//	if err := userRepo.Update(ctx, user); err != nil {
//	    return err
//	}
//
//	// Update with scope
//	user.Status = "archived"
//	if err := userRepo.Where(generated.User.Status.Eq("inactive")).Update(ctx, user); err != nil {
//	    return err
//	}
func (r *Repository[T]) Update(ctx context.Context, model *T) error {
	// Trigger BeforeUpdate hook
	if err := triggerBeforeUpdate(ctx, model); err != nil {
		return err
	}

	// Extract update data from model
	setMap := r.schema.UpdateMap(model)
	pk := r.schema.PK(model)

	// Build UPDATE statement
	builder := sq.Update(r.schema.TableName()).
		SetMap(setMap).
		Where(sq.Eq{pk.Column.Name: pk.Value})

	// Apply Scopes
	for _, scope := range r.scopes {
		builder = builder.Where(scope)
	}

	builder = builder.PlaceholderFormat(r.session.dialect.PlaceholderFormat())

	// Generate and execute SQL
	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	_, err = r.session.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	// Trigger AfterUpdate hook
	return triggerAfterUpdate(ctx, model)
}

// UpdateColumns updates specific columns for a record identified by id.
// This allows for partial updates without loading the entire record.
//
// Difference from Update():
//   - Update(): Requires complete model instance, updates all fields
//   - UpdateColumns(): Only needs ID, updates specified fields
//
// Parameters:
//   - ctx: Context, supports cancellation and timeout
//   - id: Record's primary key value
//   - assignments: Column assignment list (column = value)
//
// Returns:
//   - error: Update error
//
// Note:
//   - Empty assignments will immediately return nil (no-op)
//   - Does not trigger lifecycle hooks (no complete model instance)
//   - Scope conditions will be combined with primary key condition
//
// Example:
//
//	// Update single field
//	err := userRepo.UpdateColumns(ctx, userID,
//	    clause.Assignment{
//	        Column: generated.User.Status.Column(),
//	        Value:  "active",
//	    },
//	)
//
//	// Update multiple fields
//	err := userRepo.UpdateColumns(ctx, userID,
//	    clause.Assignment{Column: generated.User.Name.Column(), Value: "New Name"},
//	    clause.Assignment{Column: generated.User.UpdatedAt.Column(), Value: time.Now()},
//	)
//
//	// Conditional update with scope
//	err := userRepo.
//	    Where(generated.User.Status.Eq("pending")).
//	    UpdateColumns(ctx, userID,
//	        clause.Assignment{Column: generated.User.Status.Column(), Value: "processed"},
//	    )
func (r *Repository[T]) UpdateColumns(ctx context.Context, id any, assignments ...clause.Assignment) error {
	// Empty assignment fast return
	if len(assignments) == 0 {
		return nil
	}

	// Get primary key metadata
	pkMeta := r.schema.PK(nil)

	// Build UPDATE statement
	builder := sq.Update(r.schema.TableName()).
		Where(sq.Eq{pkMeta.Column.Name: id})

	// Apply Scopes
	for _, scope := range r.scopes {
		builder = builder.Where(scope)
	}

	builder = builder.PlaceholderFormat(r.session.dialect.PlaceholderFormat())

	// Add column assignments
	for _, assignment := range assignments {
		builder = builder.Set(assignment.Column.ColumnName(), assignment.Value)
	}

	// Generate and execute SQL
	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	_, err = r.session.Exec(ctx, query, args...)
	return err
}

// Delete deletes a record by primary key.
// Performs hard delete, record will be permanently removed from database.
//
// Parameters:
//   - ctx: Context, supports cancellation and timeout
//   - id: Record's primary key value
//
// Returns:
//   - error: Deletion error
//
// Note:
//   - This is hard delete, record will be permanently removed
//   - Does not trigger lifecycle hooks (no model instance)
//   - For soft delete models, recommend using SoftDelete()
//   - Scope conditions will be combined with primary key condition
//
// Example:
//
//	// Basic delete
//	if err := userRepo.Delete(ctx, userID); err != nil {
//	    return err
//	}
//
//	// Conditional delete
//	if err := userRepo.
//	    Where(generated.User.Status.Eq("inactive")).
//	    Delete(ctx, userID); err != nil {
//	    return err
//	}
func (r *Repository[T]) Delete(ctx context.Context, id any) error {
	// Check if model supports soft delete and we are not in unscoped mode
	sdCol := r.schema.SoftDeleteColumn()
	if sdCol != "" && !r.unscoped {
		// Perform soft delete
		sdVal := r.schema.SoftDeleteValue()
		return r.UpdateColumns(ctx, id, clause.Assignment{
			Column: clause.Column{Name: sdCol},
			Value:  sdVal,
		})
	}

	// Get primary key metadata
	pkMeta := r.schema.PK(nil)

	// Build DELETE statement
	builder := sq.Delete(r.schema.TableName()).
		Where(sq.Eq{pkMeta.Column.Name: id})

	// Apply Scopes
	for _, scope := range r.scopes {
		builder = builder.Where(scope)
	}

	builder = builder.PlaceholderFormat(r.session.dialect.PlaceholderFormat())

	// Generate and execute SQL
	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	_, err = r.session.Exec(ctx, query, args...)
	return err
}

// DeleteModel deletes a record by model instance, triggering lifecycle hooks.
// Use when you need to execute custom logic before/after deletion.
//
// Operation flow:
//  1. Trigger BeforeDelete hook (if model implements BeforeDeleteInterface)
//  2. Extract primary key from model
//  3. Build DELETE statement with primary key condition
//  4. Apply all scope conditions
//  5. Execute deletion
//  6. Trigger AfterDelete hook (if model implements AfterDeleteInterface)
//
// Parameters:
//   - ctx: Context, supports cancellation and timeout
//   - model: Model instance pointer, must contain valid primary key value
//
// Returns:
//   - error: Deletion error or hook error
//
// Example:
//
//	// Query first then delete (supports hooks)
//	user, err := userRepo.FindOne(ctx, userID)
//	if err != nil {
//	    return err
//	}
//
//	if err := userRepo.DeleteModel(ctx, user); err != nil {
//	    return err
//	}
func (r *Repository[T]) DeleteModel(ctx context.Context, model *T) error {
	// Trigger BeforeDelete hook
	if err := triggerBeforeDelete(ctx, model); err != nil {
		return err
	}

	// Check if model supports soft delete and we are not in unscoped mode
	sdCol := r.schema.SoftDeleteColumn()
	if sdCol != "" && !r.unscoped {
		// Extract primary key from model
		pk := r.schema.PK(model)
		sdVal := r.schema.SoftDeleteValue()

		// Build UPDATE statement, set soft delete column
		builder := sq.Update(r.schema.TableName()).
			Set(sdCol, sdVal).
			Where(sq.Eq{pk.Column.Name: pk.Value}).
			PlaceholderFormat(r.session.dialect.PlaceholderFormat())

		// Apply Scopes
		for _, scope := range r.scopes {
			builder = builder.Where(scope)
		}

		// Generate and execute SQL
		query, args, err := builder.ToSql()
		if err != nil {
			return err
		}

		_, err = r.session.Exec(ctx, query, args...)
		if err != nil {
			return err
		}

		// Sync model instance's soft delete field
		r.schema.SetDeletedAt(model)

		// Trigger AfterDelete hook
		return triggerAfterDelete(ctx, model)
	}

	// Extract primary key from model
	pk := r.schema.PK(model)

	// Build DELETE statement
	builder := sq.Delete(r.schema.TableName()).
		Where(sq.Eq{pk.Column.Name: pk.Value})

	// Apply Scopes
	for _, scope := range r.scopes {
		builder = builder.Where(scope)
	}

	builder = builder.PlaceholderFormat(r.session.dialect.PlaceholderFormat())

	// Generate and execute SQL
	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	_, err = r.session.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	// Trigger AfterDelete hook
	return triggerAfterDelete(ctx, model)
}

// Query returns a QueryBuilder for building complex queries.
// This is the starting point for building queries, supports method chaining.
//
// Returns:
//   - *QueryBuilder[T]: Query builder
//
// Query features:
//   - Automatically applies soft delete filter (if model supports)
//   - Supports conditions, sorting, pagination, aggregation
//   - Supports relation preloading
//   - Supports subqueries
//
// Example:
//
//	// Basic query
//	users, err := userRepo.Query().Find(ctx)
//
//	// Conditional query
//	activeUsers, err := userRepo.Query().
//	    Where(generated.User.Status.Eq("active")).
//	    OrderBy(generated.User.CreatedAt.Desc()).
//	    Find(ctx)
//
//	// Paginated query
//	users, err := userRepo.Query().
//	    Limit(10).
//	    Offset(20).
//	    Find(ctx)
//
//	// Aggregation query
//	count, err := userRepo.Query().
//	    Where(generated.User.Status.Eq("active")).
//	    Count(ctx)
func (r *Repository[T]) Query() *QueryBuilder[T] {
	return Query[T](r.session)
}

// FindOne queries a single record by primary key.
// This is a shortcut for getting a record by ID.
//
// Parameters:
//   - ctx: Context, supports cancellation and timeout
//   - id: Record's primary key value
//
// Returns:
//   - *T: Found model instance
//   - error: Query error (ErrNotFound indicates not found)
//
// Note:
//   - Automatically applies soft delete filter
//   - Scope conditions will be combined with primary key condition
//   - If record not found, returns ErrNotFound
//
// Example:
//
//	user, err := userRepo.FindOne(ctx, 123)
//	if err != nil {
//	    if errors.Is(err, sqlc.ErrNotFound) {
//	        // User not found
//	        return nil
//	    }
//	    return err
//	}
//	fmt.Println("User:", user.Name)
func (r *Repository[T]) FindOne(ctx context.Context, id any) (*T, error) {
	// Get primary key metadata
	pkMeta := r.schema.PK(nil)
	query := r.Query().Where(clause.Eq{Column: pkMeta.Column, Value: id})

	// Apply Scopes to Query
	for _, scope := range r.scopes {
		query = query.Where(scope)
	}
	return query.First(ctx)
}

// Restore restores a soft-deleted record by clearing the soft delete marker.
// Returns an error if the model doesn't support soft delete.
//
// Parameters:
//   - ctx: Context, supports cancellation and timeout
//   - id: Record's primary key value
//
// Returns:
//   - error: Restore error, returns error if model doesn't support soft delete
//
// Note:
//   - Only effective for models supporting soft delete
//   - Scope conditions will be combined with primary key condition
//   - If record was not deleted, operation still succeeds (no effect)
//
// Example:
//
//	// Restore deleted user
//	if err := userRepo.Restore(ctx, userID); err != nil {
//	    return err
//	}
//
//	// Now user can be queried normally
//	user, err := userRepo.FindOne(ctx, userID)
func (r *Repository[T]) Restore(ctx context.Context, id any) error {
	// Check if model supports soft delete
	sdCol := r.schema.SoftDeleteColumn()
	if sdCol == "" {
		return fmt.Errorf("sqlc: model does not support soft delete")
	}

	// Get primary key metadata
	pkMeta := r.schema.PK(nil)

	// Build UPDATE statement, clear soft delete marker
	builder := sq.Update(r.schema.TableName()).
		Set(sdCol, nil).
		Where(sq.Eq{pkMeta.Column.Name: id}).
		PlaceholderFormat(r.session.dialect.PlaceholderFormat())

	// Apply Scopes
	for _, scope := range r.scopes {
		builder = builder.Where(scope)
	}

	// Generate and execute SQL
	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	_, err = r.session.Exec(ctx, query, args...)
	return err
}

// FirstOrCreate returns the first matching record, or creates one with defaults.
// This is the recommended way to implement "find or create" pattern.
//
// Operation flow:
//  1. Build query with scope conditions
//  2. Try to find record
//  3. If found, return record
//  4. If not found, create record using defaults
//  5. Trigger BeforeCreate/AfterCreate hooks
//
// Parameters:
//   - ctx: Context, supports cancellation and timeout
//   - defaults: Default value model, used to create new record
//
// Returns:
//   - *T: Found or created model instance
//   - error: Query or creation error
//
// Note:
//   - Scope conditions are used to find record
//   - defaults should contain all necessary fields
//   - Creation will trigger lifecycle hooks
//   - Auto-increment ID will be backfilled to defaults
//
// Example:
//
//	// Find or create user
//	user, err := userRepo.
//	    Where(generated.User.Email.Eq("test@example.com")).
//	    FirstOrCreate(ctx, &models.User{
//	        Email: "test@example.com",
//	        Name:  "New User",
//	    })
//
//	if err != nil {
//	    return err
//	}
//
//	// user could be existing user or newly created user
//	fmt.Println("User ID:", user.ID)
func (r *Repository[T]) FirstOrCreate(ctx context.Context, defaults *T) (*T, error) {
	// Build query
	query := r.Query()

	// Apply Scopes to Query
	for _, scope := range r.scopes {
		query = query.Where(scope)
	}

	// Try to find record
	result, err := query.Take(ctx)
	if err == nil {
		// Found record, return directly
		return result, nil
	}

	// Check if it's "not found" error
	if errors.Is(err, ErrNotFound) {
		// Create new record with defaults
		if err := r.Create(ctx, defaults); err != nil {
			return nil, fmt.Errorf("sqlc: first or create failed: %w", err)
		}
		return defaults, nil
	}

	// Other errors
	return nil, err
}
