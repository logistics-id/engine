package postgres

import (
	"context"
	"fmt"

	"github.com/logistics-id/engine/common"
	"github.com/uptrace/bun"
)

type CustomQueryFn func(q *bun.SelectQuery) *bun.SelectQuery

type BaseRepository[T any] struct {
	DB               *bun.DB
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
	return &BaseRepository[T]{
		DB:               r.DB,
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

	if err := q.Scan(r.Context, &result); err != nil {
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

	err := q.Limit(1).Scan(r.Context, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
