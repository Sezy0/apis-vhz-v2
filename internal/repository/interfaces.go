package repository

import (
	"context"
	"time"

	"vinzhub-rest-api-v2/internal/model"
)

// InventoryRepository defines inventory data access methods.
type InventoryRepository interface {
	// UpsertRawInventory inserts or updates raw JSON inventory.
	UpsertRawInventory(ctx context.Context, keyAccountID int64, robloxUserID string, rawJSON []byte) error

	// GetRawInventory retrieves raw JSON inventory by Roblox user ID.
	GetRawInventory(ctx context.Context, robloxUserID string) ([]byte, *time.Time, error)

	// BatchUpsertRawInventory inserts or updates multiple inventories efficiently.
	BatchUpsertRawInventory(ctx context.Context, items []model.InventoryItem) error

	// GetStats returns statistics about the inventory database.
	GetStats(ctx context.Context) (map[string]interface{}, error)

	// Close closes the repository connection.
	Close() error
}

// KeyAccountRepository defines key account data access methods.
type KeyAccountRepository interface {
	// GetKeyAccountByRobloxUser finds key_account by roblox_user_id.
	GetKeyAccountByRobloxUser(ctx context.Context, robloxUserID string) (int64, error)

	// ValidateKeyAndHWID validates a key+hwid+roblox_id combination for token generation.
	ValidateKeyAndHWID(ctx context.Context, key, hwid, robloxUserID string) (*model.KeyAccountValidation, error)
}
