package sqlc

import (
	"context"
	"slices"

	sq "github.com/Masterminds/squirrel"
	"github.com/arllen133/sqlc/clause"
)

// Repository manages CRUD operations for a model T
type Repository[T any] struct {
	session *Session
	schema  Schema[T]
	scopes  []clause.Expression
}

func NewRepository[T any](session *Session) *Repository[T] {
	return &Repository[T]{
		session: session,
		schema:  LoadSchema[T](),
		scopes:  make([]clause.Expression, 0),
	}
}

// Where returns a new Repository instance with the given conditions appended.
// This allows for method chaining, e.g. repo.Where(cond).Update(...)
func (r *Repository[T]) Where(conds ...clause.Expression) *Repository[T] {
	newRepo := *r
	newRepo.scopes = append(newRepo.scopes, conds...)
	return &newRepo
}

func (r *Repository[T]) Create(ctx context.Context, model *T) error {
	if err := triggerBeforeCreate(ctx, model); err != nil {
		return err
	}

	cols, vals := r.schema.InsertRow(model)

	builder := sq.Insert(r.schema.TableName()).
		Columns(cols...).
		Values(vals...).
		PlaceholderFormat(r.session.dialect.PlaceholderFormat())

	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	result, err := r.session.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	if r.schema.AutoIncrement() {
		id, err := result.LastInsertId()
		if err == nil {
			r.schema.SetPK(model, id)
		}
	}

	return triggerAfterCreate(ctx, model)
}

// BatchCreate inserts multiple records in a single statement
func (r *Repository[T]) BatchCreate(ctx context.Context, models []*T) error {
	if len(models) == 0 {
		return nil
	}

	for _, model := range models {
		if err := triggerBeforeCreate(ctx, model); err != nil {
			return err
		}
	}

	builder := sq.Insert(r.schema.TableName()).
		PlaceholderFormat(r.session.dialect.PlaceholderFormat())

	for i, model := range models {
		cols, vals := r.schema.InsertRow(model)
		if i == 0 {
			builder = builder.Columns(cols...)
		}
		builder = builder.Values(vals...)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	_, err = r.session.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	// Note: We cannot easily populate auto-increment IDs for batch insert across all databases.
	// For MVP, we skip updating model IDs.

	for _, model := range models {
		if err := triggerAfterCreate(ctx, model); err != nil {
			return err
		}
	}
	return nil
}

// Upsert Options
type upsertConfig struct {
	conflictCols []string
	updateCols   []string
}

type UpsertOption func(*upsertConfig)

// OnConflict specifies the columns to check for conflict (e.g., PRIMARY KEY or UNIQUE constraints)
func OnConflict(columns ...clause.Columnar) UpsertOption {
	return func(c *upsertConfig) {
		c.conflictCols = ResolveColumnNames(columns)
	}
}

// DoUpdate specifies which columns to update when a conflict occurs.
// If not specified, all model columns (except conflict columns) are updated.
func DoUpdate(columns ...clause.Columnar) UpsertOption {
	return func(c *upsertConfig) {
		c.updateCols = ResolveColumnNames(columns)
	}
}

// Upsert inserts or updates a record.
// By default, it uses the Primary Key as the conflict target and updates all other columns.
// You can customize this utilizing OnConflict() and DoUpdate() options.
func (r *Repository[T]) Upsert(ctx context.Context, model *T, opts ...UpsertOption) error {
	config := &upsertConfig{}
	for _, opt := range opts {
		opt(config)
	}

	if err := triggerBeforeCreate(ctx, model); err != nil {
		return err
	}

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

	upsertClause := r.session.dialect.UpsertClause(r.schema.TableName(), conflictCols, updateCols)

	builder := sq.Insert(r.schema.TableName()).
		Columns(cols...).
		Values(vals...).
		Suffix(upsertClause).
		PlaceholderFormat(r.session.dialect.PlaceholderFormat())

	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	_, err = r.session.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	return triggerAfterCreate(ctx, model)
}

func (r *Repository[T]) Update(ctx context.Context, model *T) error {
	if err := triggerBeforeUpdate(ctx, model); err != nil {
		return err
	}

	setMap := r.schema.UpdateMap(model)
	pk := r.schema.PK(model)

	builder := sq.Update(r.schema.TableName()).
		SetMap(setMap).
		Where(sq.Eq{pk.Column.Name: pk.Value})

	// Apply Scopes
	for _, scope := range r.scopes {
		builder = builder.Where(scope)
	}

	builder = builder.PlaceholderFormat(r.session.dialect.PlaceholderFormat())

	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	_, err = r.session.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	return triggerAfterUpdate(ctx, model)
}

// UpdateColumns updates specific columns for a record identified by id.
// This allows for partial updates without loading the entire record.
func (r *Repository[T]) UpdateColumns(ctx context.Context, id any, assignments ...clause.Assignment) error {
	if len(assignments) == 0 {
		return nil
	}

	pkMeta := r.schema.PK(nil)

	builder := sq.Update(r.schema.TableName()).
		Where(sq.Eq{pkMeta.Column.Name: id})

	// Apply Scopes
	for _, scope := range r.scopes {
		builder = builder.Where(scope)
	}

	builder = builder.PlaceholderFormat(r.session.dialect.PlaceholderFormat())

	for _, assignment := range assignments {
		builder = builder.Set(assignment.Column.ColumnName(), assignment.Value)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	_, err = r.session.Exec(ctx, query, args...)
	return err
}

func (r *Repository[T]) Delete(ctx context.Context, id any) error {
	// For Delete, we don't always have the model instance to trigger hooks easily
	// if we only have the ID. For MVP, we'll assume standard Delete doesn't
	// trigger hooks unless we pass the model.
	// But let's add a DeleteModel method later if needed.
	pkMeta := r.schema.PK(nil)

	builder := sq.Delete(r.schema.TableName()).
		Where(sq.Eq{pkMeta.Column.Name: id})

	// Apply Scopes
	for _, scope := range r.scopes {
		builder = builder.Where(scope)
	}

	builder = builder.PlaceholderFormat(r.session.dialect.PlaceholderFormat())

	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	_, err = r.session.Exec(ctx, query, args...)
	return err
}

// DeleteModel deletes a record by model instance, triggering lifecycle hooks.
func (r *Repository[T]) DeleteModel(ctx context.Context, model *T) error {
	if err := triggerBeforeDelete(ctx, model); err != nil {
		return err
	}

	pk := r.schema.PK(model)

	builder := sq.Delete(r.schema.TableName()).
		Where(sq.Eq{pk.Column.Name: pk.Value})

	// Apply Scopes
	for _, scope := range r.scopes {
		builder = builder.Where(scope)
	}

	builder = builder.PlaceholderFormat(r.session.dialect.PlaceholderFormat())

	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	_, err = r.session.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	return triggerAfterDelete(ctx, model)
}

func (r *Repository[T]) Query() *QueryBuilder[T] {
	return Query[T](r.session)
}

func (r *Repository[T]) FindOne(ctx context.Context, id any) (*T, error) {
	pkMeta := r.schema.PK(nil)
	query := r.Query().Where(clause.Eq{Column: pkMeta.Column, Value: id})

	// Apply Scopes to Query
	for _, scope := range r.scopes {
		query = query.Where(scope)
	}
	return query.First(ctx)
}
