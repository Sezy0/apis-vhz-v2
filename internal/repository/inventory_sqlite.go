package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"vinzhub-rest-api-v2/internal/model"

	_ "modernc.org/sqlite" // Pure Go SQLite driver - no CGO required
)

// SQLiteInventoryRepository implements InventoryRepository using SQLite.
// Thread-safe with WAL mode for high-concurrency reads.
type SQLiteInventoryRepository struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewSQLiteInventoryRepository creates a new SQLite inventory repository.
// dbPath is the path to the SQLite database file (e.g., "./data/inventory.db")
func NewSQLiteInventoryRepository(dbPath string) (*SQLiteInventoryRepository, error) {
	// Open with WAL mode and other optimizations
	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_synchronous=NORMAL&_cache_size=10000&_busy_timeout=5000", dbPath)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite: %w", err)
	}

	// SQLite connection pool settings
	db.SetMaxOpenConns(1) // SQLite only supports 1 writer
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0) // Keep connection alive

	// Create table if not exists
	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	log.Printf("[SQLiteInventoryRepository] Initialized with database: %s", dbPath)
	return &SQLiteInventoryRepository{db: db}, nil
}

// createTables creates the inventory table.
func createTables(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS fishit_inventory_raw (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		key_account_id INTEGER DEFAULT 0,
		roblox_user_id TEXT NOT NULL UNIQUE,
		inventory_json TEXT NOT NULL,
		synced_at DATETIME NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_roblox_user ON fishit_inventory_raw(roblox_user_id);
	CREATE INDEX IF NOT EXISTS idx_synced_at ON fishit_inventory_raw(synced_at);
	`
	_, err := db.Exec(query)
	return err
}

// UpsertRawInventory inserts or updates raw JSON inventory.
func (r *SQLiteInventoryRepository) UpsertRawInventory(ctx context.Context, keyAccountID int64, robloxUserID string, rawJSON []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	query := `
		INSERT INTO fishit_inventory_raw (key_account_id, roblox_user_id, inventory_json, synced_at)
		VALUES (?, ?, ?, datetime('now'))
		ON CONFLICT(roblox_user_id) DO UPDATE SET
			key_account_id = COALESCE(excluded.key_account_id, key_account_id),
			inventory_json = excluded.inventory_json,
			synced_at = datetime('now')`

	_, err := r.db.ExecContext(ctx, query, keyAccountID, robloxUserID, string(rawJSON))
	if err != nil {
		return fmt.Errorf("failed to upsert raw inventory: %w", err)
	}
	return nil
}

// BatchUpsertRawInventory inserts or updates multiple inventories efficiently.
func (r *SQLiteInventoryRepository) BatchUpsertRawInventory(ctx context.Context, items []model.InventoryItem) error {
	if len(items) == 0 {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO fishit_inventory_raw (key_account_id, roblox_user_id, inventory_json, synced_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(roblox_user_id) DO UPDATE SET
			key_account_id = COALESCE(excluded.key_account_id, key_account_id),
			inventory_json = excluded.inventory_json,
			synced_at = excluded.synced_at`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, item := range items {
		_, err := stmt.ExecContext(ctx, item.KeyAccountID, item.RobloxUserID, string(item.RawJSON), item.SyncedAt)
		if err != nil {
			return fmt.Errorf("failed to batch upsert item %s: %w", item.RobloxUserID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// GetRawInventory retrieves raw JSON inventory by Roblox user ID.
func (r *SQLiteInventoryRepository) GetRawInventory(ctx context.Context, robloxUserID string) ([]byte, *time.Time, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query := `SELECT inventory_json, synced_at FROM fishit_inventory_raw WHERE roblox_user_id = ?`

	var rawJSON string
	var syncedAt time.Time

	err := r.db.QueryRowContext(ctx, query, robloxUserID).Scan(&rawJSON, &syncedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("failed to get raw inventory: %w", err)
	}

	return []byte(rawJSON), &syncedAt, nil
}

// GetStats returns statistics about the inventory database.
func (r *SQLiteInventoryRepository) GetStats(ctx context.Context) (map[string]interface{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := make(map[string]interface{})

	// Total count
	var count int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM fishit_inventory_raw").Scan(&count); err != nil {
		return nil, err
	}
	stats["total_inventories"] = count

	// Last sync time
	var lastSync sql.NullTime
	if err := r.db.QueryRowContext(ctx, "SELECT MAX(synced_at) FROM fishit_inventory_raw").Scan(&lastSync); err == nil && lastSync.Valid {
		stats["last_sync"] = lastSync.Time
	}

	// Database file size (approximate from page count)
	var pageCount, pageSize int64
	r.db.QueryRowContext(ctx, "PRAGMA page_count").Scan(&pageCount)
	r.db.QueryRowContext(ctx, "PRAGMA page_size").Scan(&pageSize)
	stats["db_size_bytes"] = pageCount * pageSize

	return stats, nil
}

// DeleteInactiveUsers deletes inventory records that haven't been synced within the threshold.
func (r *SQLiteInventoryRepository) DeleteInactiveUsers(ctx context.Context, threshold time.Duration) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	cutoffTime := time.Now().Add(-threshold)
	
	query := `DELETE FROM fishit_inventory_raw WHERE synced_at < ?`
	result, err := r.db.ExecContext(ctx, query, cutoffTime)
	if err != nil {
		return 0, fmt.Errorf("failed to delete inactive users: %w", err)
	}
	
	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	
	if deleted > 0 {
		log.Printf("[SQLite] Cleaned up %d inactive inventory records (threshold: %v)", deleted, threshold)
	}
	
	return deleted, nil
}

// Close closes the database connection.
func (r *SQLiteInventoryRepository) Close() error {
	return r.db.Close()
}

// Ensure SQLiteInventoryRepository implements InventoryRepository
var _ InventoryRepository = (*SQLiteInventoryRepository)(nil)
