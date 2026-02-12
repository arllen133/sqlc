// Package sqlc provides a type-safe ORM library using generics and code generation.
// This file implements the model lifecycle hooks mechanism.
//
// Lifecycle hooks allow executing custom logic at critical moments of database operations, such as:
//   - Data validation: Validate data integrity before saving
//   - Auto fields: Automatically set timestamps like created_at, updated_at
//   - Data transformation: Transform data format before saving or after loading
//   - Audit logging: Record data change history
//   - Cache management: Clear related caches after changes
//   - Cascade operations: Automatically handle related data
//
// Hook execution order:
//   - Create: BeforeCreate → INSERT → AfterCreate
//   - Update: BeforeUpdate → UPDATE → AfterUpdate
//   - Delete: BeforeDelete → DELETE → AfterDelete
//
// Usage example:
//
//	type User struct {
//	    ID        int64     `db:"id,primaryKey"`
//	    Email     string    `db:"email"`
//	    Name      string    `db:"name"`
//	    CreatedAt time.Time `db:"created_at"`
//	    UpdatedAt time.Time `db:"updated_at"`
//	}
//
//	// Implement BeforeCreate hook
//	func (u *User) BeforeCreate(ctx context.Context) error {
//	    // Validate email format
//	    if !strings.Contains(u.Email, "@") {
//	        return errors.New("invalid email format")
//	    }
//	    // Set creation time
//	    u.CreatedAt = time.Now()
//	    u.UpdatedAt = time.Now()
//	    return nil
//	}
//
//	// Implement BeforeUpdate hook
//	func (u *User) BeforeUpdate(ctx context.Context) error {
//	    // Auto-update modification time
//	    u.UpdatedAt = time.Now()
//	    return nil
//	}
package sqlc

import (
	"context"
)

// BeforeCreateInterface defines the hook interface for before creation.
// If a model implements this interface, Create() and BatchCreate() methods will call BeforeCreate() before insertion.
//
// Use cases:
//   - Data validation: Validate required fields, format validation, business rule checks
//   - Auto fields: Set created_at, generate UUIDs, calculate default values
//   - Data transformation: Encrypt passwords, normalize data formats
//   - Auditing: Record creation operations
//
// Notes:
//   - If error is returned, creation operation will be aborted
//   - Should not execute potentially failing network operations (e.g., API calls)
//   - Should not execute database operations that might cause recursion
//   - Should execute quickly to avoid blocking
//
// Example:
//
//	type User struct {
//	    ID           int64      `db:"id,primaryKey"`
//	    Email        string     `db:"email"`
//	    PasswordHash string     `db:"password_hash"`
//	    CreatedAt    time.Time  `db:"created_at"`
//	}
//
//	func (u *User) BeforeCreate(ctx context.Context) error {
//	    // Validate email
//	    if u.Email == "" {
//	        return errors.New("email is required")
//	    }
//
//	    // Encrypt password (should use bcrypt in production)
//	    if u.PasswordHash != "" {
//	        hashed, err := bcrypt.GenerateFromPassword([]byte(u.PasswordHash), bcrypt.DefaultCost)
//	        if err != nil {
//	            return err
//	        }
//	        u.PasswordHash = string(hashed)
//	    }
//
//	    // Set creation time
//	    u.CreatedAt = time.Now()
//
//	    return nil
//	}
type BeforeCreateInterface interface {
	BeforeCreate(context.Context) error
}

