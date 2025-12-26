package repository

import (
	"context"
	"fmt"
	"log"
	"time"

	"vinzhub-rest-api-v2/internal/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDBInventoryRepository implements InventoryRepository using MongoDB.
type MongoDBInventoryRepository struct {
	client     *mongo.Client
	db         *mongo.Database
	collection *mongo.Collection
}

// NewMongoDBInventoryRepository creates a new MongoDB inventory repository.
func NewMongoDBInventoryRepository(uri, database, collection string) (*MongoDBInventoryRepository, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Connect with retry
	clientOpts := options.Client().
		ApplyURI(uri).
		SetMaxPoolSize(50).
		SetMinPoolSize(5).
		SetMaxConnIdleTime(5 * time.Minute).
		SetRetryWrites(true)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db := client.Database(database)
	coll := db.Collection(collection)

	// Create index on roblox_user_id
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "roblox_user_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	_, err = coll.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		log.Printf("[MongoDB] Warning: failed to create index: %v", err)
	}

	log.Printf("[MongoDB] Connected to %s/%s", database, collection)
	return &MongoDBInventoryRepository{
		client:     client,
		db:         db,
		collection: coll,
	}, nil
}

// InventoryDocument represents a document in MongoDB.
type InventoryDocument struct {
	RobloxUserID string    `bson:"roblox_user_id"`
	KeyAccountID int64     `bson:"key_account_id,omitempty"`
	InventoryJSON bson.Raw `bson:"inventory_json"`
	SyncedAt     time.Time `bson:"synced_at"`
}

// UpsertRawInventory inserts or updates raw JSON inventory.
func (r *MongoDBInventoryRepository) UpsertRawInventory(ctx context.Context, keyAccountID int64, robloxUserID string, rawJSON []byte) error {
	filter := bson.M{"roblox_user_id": robloxUserID}
	update := bson.M{
		"$set": bson.M{
			"key_account_id":  keyAccountID,
			"inventory_json":  bson.Raw(rawJSON),
			"synced_at":       time.Now(),
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to upsert inventory: %w", err)
	}
	return nil
}

// BatchUpsertRawInventory upserts multiple inventory records.
func (r *MongoDBInventoryRepository) BatchUpsertRawInventory(ctx context.Context, items []model.InventoryItem) error {
	if len(items) == 0 {
		return nil
	}

	models := make([]mongo.WriteModel, len(items))
	for i, item := range items {
		filter := bson.M{"roblox_user_id": item.RobloxUserID}
		update := bson.M{
			"$set": bson.M{
				"key_account_id":  item.KeyAccountID,
				"inventory_json":  bson.Raw(item.RawJSON),
				"synced_at":       item.SyncedAt,
			},
		}
		models[i] = mongo.NewUpdateOneModel().SetFilter(filter).SetUpdate(update).SetUpsert(true)
	}

	opts := options.BulkWrite().SetOrdered(false)
	_, err := r.collection.BulkWrite(ctx, models, opts)
	if err != nil {
		return fmt.Errorf("failed to batch upsert: %w", err)
	}

	log.Printf("[MongoDB] Batch upserted %d items", len(items))
	return nil
}

// GetRawInventory retrieves raw JSON inventory by Roblox user ID.
func (r *MongoDBInventoryRepository) GetRawInventory(ctx context.Context, robloxUserID string) ([]byte, *time.Time, error) {
	filter := bson.M{"roblox_user_id": robloxUserID}

	var doc InventoryDocument
	err := r.collection.FindOne(ctx, filter).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get inventory: %w", err)
	}

	return []byte(doc.InventoryJSON), &doc.SyncedAt, nil
}

// GetStats returns statistics about the inventory collection.
func (r *MongoDBInventoryRepository) GetStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	stats["status"] = "connected"

	// Count documents
	count, err := r.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return stats, err
	}
	stats["total_inventories"] = count

	// Get last sync time
	opts := options.FindOne().SetSort(bson.D{{Key: "synced_at", Value: -1}})
	var doc InventoryDocument
	err = r.collection.FindOne(ctx, bson.M{}, opts).Decode(&doc)
	if err == nil {
		stats["last_sync"] = doc.SyncedAt
	}

	// Get collection stats
	result := r.db.RunCommand(ctx, bson.D{{Key: "collStats", Value: r.collection.Name()}})
	var collStats bson.M
	if err := result.Decode(&collStats); err == nil {
		if size, ok := collStats["size"].(int64); ok {
			stats["db_size_bytes"] = size
		} else if size, ok := collStats["size"].(int32); ok {
			stats["db_size_bytes"] = int64(size)
		}
	}

	return stats, nil
}

// Close closes the MongoDB connection.
func (r *MongoDBInventoryRepository) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return r.client.Disconnect(ctx)
}
