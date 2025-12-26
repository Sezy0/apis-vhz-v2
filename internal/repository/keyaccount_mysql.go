package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"vinzhub-rest-api-v2/internal/model"
)

// MySQLKeyAccountRepository implements KeyAccountRepository using MySQL.
type MySQLKeyAccountRepository struct {
	db *sql.DB
}

// NewMySQLKeyAccountRepository creates a new MySQL key account repository.
func NewMySQLKeyAccountRepository(db *sql.DB) *MySQLKeyAccountRepository {
	return &MySQLKeyAccountRepository{db: db}
}

// GetKeyAccountByRobloxUser finds key_account by roblox_user_id.
func (r *MySQLKeyAccountRepository) GetKeyAccountByRobloxUser(ctx context.Context, robloxUserID string) (int64, error) {
	query := `SELECT id FROM key_accounts WHERE roblox_user_id = ? AND is_active = 1 LIMIT 1`

	var id int64
	err := r.db.QueryRowContext(ctx, query, robloxUserID).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("key account not found for roblox user: %s", robloxUserID)
		}
		return 0, fmt.Errorf("failed to get key account: %w", err)
	}

	return id, nil
}

// ValidateKeyAccount checks if key_account_id exists and is active.
func (r *MySQLKeyAccountRepository) ValidateKeyAccount(ctx context.Context, keyAccountID int64) (bool, error) {
	query := `SELECT COUNT(*) FROM key_accounts WHERE id = ? AND is_active = 1`

	var count int
	err := r.db.QueryRowContext(ctx, query, keyAccountID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to validate key account: %w", err)
	}

	return count > 0, nil
}

// ValidateKeyAndHWID validates a key+hwid+roblox_id combination for token generation.
// If key is valid but key_account doesn't exist, auto-creates one.
func (r *MySQLKeyAccountRepository) ValidateKeyAndHWID(ctx context.Context, key, hwid, robloxUserID string) (*model.KeyAccountValidation, error) {
	log.Printf("[KeyAccountRepository] Validating key for roblox_id=%s", robloxUserID)

	// Step 1: Check if key exists and is active
	var keyID int64
	var keyStatus string
	keyQuery := "SELECT id, status FROM `keys` WHERE `key` = ? LIMIT 1"
	err := r.db.QueryRowContext(ctx, keyQuery, key).Scan(&keyID, &keyStatus)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid key or account not found")
		}
		return nil, fmt.Errorf("failed to validate key: %w", err)
	}

	if keyStatus != "active" {
		return nil, fmt.Errorf("key is not active (status: %s)", keyStatus)
	}

	// Step 2: Check if key_account exists for this key+roblox_id
	var result model.KeyAccountValidation
	accountQuery := `
		SELECT id, key_id, roblox_user_id, COALESCE(roblox_username, ''), COALESCE(hwid, '')
		FROM key_accounts 
		WHERE key_id = ? AND roblox_user_id = ? AND is_active = 1
		LIMIT 1`
	
	err = r.db.QueryRowContext(ctx, accountQuery, keyID, robloxUserID).Scan(
		&result.KeyAccountID,
		&result.KeyID,
		&result.RobloxUserID,
		&result.RobloxUsername,
		&result.HWID,
	)

	if err == sql.ErrNoRows {
		// Step 3: Auto-create key_account if it doesn't exist
		log.Printf("[KeyAccountRepository] Auto-creating key_account for key_id=%d, roblox_id=%s", keyID, robloxUserID)
		
		insertQuery := `
			INSERT INTO key_accounts (key_id, roblox_user_id, hwid, is_active, first_used_at, last_used_at)
			VALUES (?, ?, ?, 1, NOW(), NOW())`
		
		res, err := r.db.ExecContext(ctx, insertQuery, keyID, robloxUserID, hwid)
		if err != nil {
			return nil, fmt.Errorf("failed to create key account: %w", err)
		}
		
		newID, _ := res.LastInsertId()
		result = model.KeyAccountValidation{
			KeyAccountID: newID,
			KeyID:        keyID,
			RobloxUserID: robloxUserID,
			HWID:         hwid,
			KeyStatus:    keyStatus,
		}
		return &result, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to query key account: %w", err)
	}

	result.KeyStatus = keyStatus

	// Note: HWID validation is handled by PHP API (primary validator)
	// Go API just updates HWID to stay in sync with latest session
	// This allows users who validated via loader to use InventorySync

	// Update HWID and last_used_at
	if hwid != "" {
		updateQuery := `UPDATE key_accounts SET hwid = ?, last_used_at = NOW() WHERE id = ?`
		_, err = r.db.ExecContext(ctx, updateQuery, hwid, result.KeyAccountID)
		if err != nil {
			log.Printf("[KeyAccountRepository] Failed to update HWID: %v", err)
		}
		result.HWID = hwid
	}

	return &result, nil
}

// Ensure MySQLKeyAccountRepository implements KeyAccountRepository
var _ KeyAccountRepository = (*MySQLKeyAccountRepository)(nil)