// AfterCreateInterface defines the hook interface for after creation.
// If a model implements this interface, Create() and BatchCreate() methods will call AfterCreate() after successful insertion.
//
// Use cases:
//   - Audit logging: Record creation operations to log system
//   - Cache management: Warm up cache, send cache invalidation notifications
//   - Notifications: Send welcome emails, notify administrators
//   - Cascade creation: Create related records (use with caution)
//   - Search indexing: Update search engine index
//
// Notes:
//   - Executes within transaction, if it fails the transaction will rollback
//   - If error is returned, the entire creation operation will rollback
//   - Should not execute database operations that might cause recursion
//   - For BatchCreate, called for each model
//
// Example:
//
//	type Order struct {
//	    ID      int64 `db:"id,primaryKey"`
//	    UserID  int64 `db:"user_id"`
//	    Amount  int64 `db:"amount"`
//	    Status  string `db:"status"`
//	}
//
//	func (o *Order) AfterCreate(ctx context.Context) error {
//	    // Send order creation notification
//	    notification := &Notification{
//	        UserID:  o.UserID,
//	        Type:    "order_created",
//	        Message: fmt.Sprintf("Order #%d has been created", o.ID),
//	    }
//	    // Note: Need to access Repository via context or use global service
//	    return notificationService.Send(ctx, notification)
//	}
type AfterCreateInterface interface {
	AfterCreate(context.Context) error
}

// BeforeUpdateInterface defines the hook interface for before update.
// If a model implements this interface, Update() method will call BeforeUpdate() before updating.
//
// Use cases:
//   - Data validation: Validate update data legality
//   - Auto fields: Update updated_at, record modifier
//   - Change tracking: Record which fields were modified
//   - Condition checking: Ensure business rules (e.g., order status transitions)
//   - Data transformation: Encrypt sensitive fields
//
// Notes:
//   - If error is returned, update operation will be aborted
//   - Should not execute database operations that might cause recursion
//   - Not triggered for UpdateColumns() (no complete model instance)
//
// Example:
//
//	type User struct {
//	    ID        int64     `db:"id,primaryKey"`
//	    Email     string    `db:"email"`
//	    Name      string    `db:"name"`
//	    UpdatedAt time.Time `db:"updated_at"`
//	}
//
//	func (u *User) BeforeUpdate(ctx context.Context) error {
//	    // Auto-update modification time
//	    u.UpdatedAt = time.Now()
//
//	    // Validate email format
//	    if u.Email != "" && !strings.Contains(u.Email, "@") {
//	        return errors.New("invalid email format")
//	    }
//
//	    return nil
//	}
//
// Order status transition example:
//
//	type Order struct {
//	    ID     int64 `db:"id,primaryKey"`
//	    Status string `db:"status"`
//	}
//
//	func (o *Order) BeforeUpdate(ctx context.Context) error {
//	    // Load original order from database
//	    original, err := orderRepo.FindOne(ctx, o.ID)
//	    if err != nil {
//	        return err
//	    }
//
//	    // Validate status transition
//	    validTransitions := map[string][]string{
//	        "pending":   {"paid", "cancelled"},
//	        "paid":      {"shipped", "refunded"},
//	        "shipped":   {"delivered", "returned"},
//	        "delivered": {"returned"},
//	    }
//
//	    allowed, ok := validTransitions[original.Status]
//	    if !ok {
//	        return fmt.Errorf("invalid current status: %s", original.Status)
//	    }
//
//	    for _, s := range allowed {
//	        if s == o.Status {
//	            return nil // Valid status transition
//	        }
//	    }
//
//	    return fmt.Errorf("cannot transition from %s to %s", original.Status, o.Status)
//	}
type BeforeUpdateInterface interface {
	BeforeUpdate(context.Context) error
}

// AfterUpdateInterface defines the hook interface for after update.
// If a model implements this interface, Update() method will call AfterUpdate() after successful update.
//
// Use cases:
//   - Audit logging: Record data changes
//   - Cache management: Clear related caches
//   - Notifications: Notify relevant parties of data changes
//   - Search indexing: Update search engine index
//   - Status change handling: Handle state machine transition side effects
//
// Notes:
//   - Executes within transaction, if it fails the transaction will rollback
//   - If error is returned, the entire update operation will rollback
//   - Should not execute database operations that might cause recursion
//   - Not triggered for UpdateColumns() (no complete model instance)
//
// Example:
//
//	type Product struct {
//	    ID          int64   `db:"id,primaryKey"`
//	    Name        string  `db:"name"`
//	    Price       float64 `db:"price"`
//	    Stock       int     `db:"stock"`
//	}
//
//	func (p *Product) AfterUpdate(ctx context.Context) error {
//	    // Clear product cache
//	    cache.Delete(fmt.Sprintf("product:%d", p.ID))
//
//	    // If price changed, record price history
//	    // Note: Need to get original price from context or other means
//	    priceHistory := &PriceHistory{
//	        ProductID: p.ID,
//	        Price:     p.Price,
//	        ChangedAt: time.Now(),
//	    }
//	    return priceHistoryRepo.Create(ctx, priceHistory)
//	}
type AfterUpdateInterface interface {
	AfterUpdate(context.Context) error
}

