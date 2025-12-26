package model

import "time"

// RawInventory represents raw JSON inventory data.
type RawInventory struct {
	ID            int64     `json:"id"`
	KeyAccountID  int64     `json:"key_account_id"`
	RobloxUserID  string    `json:"roblox_user_id"`
	InventoryJSON []byte    `json:"inventory_json"`
	SyncedAt      time.Time `json:"synced_at"`
	CreatedAt     time.Time `json:"created_at"`
}

// InventoryItem represents a single inventory record for batch operations.
type InventoryItem struct {
	KeyAccountID int64
	RobloxUserID string
	RawJSON      []byte
	SyncedAt     time.Time
}

// BufferedInventory represents a pending inventory update in the buffer.
type BufferedInventory struct {
	KeyAccountID int64     `json:"key_account_id"`
	RobloxUserID string    `json:"roblox_user_id"`
	RawJSON      []byte    `json:"raw_json"`
	UpdatedAt    time.Time `json:"updated_at"`
}
