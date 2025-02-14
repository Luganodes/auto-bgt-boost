package repository

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CollectionOperations interface {
	InsertOne(ctx context.Context, document interface{}) error
	FindOne(ctx context.Context, filter bson.M, opts ...*options.FindOneOptions) *mongo.SingleResult
	FindMany(ctx context.Context, filter bson.M, opts *options.FindOptions, documents interface{}) error
	UpdateOne(ctx context.Context, filter bson.M, update interface{}) error
	DeleteOne(ctx context.Context, filter bson.M) error
}

type mongoCollection struct {
	coll *mongo.Collection
}

func (r *mongoRepository) Collection(name string) CollectionOperations {
	return &mongoCollection{
		coll: r.client.Database(r.dbName).Collection(name),
	}
}

func (c *mongoCollection) InsertOne(ctx context.Context, document interface{}) error {
	doc, ok := document.(map[string]interface{})
	if !ok {
		docBytes, err := bson.Marshal(document)
		if err != nil {
			return fmt.Errorf("failed to marshal document: %v", err)
		}
		if err := bson.Unmarshal(docBytes, &doc); err != nil {
			return fmt.Errorf("failed to unmarshal document: %v", err)
		}
	}

	now := time.Now()
	doc["created_at"] = now
	doc["updated_at"] = now
	if _, err := c.coll.InsertOne(ctx, doc); err != nil {
		return fmt.Errorf("failed to insert document: %v", err)
	}
	return nil
}

func (c *mongoCollection) FindOne(ctx context.Context, filter bson.M, opts ...*options.FindOneOptions) *mongo.SingleResult {
	return c.coll.FindOne(ctx, filter, opts...)
}

func (c *mongoCollection) FindMany(ctx context.Context, filter bson.M, opts *options.FindOptions, documents interface{}) error {
	cursor, err := c.coll.Find(ctx, filter, opts)
	if err != nil {
		return fmt.Errorf("failed to find documents: %v", err)
	}
	if err := cursor.All(ctx, documents); err != nil {
		return fmt.Errorf("failed to decode documents: %v", err)
	}
	return nil
}

func (c *mongoCollection) UpdateOne(ctx context.Context, filter bson.M, update interface{}) error {
	finalUpdate := bson.M{
		"$set": update,
		"$currentDate": bson.M{
			"updated_at": true,
		},
	}

	if _, err := c.coll.UpdateOne(ctx, filter, finalUpdate); err != nil {
		return fmt.Errorf("failed to update document: %v", err)
	}
	return nil
}

func (c *mongoCollection) DeleteOne(ctx context.Context, filter bson.M) error {
	if _, err := c.coll.DeleteOne(ctx, filter); err != nil {
		return fmt.Errorf("failed to delete document: %v", err)
	}
	return nil
}