// BeforeDeleteInterface defines the hook interface for before delete.
// If a model implements this interface, DeleteModel() method will call BeforeDelete() before deletion.
//
// Use cases:
//   - Data validation: Check if deletion is allowed (e.g., if order is completed)
//   - Cascade deletion: Delete related data
//   - Archiving: Archive data before deletion
//   - Auditing: Record deletion operations
//   - Cleanup: Clean up related resources (files, caches, etc.)
//
// Notes:
//   - If error is returned, deletion operation will be aborted
//   - Not triggered for Delete() (no model instance)
//   - Not triggered for SoftDelete() (no model instance)
//   - Should not execute database operations that might cause recursion
//
// Example:
//
//	type User struct {
//	    ID       int64  `db:"id,primaryKey"`
//	    Email    string `db:"email"`
//	    AvatarURL string `db:"avatar_url"`
//	}
//
//	func (u *User) BeforeDelete(ctx context.Context) error {
//	    // Check if user has pending orders
//	    count, err := orderRepo.Query().
//	        Where(generated.Order.UserID.Eq(u.ID)).
//	        Where(generated.Order.Status.Ne("completed")).
//	        Count(ctx)
//	    if err != nil {
//	        return err
//	    }
//	    if count > 0 {
//	        return errors.New("cannot delete user with pending orders")
//	    }
//
//	    // Delete user avatar file
//	    if u.AvatarURL != "" {
//	        storage.Delete(u.AvatarURL)
//	    }
//
//	    return nil
//	}
//
// Cascade deletion example:
//
//	type Order struct {
//	    ID int64 `db:"id,primaryKey"`
//	}
//
//	func (o *Order) BeforeDelete(ctx context.Context) error {
//	    // Delete all order items
//	    orderItems, err := orderItemRepo.Query().
//	        Where(generated.OrderItem.OrderID.Eq(o.ID)).
//	        Find(ctx)
//	    if err != nil {
//	        return err
//	    }
//
//	    for _, item := range orderItems {
//	        if err := orderItemRepo.DeleteModel(ctx, item); err != nil {
//	            return err
//	        }
//	    }
//
//	    return nil
//	}
type BeforeDeleteInterface interface {
	BeforeDelete(context.Context) error
}

// AfterDeleteInterface defines the hook interface for after delete.
// If a model implements this interface, DeleteModel() method will call AfterDelete() after successful deletion.
//
// Use cases:
//   - Audit logging: Record deletion operations
//   - Cache management: Clear related caches
//   - Notifications: Notify relevant parties of data deletion
//   - Search indexing: Remove index from search engine
//   - Statistics update: Update counters
//
// Notes:
//   - Executes within transaction, if it fails the transaction will rollback
//   - If error is returned, the entire deletion operation will rollback
//   - Not triggered for Delete() (no model instance)
//   - For SoftDelete(), use SoftDeleteModel
//   - Database record is already deleted at this point, cannot query anymore
//
// Example:
//
//	type Document struct {
//	    ID      int64  `db:"id,primaryKey"`
//	    Title   string `db:"title"`
//	    OwnerID int64  `db:"owner_id"`
//	}
//
//	func (d *Document) AfterDelete(ctx context.Context) error {
//	    // Remove index from search engine
//	    searchService.DeleteDocument(ctx, d.ID)
//
//	    // Clear related caches
//	    cache.Delete(fmt.Sprintf("document:%d", d.ID))
//	    cache.Delete(fmt.Sprintf("user:%d:documents", d.OwnerID))
//
//	    // Record audit log
//	    auditLog := &AuditLog{
//	        Action:    "delete",
//	        Entity:    "document",
//	        EntityID:  d.ID,
//	        Timestamp: time.Now(),
//	    }
//	    return auditLogRepo.Create(ctx, auditLog)
//	}
type AfterDeleteInterface interface {
	AfterDelete(context.Context) error
}

