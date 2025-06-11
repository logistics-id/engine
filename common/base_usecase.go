package common

import (
	"context"
)

type BaseUsecase[T any] struct {
	Repo    BaseRepositoryInterface[T]
	Context context.Context
}

func NewBaseUsecase[T any](repo BaseRepositoryInterface[T]) *BaseUsecase[T] {
	return &BaseUsecase[T]{Repo: repo}
}

func (u *BaseUsecase[T]) WithContext(ctx context.Context) *BaseUsecase[T] {
	return &BaseUsecase[T]{
		Repo:    u.Repo.WithContext(ctx),
		Context: ctx,
	}
}

func (u *BaseUsecase[T]) Create(entity *T) error {
	return u.Repo.Insert(entity)
}

func (u *BaseUsecase[T]) GetByID(id any) (*T, error) {
	return u.Repo.FindByID(id)
}

func (u *BaseUsecase[T]) Update(entity *T, fields ...string) error {
	return u.Repo.Update(entity, fields...)
}

func (u *BaseUsecase[T]) Delete(id any) error {
	return u.Repo.SoftDelete(id)
}
