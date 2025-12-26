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
// Returns key_account details if valid, error otherwise.
func (r *MySQLKeyAccountRepository) ValidateKeyAndHWID(ctx context.Context, key, hwid, robloxUserID string) (*model.KeyAccountValidation, error) {
	log.Printf("[KeyAccountRepository] Validating key for roblox_id=%s", robloxUserID)

	query := `
		SELECT 
			ka.id as key_account_id,
			ka.key_id,
			ka.roblox_user_id,
			ka.roblox_username,
			ka.hwid,
			k.status as key_status
		FROM key_accounts ka
		JOIN ` + "`keys`" + ` k ON ka.key_id = k.id
		WHERE k.` + "`key`" + ` = ?
		  AND ka.roblox_user_id = ?
		  AND ka.is_active = 1
		  AND LOWER(k.status) = 'active'
		LIMIT 1`

	var result model.KeyAccountValidation
	err := r.db.QueryRowContext(ctx, query, key, robloxUserID).Scan(
		&result.KeyAccountID,
		&result.KeyID,
		&result.RobloxUserID,
		&result.RobloxUsername,
		&result.HWID,
		&result.KeyStatus,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid key or account not found")
		}
		return nil, fmt.Errorf("failed to validate key: %w", err)
	}

	// Validate HWID if already set (not empty)
	if result.HWID != "" && result.HWID != hwid {
		return nil, fmt.Errorf("hwid mismatch")
	}

	// Update HWID if not set yet
	if result.HWID == "" && hwid != "" {
		updateQuery := `UPDATE key_accounts SET hwid = ? WHERE id = ?`
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
