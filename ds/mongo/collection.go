package mongo

import (
	"reflect"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/net/context"
)

// ID is the default MongoDB document ID field name.
const ID = "_id"

// Collection wraps a MongoDB collection with a default context for convenience.
type Collection struct {
	*mongo.Collection
	context context.Context
}

// Model is an alias for any struct representing a MongoDB document.
type Model any

// Count returns the number of documents matching the given filter.
// Returns the count and any error encountered.
func (c *Collection) Count(filter any) (int64, error) {
	return c.CountDocuments(c.context, filter)
}

// Show finds a document by its ID (string or ObjectID) and decodes it into 'model'.
// Optional FindOneOptions can be passed.
// Returns an error if the document is not found or the ID is invalid.
func (c *Collection) Show(id any, model Model, opts ...*options.FindOneOptions) error {
	// Support string ObjectID (hex) or primitive.ObjectID
	if idStr, ok := id.(string); ok {
		objID, err := primitive.ObjectIDFromHex(idStr)
		if err != nil {
			return err
		}
		id = objID
	}
	return c.FindOne(c.context, bson.M{ID: id}, opts...).Decode(model)
}

// Create inserts the given model into the collection.
// Optionally accepts InsertOneOptions. On success, sets the inserted ID back to the model.
// Returns error if insertion fails.
func (c *Collection) Create(model Model, opts ...*options.InsertOneOptions) error {
	res, err := c.InsertOne(c.context, model, opts...)
	if err == nil {
		setID(model, res.InsertedID)
	}
	return err
}

// Delete removes a document matching the model's ID from the collection.
// Returns error if deletion fails.
func (c *Collection) Delete(model Model) error {
	_, err := c.DeleteOne(c.context, bson.M{ID: getID(model)})
	return err
}

// Update updates only the specified fields of the given model document by ID.
// Returns error if the update fails.
func (c *Collection) Update(model Model, fields ...string) error {
	_, err := c.Collection.UpdateOne(
		c.context,
		bson.M{ID: getID(model)},
		bson.M{"$set": StructFilter(model, fields...)},
	)
	return err
}

// Finds executes a find query with the given filter and options,
// and decodes all results into 'results' (must be a pointer to a slice).
// Returns error if the find or decoding fails.
func (c *Collection) Finds(results any, filter any, opts ...*options.FindOptions) error {
	cur, err := c.Find(c.context, filter, opts...)
	if err != nil {
		return err
	}
	return cur.All(c.context, results)
}

// GetOne finds a single document matching the given filter and decodes it into the model.
// Returns error if not found.
func (c *Collection) GetOne(filter any, model Model, opts ...*options.FindOneOptions) error {
	return c.FindOne(c.context, filter, opts...).Decode(model)
}

// WithContext sets a new context for the Collection and returns itself for chaining.
func (c *Collection) WithContext(ctx context.Context) *Collection {
	c.context = ctx
	return c
}

// getID retrieves the "_id" field value from the model using reflection.
// Assumes 'model' is a pointer to a struct with an exported "_id" or "ID" field.
func getID(m Model) any {
	v := reflect.ValueOf(m)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return nil
	}
	s := v.Elem()

	// Try "_id" (MongoDB convention)
	idField := s.FieldByName("_id")
	if idField.IsValid() {
		return idField.Interface()
	}
	// Fallback to "ID"
	idField = s.FieldByName("ID")
	if idField.IsValid() {
		return idField.Interface()
	}
	return nil
}

// setID assigns the given id value to the model's "_id" or "ID" field using reflection.
// Assumes 'model' is a pointer to a struct with a settable "_id" or "ID" field.
func setID(m Model, id any) {
	v := reflect.ValueOf(m)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return
	}
	s := v.Elem()
	// Try "_id" (MongoDB convention)
	idField := s.FieldByName("_id")
	if idField.IsValid() && idField.CanSet() {
		idVal := reflect.ValueOf(id)
		if idVal.Type().AssignableTo(idField.Type()) {
			idField.Set(idVal)
		}
		return
	}
	// Fallback to "ID"
	idField = s.FieldByName("ID")
	if idField.IsValid() && idField.CanSet() {
		idVal := reflect.ValueOf(id)
		if idVal.Type().AssignableTo(idField.Type()) {
			idField.Set(idVal)
		}
	}
}

// NewCollection creates and returns a new Collection from the default DB with the given name and options.
// The returned Collection uses the global default context, but you can override it with WithContext.
func NewCollection(name string, opts ...*options.CollectionOptions) *Collection {
	coll := defaultDB.Collection(name, opts...)
	return &Collection{Collection: coll}
}
