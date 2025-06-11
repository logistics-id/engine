package common

import "context"

// BaseRepositoryInterface defines common repository behaviors for all datastores.
type BaseRepositoryInterface[T any] interface {
	WithContext(ctx context.Context) BaseRepositoryInterface[T]
	Insert(entity *T) error
	FindByID(id any) (*T, error)
	Update(entity *T, fields ...string) error
	SoftDelete(id any) error
}
