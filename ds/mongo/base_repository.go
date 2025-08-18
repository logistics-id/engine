package mongo

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/logistics-id/engine/common"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CustomQueryFn func(filter bson.M) bson.M

type BaseRepository[T any] struct {
	Collection       *Collection
	Context          context.Context
	searchFields     []string
	enableSoftDelete bool
}

func NewBaseRepository[T any](col *Collection, searchFields []string, enableSoftDelete bool) *BaseRepository[T] {
	return &BaseRepository[T]{
		Collection:       col,
		searchFields:     searchFields,
		enableSoftDelete: enableSoftDelete,
	}
}

func (r *BaseRepository[T]) WithContext(ctx context.Context) common.BaseRepositoryInterface[T] {
	return &BaseRepository[T]{
		Collection:       r.Collection,
		Context:          ctx,
		searchFields:     r.searchFields,
		enableSoftDelete: r.enableSoftDelete,
	}
}

func (r *BaseRepository[T]) Insert(entity *T) error {
	_, err := r.Collection.InsertOne(r.Context, entity)
	return err
}

func (r *BaseRepository[T]) FindByID(id any) (*T, error) {
	idStr, ok := id.(string)
	if !ok {
		return nil, errors.New("id not string.")
	}

	mid, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		return nil, err
	}

	var result T
	filter := bson.M{"_id": mid}
	if r.enableSoftDelete {
		filter["is_deleted"] = false
	}
	err = r.Collection.FindOne(r.Context, filter).Decode(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *BaseRepository[T]) Update(entity *T, fields ...string) error {
	if entity == nil {
		return errors.New("entity is nil")
	}

	id, err := extractEntityID(entity)
	if err != nil {
		return fmt.Errorf("failed to extract ID: %w", err)
	}

	update := bson.M{}
	val := reflect.ValueOf(entity).Elem()
	typ := val.Type()

	for _, field := range fields {
		structField, ok := typ.FieldByName(field)
		if !ok {
			return fmt.Errorf("field %s not found on struct", field)
		}
		update[structField.Tag.Get("bson")] = val.FieldByName(field).Interface()
	}

	_, err = r.Collection.UpdateByID(r.Context, id, bson.M{"$set": update})
	return err
}

func (r *BaseRepository[T]) SoftDelete(id any) error {
	if !r.enableSoftDelete {
		return nil
	}
	_, err := r.Collection.UpdateByID(r.Context, id, bson.M{"$set": bson.M{"is_deleted": true}})
	return err
}

func (r *BaseRepository[T]) FindOne(customQuery CustomQueryFn) (*T, error) {
	var result T
	filter := bson.M{}
	if customQuery != nil {
		filter = customQuery(filter)
	}
	if r.enableSoftDelete {
		filter["is_deleted"] = false
	}
	err := r.Collection.FindOne(r.Context, filter).Decode(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *BaseRepository[T]) FindAll(opts *common.QueryOption, query CustomQueryFn) ([]*T, int64, error) {
	filter := bson.M{}
	if query != nil {
		filter = query(filter)
	}
	if r.enableSoftDelete {
		filter["is_deleted"] = false
	}

	var results []*T
	cursor, err := r.Collection.Find(
		r.Context,
		filter,
		options.Find().
			SetLimit(int64(opts.GetLimit())).
			SetSkip(int64(opts.GetOffset())).
			SetSort(convertSortFields(opts.GetOrders())),
	)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(r.Context)

	for cursor.Next(r.Context) {
		var elem T
		if err := cursor.Decode(&elem); err != nil {
			return nil, 0, err
		}
		results = append(results, &elem)
	}

	count, err := r.Collection.CountDocuments(r.Context, filter)
	if err != nil {
		return nil, 0, err
	}

	return results, count, nil
}

// extractEntityID assumes the struct has a field with `bson:"_id"`
func extractEntityID(entity any) (any, error) {
	val := reflect.ValueOf(entity).Elem()
	typ := val.Type()

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if tag := field.Tag.Get("bson"); tag == "_id" {
			return val.Field(i).Interface(), nil
		}
	}
	return nil, errors.New("no _id field found in entity")
}

func convertSortFields(fields []string) bson.D {
	sort := bson.D{}
	for _, field := range fields {
		order := 1
		if field != "" && field[0] == '-' {
			order = -1
			field = field[1:]
		}
		sort = append(sort, bson.E{Key: field, Value: order})
	}
	return sort
}
