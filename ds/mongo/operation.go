package mongo

import (
	"reflect"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/net/context"
)

// Model is a placeholder for any struct with an `ID` field.
type Model any

// setID uses reflection to assign an ID to a model's `ID` field.
func setID(m Model, id any) {
	reflect.ValueOf(m).Elem().FieldByName("_id").Set(reflect.ValueOf(id))
}

// getID retrieves the ID from the model using reflection.
func getID(m Model) any {
	return reflect.ValueOf(m).Elem().FieldByName("_id").Interface()
}

// Create inserts a model into the collection and sets the inserted ID back into the model.
func create(ctx context.Context, c *Collection, model Model, opts ...*options.InsertOneOptions) error {
	res, err := c.InsertOne(ctx, model, opts...)
	if err == nil {
		setID(model, res.InsertedID)
	}
	return err
}

// First finds one document by filter and decodes it into the provided model.
func first(ctx context.Context, c *Collection, filter any, model Model, opts ...*options.FindOneOptions) error {
	return c.FindOne(ctx, filter, opts...).Decode(model)
}

// Update performs an update using the model's ID and replaces all fields.
func update(ctx context.Context, c *Collection, model Model, opts ...*options.UpdateOptions) error {
	_, err := c.UpdateOne(ctx, bson.M{"_id": getID(model)}, bson.M{"$set": model}, opts...)
	return err
}

// Delete removes a document using the model's ID.
func del(ctx context.Context, c *Collection, model Model) error {
	_, err := c.DeleteOne(ctx, bson.M{"_id": getID(model)})
	return err
}
