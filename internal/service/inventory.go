package service

import (
	"context"
	"time"

	"vinzhub-rest-api-v2/internal/cache"
	"vinzhub-rest-api-v2/internal/model"
	"vinzhub-rest-api-v2/internal/repository"
)

// InventoryService handles inventory business logic.
type InventoryService struct {
	inventoryRepo  repository.InventoryRepository
	keyAccountRepo repository.KeyAccountRepository
	buffer         *cache.RedisInventoryBuffer
}

// NewInventoryService creates a new inventory service.
// Returns nil if inventoryRepo is nil (required dependency).
func NewInventoryService(
	inventoryRepo repository.InventoryRepository,
	keyAccountRepo repository.KeyAccountRepository,
) *InventoryService {
	if inventoryRepo == nil {
		return nil
	}
	return &InventoryService{
		inventoryRepo:  inventoryRepo,
		keyAccountRepo: keyAccountRepo,
	}
}

// NewInventoryServiceWithBuffer creates a new inventory service with Redis buffer.
// Redis buffer is REQUIRED. inventoryRepo can be nil (Redis-only mode).
func NewInventoryServiceWithBuffer(
	inventoryRepo repository.InventoryRepository,
	keyAccountRepo repository.KeyAccountRepository,
	buffer *cache.RedisInventoryBuffer,
) *InventoryService {
	if buffer == nil {
		return nil
	}
	return &InventoryService{
		inventoryRepo:  inventoryRepo,
		keyAccountRepo: keyAccountRepo,
		buffer:         buffer,
	}
}

// SetBuffer sets the Redis buffer for write-behind caching.
func (s *InventoryService) SetBuffer(buffer *cache.RedisInventoryBuffer) {
	s.buffer = buffer
}

// SyncRawInventory stores raw JSON inventory data.
// If buffer is set, writes to Redis first (fast), otherwise direct to DB.
func (s *InventoryService) SyncRawInventory(ctx context.Context, robloxUserID string, rawJSON []byte) error {
	// Get key account ID (optional - can be 0 if not linked or repo unavailable)
	var keyAccountID int64
	if s.keyAccountRepo != nil {
		keyAccountID, _ = s.keyAccountRepo.GetKeyAccountByRobloxUser(ctx, robloxUserID)
	}

	// If buffer is available, use write-behind caching
	if s.buffer != nil {
		return s.buffer.Add(ctx, keyAccountID, robloxUserID, rawJSON)
	}

	// Fallback to direct DB write
	return s.inventoryRepo.UpsertRawInventory(ctx, keyAccountID, robloxUserID, rawJSON)
}

// GetRawInventory retrieves raw JSON inventory data.
// Checks Redis buffer first, then falls back to database.
func (s *InventoryService) GetRawInventory(ctx context.Context, robloxUserID string) ([]byte, *time.Time, error) {
	// Check buffer first
	if s.buffer != nil {
		if inv, err := s.buffer.Get(ctx, robloxUserID); err == nil && inv != nil {
			return inv.RawJSON, &inv.UpdatedAt, nil
		}
	}

	// Fall back to database
	return s.inventoryRepo.GetRawInventory(ctx, robloxUserID)
}

// CreateFlushFunc creates a flush function for the Redis buffer.
func CreateFlushFunc(repo repository.InventoryRepository) cache.FlushFunc {
	return func(ctx context.Context, items []*model.BufferedInventory) error {
		inventoryItems := make([]model.InventoryItem, len(items))
		for i, item := range items {
			inventoryItems[i] = model.InventoryItem{
				KeyAccountID: item.KeyAccountID,
				RobloxUserID: item.RobloxUserID,
				RawJSON:      item.RawJSON,
				SyncedAt:     item.UpdatedAt,
			}
		}
		return repo.BatchUpsertRawInventory(ctx, inventoryItems)
	}
}
