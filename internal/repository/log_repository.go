package repository

import (
	"context"
	"time"

	"vinzhub-rest-api-v2/internal/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// LogRepository defines the interface for log storage
type LogRepository interface {
	InsertObfuscationLog(ctx context.Context, log *model.ObfuscationLog) error
	GetObfuscationLogs(ctx context.Context, limit, offset int) ([]model.ObfuscationLog, int64, error)
	Close() error
}

// MongoDBLogRepository implements LogRepository for MongoDB
type MongoDBLogRepository struct {
	client     *mongo.Client
	collection *mongo.Collection
}

// NewMongoDBLogRepository creates a new MongoDB log repository
func NewMongoDBLogRepository(uri, dbName, collectionName string) (*MongoDBLogRepository, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	collection := client.Database(dbName).Collection(collectionName)

	return &MongoDBLogRepository{
		client:     client,
		collection: collection,
	}, nil
}

// InsertObfuscationLog inserts a new log entry
func (r *MongoDBLogRepository) InsertObfuscationLog(ctx context.Context, log *model.ObfuscationLog) error {
	log.CreatedAt = time.Now()
	_, err := r.collection.InsertOne(ctx, log)
	return err
}

// GetObfuscationLogs retrieval logs with pagination
func (r *MongoDBLogRepository) GetObfuscationLogs(ctx context.Context, limit, offset int) ([]model.ObfuscationLog, int64, error) {
	findOptions := options.Find()
	findOptions.SetSort(bson.D{{Key: "created_at", Value: -1}})
	findOptions.SetLimit(int64(limit))
	findOptions.SetSkip(int64(offset))

	cursor, err := r.collection.Find(ctx, bson.M{}, findOptions)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var logs []model.ObfuscationLog
	if err := cursor.All(ctx, &logs); err != nil {
		return nil, 0, err
	}
	
	// Ensure not nil slice for JSON
	if logs == nil {
		logs = []model.ObfuscationLog{}
	}

	count, err := r.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, 0, err
	}

	return logs, count, nil
}

// Close closes the MongoDB connection
func (r *MongoDBLogRepository) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return r.client.Disconnect(ctx)
}
