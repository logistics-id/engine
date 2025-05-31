package mongo

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/net/context"
)

const ID = "_id"

type Collection struct {
	*mongo.Collection
}

func (coll *Collection) Counts(ctx context.Context, filter any) (int64, error) {
	return coll.CountDocuments(ctx, filter)
}

// FindByID method finds a doc and decodes it to a model, otherwise returns an error.
// The id field can be any value that if passed to the `PrepareID` method, it returns
// a valid ID (e.g string, bson.ObjectId).
func (coll *Collection) Show(ctx context.Context, id any, model Model) error {
	return coll.FindByIDWithCtx(ctx, id, model)
}

// Create method inserts a new model into the database.
func (coll *Collection) Create(ctx context.Context, model Model, opts ...*options.InsertOneOptions) error {
	return coll.CreateWithCtx(ctx, model, opts...)
}

// Delete method deletes a model (doc) from a collection.
// To perform additional operations when deleting a model
// you should use hooks rather than overriding this method.
func (coll *Collection) Delete(ctx context.Context, model Model) error {
	return del(ctx, coll, model)
}

func (coll *Collection) Update(ctx context.Context, model Model, fields ...string) error {
	_, e := coll.Collection.UpdateOne(ctx, bson.M{ID: getID(model)}, bson.M{"$set": StructFilter(model, fields...)})

	return e
}

// Finds, decodes and returns the results.
func (coll *Collection) Finds(ctx context.Context, results any, filter any, opts ...*options.FindOptions) error {
	return coll.SimpleFindWithCtx(ctx, results, filter, opts...)
}

// FindByIDWithCtx method finds a doc and decodes it to a model, otherwise returns an error.
// The id field can be any value that if passed to the `PrepareID` method, it returns
// a valid ID (e.g string, bson.ObjectId).
func (coll *Collection) FindByIDWithCtx(ctx context.Context, id any, model Model) (e error) {

	if idStr, ok := id.(string); ok {
		if id, e = primitive.ObjectIDFromHex(idStr); e != nil {
			return e
		}
	}

	return first(ctx, coll, bson.M{ID: id}, model)
}

// CreateWithCtx method inserts a new model into the database.
func (coll *Collection) CreateWithCtx(ctx context.Context, model Model, opts ...*options.InsertOneOptions) error {
	return create(ctx, coll, model, opts...)
}

// UpdateWithCtx function persists the changes made to a model to the database using the specified context.
// Calling this method also invokes the model's mgm updating, updated,
// saving, and saved hooks.
func (coll *Collection) UpdateWithCtx(ctx context.Context, model Model, opts ...*options.UpdateOptions) error {
	return update(ctx, coll, model, opts...)
}

// DeleteWithCtx method deletes a model (doc) from a collection using the specified context.
// To perform additional operations when deleting a model
// you should use hooks rather than overriding this method.
func (coll *Collection) DeleteWithCtx(ctx context.Context, model Model) error {
	return del(ctx, coll, model)
}

// SimpleFindWithCtx finds, decodes and returns the results using the specified context.
func (coll *Collection) SimpleFindWithCtx(ctx context.Context, results any, filter any, opts ...*options.FindOptions) error {
	cur, err := coll.Find(ctx, filter, opts...)

	if err != nil {
		return err
	}

	return cur.All(ctx, results)
}

func (coll *Collection) GetOne(ctx context.Context, filter any, model Model) (e error) {

	return first(ctx, coll, filter, model)
}

// NewCollection returns a wrapped collection.
func NewCollection(name string, opts ...*options.CollectionOptions) *Collection {
	coll := defaultDB.Collection(name, opts...)
	return &Collection{Collection: coll}
}