// triggerBeforeCreate triggers the BeforeCreate hook for a model.
// If the model implements BeforeCreateInterface, calls its BeforeCreate method.
//
// Parameters:
//   - ctx: Context for propagating cancellation signals and trace information
//   - model: Model instance (any type)
//
// Returns:
//   - error: Error returned by hook, nil if model doesn't implement interface
//
// Usage scenarios:
//   - Repository.Create() calls before insertion
//   - Repository.BatchCreate() calls for each model before insertion
//   - Repository.Upsert() calls before insertion
//
// Example (internal use):
//
//	func (r *Repository[T]) Create(ctx context.Context, model *T) error {
//	    if err := triggerBeforeCreate(ctx, model); err != nil {
//	        return err // Hook failed, abort creation
//	    }
//	    // ... execute insertion
//	}
func triggerBeforeCreate(ctx context.Context, model any) error {
	// Use type assertion to check if model implements BeforeCreateInterface
	// If implemented, call its BeforeCreate method
	if m, ok := model.(BeforeCreateInterface); ok {
		return m.BeforeCreate(ctx)
	}
	// Interface not implemented, return nil (no-op)
	return nil
}

// triggerAfterCreate triggers the AfterCreate hook for a model.
// If the model implements AfterCreateInterface, calls its AfterCreate method.
//
// Parameters:
//   - ctx: Context for propagating cancellation signals and trace information
//   - model: Model instance (any type)
//
// Returns:
//   - error: Error returned by hook, nil if model doesn't implement interface
//
// Usage scenarios:
//   - Repository.Create() calls after successful insertion
//   - Repository.BatchCreate() calls for each model after successful insertion
//   - Repository.Upsert() calls after successful insertion/update
//
// Note:
//   - Executes within transaction, if error is returned transaction will rollback
//   - Record is already inserted in database at this point, can access auto-increment ID
//
// Example (internal use):
//
//	func (r *Repository[T]) Create(ctx context.Context, model *T) error {
//	    // ... execute insertion
//	    if r.schema.AutoIncrement() {
//	        // Backfill ID
//	    }
//	    return triggerAfterCreate(ctx, model) // Trigger AfterCreate hook
//	}
func triggerAfterCreate(ctx context.Context, model any) error {
	// Use type assertion to check if model implements AfterCreateInterface
	// If implemented, call its AfterCreate method
	if m, ok := model.(AfterCreateInterface); ok {
		return m.AfterCreate(ctx)
	}
	// Interface not implemented, return nil (no-op)
	return nil
}

// triggerBeforeUpdate triggers the BeforeUpdate hook for a model.
// If the model implements BeforeUpdateInterface, calls its BeforeUpdate method.
//
// Parameters:
//   - ctx: Context for propagating cancellation signals and trace information
//   - model: Model instance (any type)
//
// Returns:
//   - error: Error returned by hook, nil if model doesn't implement interface
//
// Usage scenarios:
//   - Repository.Update() calls before updating
//
// Note:
//   - Repository.UpdateColumns() doesn't trigger (no complete model instance)
//   - If error is returned, update operation will be aborted
//
// Example (internal use):
//
//	func (r *Repository[T]) Update(ctx context.Context, model *T) error {
//	    if err := triggerBeforeUpdate(ctx, model); err != nil {
//	        return err // Hook failed, abort update
//	    }
//	    // ... execute update
//	}
func triggerBeforeUpdate(ctx context.Context, model any) error {
	// Use type assertion to check if model implements BeforeUpdateInterface
	// If implemented, call its BeforeUpdate method
	if m, ok := model.(BeforeUpdateInterface); ok {
		return m.BeforeUpdate(ctx)
	}
	// Interface not implemented, return nil (no-op)
	return nil
}

