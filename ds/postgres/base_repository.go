// Package postgres provides PostgreSQL database connectivity and repository patterns
// using Bun ORM with integrated logging and generic base repository functionality.
package postgres

import (
	"context"
	"fmt"

	"github.com/logistics-id/engine/common"
	"github.com/uptrace/bun"
)

// CustomQueryFn is a function type for custom query modifications specific to Bun/PostgreSQL
type CustomQueryFn func(q *bun.SelectQuery) *bun.SelectQuery

type BaseRepository[T any] struct {
	DB               bun.IDB
	Context          context.Context
	table            string
	searchFields     []string
	defaultRelations []string
	enableSoftDelete bool
}

func NewBaseRepository[T any](db *bun.DB, table string, searchFields, defaultRelations []string, enableSoftDelete bool) *BaseRepository[T] {
	return &BaseRepository[T]{
		DB:               db,
		table:            table,
		searchFields:     searchFields,
		defaultRelations: defaultRelations,
		enableSoftDelete: enableSoftDelete,
	}
}

func (r *BaseRepository[T]) WithContext(ctx context.Context) common.BaseRepositoryInterface[T] {
	return r.WithCtx(ctx)
}

// WithCtx returns a new BaseRepository instance with the given context.
// This method returns the concrete type, allowing access to postgres-specific methods.
// Use this in custom repositories to enable method chaining with postgres-specific methods.
func (r *BaseRepository[T]) WithCtx(ctx context.Context) *BaseRepository[T] {
	return &BaseRepository[T]{
		DB:               r.DB,
		Context:          ctx,
		table:            r.table,
		searchFields:     r.searchFields,
		defaultRelations: r.defaultRelations,
		enableSoftDelete: r.enableSoftDelete,
	}
}

// WithTx returns a new BaseRepository instance with the given transaction context.
// This method returns the concrete type, allowing access to postgres-specific methods.
// Use this when you need to execute multiple operations within a transaction.
func (r *BaseRepository[T]) WithTx(ctx context.Context, tx bun.Tx) *BaseRepository[T] {
	return &BaseRepository[T]{
		DB:               tx,
		Context:          ctx,
		table:            r.table,
		searchFields:     r.searchFields,
		defaultRelations: r.defaultRelations,
		enableSoftDelete: r.enableSoftDelete,
	}
}

func (r *BaseRepository[T]) Insert(entity *T) error {
	_, err := r.DB.NewInsert().Model(entity).Exec(r.Context)
	return err
}

func (r *BaseRepository[T]) FindByID(id any) (*T, error) {
	entity := new(T)
	q := r.DB.NewSelect().
		Model(entity)

	q.Where(fmt.Sprintf("%s.id = ?", r.table), id)

	if r.enableSoftDelete {
		q.Where(fmt.Sprintf("%s.is_deleted = false", r.table))
	}

	for _, rel := range r.defaultRelations {
		q.Relation(rel)
	}

	err := q.Scan(r.Context)
	if err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *BaseRepository[T]) Update(entity *T, fields ...string) error {
	query := r.DB.NewUpdate().Model(entity).WherePK()
	if len(fields) > 0 {
		query.Column(fields...)
	}
	_, err := query.Exec(r.Context)
	return err
}

func (r *BaseRepository[T]) SoftDelete(id any) error {
	if !r.enableSoftDelete {
		return nil
	}
	_, err := r.DB.NewUpdate().
		Model((*T)(nil)).
		Set("is_deleted = true").
		Where("id = ?", id).
		Exec(r.Context)
	return err
}

func (r *BaseRepository[T]) FindAll(opts *common.QueryOption, customQuery CustomQueryFn) ([]*T, int64, error) {
	var result []*T

	q := r.DB.NewSelect().Model(&result)

	if opts.Search != "" && len(r.searchFields) > 0 {
		FilterSearch(q, opts.Search, r.searchFields...)
	}

	for _, cond := range opts.Conditions {
		if strCond, ok := cond.(string); ok {
			q.Where(strCond)
		}
	}

	if r.enableSoftDelete {
		q.Where(fmt.Sprintf("%s.is_deleted = false", r.table))
	}

	for _, rel := range r.defaultRelations {
		q.Relation(rel)
	}

	if customQuery != nil {
		q = customQuery(q)
	}

	total, err := q.Count(r.Context)
	if err != nil || total == 0 {
		return nil, 0, err
	}

	q.OrderExpr(RequestSort(opts.GetOrders()))
	q.Limit(int(opts.GetLimit()))
	q.Offset(int(opts.GetOffset()))

	if err := q.Scan(r.Context); err != nil {
		return nil, 0, err
	}

	return result, int64(total), nil
}

func (r *BaseRepository[T]) FindOne(customQuery CustomQueryFn) (*T, error) {
	var result T

	q := r.DB.NewSelect().Model(&result)

	if customQuery != nil {
		q = customQuery(q)
	}

	for _, rel := range r.defaultRelations {
		q.Relation(rel)
	}

	err := q.Limit(1).Scan(r.Context, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// RunInTx executes a function within a database transaction.
// This method provides full control - you receive the context and transaction,
// and can create multiple repository instances with WithTx as needed.
// Use this when you need to work with multiple different repositories in the same transaction.
func (r *BaseRepository[T]) RunInTx(ctx context.Context, fn func(context.Context, bun.Tx) error) error {
	return r.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return fn(ctx, tx)
	})
}

// RunInTxWithRepo executes a function within a database transaction,
// automatically passing a repository instance with the transaction context.
// This is a convenience method for simpler cases where you only need one repository.
// For multiple repositories or more control, use RunInTx instead.
func (r *BaseRepository[T]) RunInTxWithRepo(ctx context.Context, fn func(*BaseRepository[T]) error) error {
	return r.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		repoWithTx := r.WithTx(ctx, tx)
		return fn(repoWithTx)
	})
}
