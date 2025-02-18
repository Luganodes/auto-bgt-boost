package repository

import (
	"bgt_boost/internal/config"
	"bgt_boost/internal/models"
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DbRepository interface {
	Health() error
	Disconnect() error
	AddQueueBoost(ctx context.Context, boost models.QueueBoost) error
	AddActivateBoost(ctx context.Context, boost models.ActivateBoost) error
	GetInActiveBoosts(ctx context.Context) ([]models.QueueBoost, error)
	DoesQueueBoostExist(ctx context.Context, pubkey string) (bool, error)
	MarkBoostAsActivated(ctx context.Context, transactionHash string) error
	GetValidators(ctx context.Context) ([]models.Validator, error)
	GetValidator(ctx context.Context, pubkey string) (models.Validator, error)
	DoesValidatorExist(ctx context.Context, pubkey string) (bool, error)
	AddValidator(ctx context.Context, validator models.Validator) error
	UpdateValidator(ctx context.Context, pubkey string, validator models.Validator) error
	DeleteValidator(ctx context.Context, pubkey string) error
}

type mongoRepository struct {
	client *mongo.Client
	dbName string
}

func ConnectToDb(config *config.Config) (DbRepository, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	host := config.Db.Host
	port := config.Db.Port
	user := config.Db.User
	password := config.Db.Password
	dbName := config.Db.DbName

	uri := fmt.Sprintf("mongodb://%s:%d", host, port)
	if user != "" && password != "" {
		uri = fmt.Sprintf("mongodb://%s:%s@%s:%d", user, password, host, port)
	}

	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %v", err)
	}

	repo := &mongoRepository{
		client: client,
		dbName: dbName,
	}

	if err := repo.ensureIndexes(); err != nil {
		return nil, fmt.Errorf("failed to ensure indexes: %v", err)
	}

	log.Println("✅ Connected to Database")
	return repo, nil
}

func (r *mongoRepository) ensureIndexes() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Ensure indexes for the validators collection
	validatorsCollection := r.client.Database(r.dbName).Collection("validators")
	validatorsIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "pubkey", Value: 1}},
			Options: options.Index().SetName("pubkey_index").SetUnique(true),
		},
	}
	if err := r.createIndexesIfNotExist(ctx, validatorsCollection, validatorsIndexes); err != nil {
		return fmt.Errorf("failed to ensure indexes for validators collection: %v", err)
	}
	log.Println("✅ Indexes ensured successfully")
	return nil
}

func (r *mongoRepository) createIndexesIfNotExist(ctx context.Context, collection *mongo.Collection, indexes []mongo.IndexModel) error {
	cursor, err := collection.Indexes().List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list existing indexes: %v", err)
	}
	defer cursor.Close(ctx)

	var existingIndexes []bson.M
	if err = cursor.All(ctx, &existingIndexes); err != nil {
		return fmt.Errorf("failed to read existing indexes: %v", err)
	}

	for _, index := range indexes {
		var indexName string
		if index.Options != nil && index.Options.Name != nil {
			indexName = *index.Options.Name
		} else {
			// Generate a name based on the keys if no name is provided
			var keyNames []string
			for _, key := range index.Keys.(bson.D) {
				keyNames = append(keyNames, key.Key)
			}
			indexName = fmt.Sprintf("%s_index", strings.Join(keyNames, "_"))
		}

		if !indexExists(existingIndexes, indexName) {
			_, err := collection.Indexes().CreateOne(ctx, index)
			if err != nil {
				return fmt.Errorf("failed to create index %s: %v", indexName, err)
			}
			log.Printf("Created index: %s", indexName)
		} else {
			log.Printf("Index already exists: %s", indexName)
		}
	}

	return nil
}

func indexExists(existingIndexes []bson.M, indexName string) bool {
	for _, index := range existingIndexes {
		if index["name"] == indexName {
			return true
		}
	}
	return false
}

func (r *mongoRepository) Health() error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	return r.client.Ping(ctx, nil)
}

func (r *mongoRepository) Disconnect() error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	return r.client.Disconnect(ctx)
}

func (r *mongoRepository) AddQueueBoost(ctx context.Context, boost models.QueueBoost) error {
	return r.Collection("queue_boosts").InsertOne(ctx, boost)
}

func (r *mongoRepository) AddActivateBoost(ctx context.Context, boost models.ActivateBoost) error {
	return r.Collection("activate_boosts").InsertOne(ctx, boost)
}

func (r *mongoRepository) GetInActiveBoosts(ctx context.Context) ([]models.QueueBoost, error) {
	var queueBoosts []models.QueueBoost
	if err := r.Collection("queue_boosts").FindMany(ctx, bson.M{"activated": false}, nil, &queueBoosts); err != nil {
		return nil, err
	}
	return queueBoosts, nil
}

func (r *mongoRepository) DoesQueueBoostExist(ctx context.Context, pubkey string) (bool, error) {
	var queueBoost models.QueueBoost
	if err := r.Collection("queue_boosts").FindOne(ctx, bson.M{"pubkey": pubkey, "activated": false}, nil).Decode(&queueBoost); err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *mongoRepository) MarkBoostAsActivated(ctx context.Context, transactionHash string) error {
	return r.Collection("queue_boosts").UpdateOne(ctx, bson.M{"transactionHash": transactionHash}, bson.M{"activated": true})
}

func (r *mongoRepository) GetValidators(ctx context.Context) ([]models.Validator, error) {
	var validators []models.Validator
	if err := r.Collection("validators").FindMany(ctx, bson.M{}, nil, &validators); err != nil {
		return nil, err
	}
	return validators, nil
}

func (r *mongoRepository) GetValidator(ctx context.Context, pubkey string) (models.Validator, error) {
	var validator models.Validator
	if err := r.Collection("validators").FindOne(ctx, bson.M{"pubkey": pubkey}, nil).Decode(&validator); err != nil {
		return models.Validator{}, err
	}
	return validator, nil
}

func (r *mongoRepository) DoesValidatorExist(ctx context.Context, pubkey string) (bool, error) {
	var validator models.Validator
	if err := r.Collection("validators").FindOne(ctx, bson.M{"pubkey": pubkey}, nil).Decode(&validator); err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *mongoRepository) AddValidator(ctx context.Context, validator models.Validator) error {
	return r.Collection("validators").InsertOne(ctx, validator)
}

func (r *mongoRepository) UpdateValidator(ctx context.Context, pubkey string, validator models.Validator) error {
	fmt.Println("Updating validator: ", pubkey)
	fmt.Printf("Validator: %+v\n", validator)
	return r.Collection("validators").UpdateOne(ctx, bson.M{"pubkey": pubkey}, validator)
}

func (r *mongoRepository) DeleteValidator(ctx context.Context, pubkey string) error {
	return r.Collection("validators").DeleteOne(ctx, bson.M{"pubkey": pubkey})
}