// triggerAfterUpdate triggers the AfterUpdate hook for a model.
// If the model implements AfterUpdateInterface, calls its AfterUpdate method.
//
// Parameters:
//   - ctx: Context for propagating cancellation signals and trace information
//   - model: Model instance (any type)
//
// Returns:
//   - error: Error returned by hook, nil if model doesn't implement interface
//
// Usage scenarios:
//   - Repository.Update() calls after successful update
//
// Note:
//   - Repository.UpdateColumns() doesn't trigger (no complete model instance)
//   - Executes within transaction, if error is returned transaction will rollback
//   - Record is already updated in database at this point
//
// Example (internal use):
//
//	func (r *Repository[T]) Update(ctx context.Context, model *T) error {
//	    // ... execute update
//	    return triggerAfterUpdate(ctx, model) // Trigger AfterUpdate hook
//	}
func triggerAfterUpdate(ctx context.Context, model any) error {
	// Use type assertion to check if model implements AfterUpdateInterface
	// If implemented, call its AfterUpdate method
	if m, ok := model.(AfterUpdateInterface); ok {
		return m.AfterUpdate(ctx)
	}
	// Interface not implemented, return nil (no-op)
	return nil
}

// triggerBeforeDelete triggers the BeforeDelete hook for a model.
// If the model implements BeforeDeleteInterface, calls its BeforeDelete method.
//
// Parameters:
//   - ctx: Context for propagating cancellation signals and trace information
//   - model: Model instance (any type)
//
// Returns:
//   - error: Error returned by hook, nil if model doesn't implement interface
//
// Usage scenarios:
//   - Repository.DeleteModel() calls before deletion
//   - Repository.SoftDeleteModel() calls before soft deletion
//
// Note:
//   - Repository.Delete() doesn't trigger (no model instance)
//   - Repository.SoftDelete() doesn't trigger (no model instance)
//   - If error is returned, deletion operation will be aborted
//   - Record is still in database at this point, can query
//
// Example (internal use):
//
//	func (r *Repository[T]) DeleteModel(ctx context.Context, model *T) error {
//	    if err := triggerBeforeDelete(ctx, model); err != nil {
//	        return err // Hook failed, abort deletion
//	    }
//	    // ... execute deletion
//	}
func triggerBeforeDelete(ctx context.Context, model any) error {
	// Use type assertion to check if model implements BeforeDeleteInterface
	// If implemented, call its BeforeDelete method
	if m, ok := model.(BeforeDeleteInterface); ok {
		return m.BeforeDelete(ctx)
	}
	// Interface not implemented, return nil (no-op)
	return nil
}

// triggerAfterDelete triggers the AfterDelete hook for a model.
// If the model implements AfterDeleteInterface, calls its AfterDelete method.
//
// Parameters:
//   - ctx: Context for propagating cancellation signals and trace information
//   - model: Model instance (any type)
//
// Returns:
//   - error: Error returned by hook, nil if model doesn't implement interface
//
// Usage scenarios:
//   - Repository.DeleteModel() calls after successful deletion
//   - Repository.SoftDeleteModel() calls after successful soft deletion
//
// Note:
//   - Repository.Delete() doesn't trigger (no model instance)
//   - Repository.SoftDelete() doesn't trigger (no model instance)
//   - Executes within transaction, if error is returned transaction will rollback
//   - Record is already deleted from database at this point, cannot query anymore
//
// Example (internal use):
//
//	func (r *Repository[T]) DeleteModel(ctx context.Context, model *T) error {
//	    // ... execute deletion
//	    return triggerAfterDelete(ctx, model) // Trigger AfterDelete hook
//	}
func triggerAfterDelete(ctx context.Context, model any) error {
	// Use type assertion to check if model implements AfterDeleteInterface
	// If implemented, call its AfterDelete method
	if m, ok := model.(AfterDeleteInterface); ok {
		return m.AfterDelete(ctx)
	}
	// Interface not implemented, return nil (no-op)
	return nil
}
