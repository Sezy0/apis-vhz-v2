package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"vinzhub-rest-api-v2/internal/model"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// PostgresInventoryRepository implements InventoryRepository using PostgreSQL.
// Optimized for high-throughput with connection pooling and JSONB support.
type PostgresInventoryRepository struct {
	db *sql.DB
}

// NewPostgresInventoryRepository creates a new PostgreSQL inventory repository.
// dsn format: "postgres://user:password@host:port/dbname?sslmode=disable"
func NewPostgresInventoryRepository(dsn string) (*PostgresInventoryRepository, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open PostgreSQL: %w", err)
	}

	// Connection pool settings for high traffic
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	// Create table if not exists
	if err := createPostgresTables(db); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	log.Printf("[PostgresInventoryRepository] Initialized with pool: max=%d, idle=%d", 25, 10)
	return &PostgresInventoryRepository{db: db}, nil
}

// createPostgresTables creates the inventory table with JSONB support.
func createPostgresTables(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS fishit_inventory_raw (
		id BIGSERIAL PRIMARY KEY,
		key_account_id BIGINT DEFAULT 0,
		roblox_user_id TEXT NOT NULL UNIQUE,
		inventory_json JSONB NOT NULL,
		synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_inventory_roblox_user ON fishit_inventory_raw(roblox_user_id);
	CREATE INDEX IF NOT EXISTS idx_inventory_synced_at ON fishit_inventory_raw(synced_at);
	CREATE INDEX IF NOT EXISTS idx_inventory_key_account ON fishit_inventory_raw(key_account_id);
	`
	_, err := db.Exec(query)
	return err
}

// UpsertRawInventory inserts or updates raw JSON inventory using ON CONFLICT.
func (r *PostgresInventoryRepository) UpsertRawInventory(ctx context.Context, keyAccountID int64, robloxUserID string, rawJSON []byte) error {
	query := `
		INSERT INTO fishit_inventory_raw (key_account_id, roblox_user_id, inventory_json, synced_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (roblox_user_id) DO UPDATE SET
			key_account_id = COALESCE(EXCLUDED.key_account_id, fishit_inventory_raw.key_account_id),
			inventory_json = EXCLUDED.inventory_json,
			synced_at = NOW()`

	_, err := r.db.ExecContext(ctx, query, keyAccountID, robloxUserID, rawJSON)
	if err != nil {
		return fmt.Errorf("failed to upsert raw inventory: %w", err)
	}
	return nil
}

// BatchUpsertRawInventory inserts or updates multiple inventories efficiently.
// Uses a transaction with prepared statements for optimal performance.
func (r *PostgresInventoryRepository) BatchUpsertRawInventory(ctx context.Context, items []model.InventoryItem) error {
	if len(items) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO fishit_inventory_raw (key_account_id, roblox_user_id, inventory_json, synced_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (roblox_user_id) DO UPDATE SET
			key_account_id = COALESCE(EXCLUDED.key_account_id, fishit_inventory_raw.key_account_id),
			inventory_json = EXCLUDED.inventory_json,
			synced_at = EXCLUDED.synced_at`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, item := range items {
		_, err := stmt.ExecContext(ctx, item.KeyAccountID, item.RobloxUserID, item.RawJSON, item.SyncedAt)
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
func (r *PostgresInventoryRepository) GetRawInventory(ctx context.Context, robloxUserID string) ([]byte, *time.Time, error) {
	query := `SELECT inventory_json, synced_at FROM fishit_inventory_raw WHERE roblox_user_id = $1`

	var rawJSON []byte
	var syncedAt time.Time

	err := r.db.QueryRowContext(ctx, query, robloxUserID).Scan(&rawJSON, &syncedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("failed to get raw inventory: %w", err)
	}

	return rawJSON, &syncedAt, nil
}

// GetStats returns statistics about the inventory database.
func (r *PostgresInventoryRepository) GetStats(ctx context.Context) (map[string]interface{}, error) {
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

	// Table size (PostgreSQL specific)
	var tableSize int64
	sizeQuery := `SELECT pg_total_relation_size('fishit_inventory_raw')`
	if err := r.db.QueryRowContext(ctx, sizeQuery).Scan(&tableSize); err == nil {
		stats["db_size_bytes"] = tableSize
	}

	// Connection pool stats
	dbStats := r.db.Stats()
	stats["connections"] = map[string]interface{}{
		"open":     dbStats.OpenConnections,
		"in_use":   dbStats.InUse,
		"idle":     dbStats.Idle,
		"max_open": dbStats.MaxOpenConnections,
	}

	return stats, nil
}

// DeleteInactiveUsers deletes inventory records that haven't been synced within the threshold.
func (r *PostgresInventoryRepository) DeleteInactiveUsers(ctx context.Context, threshold time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-threshold)
	
	query := `DELETE FROM fishit_inventory_raw WHERE synced_at < $1`
	result, err := r.db.ExecContext(ctx, query, cutoffTime)
	if err != nil {
		return 0, fmt.Errorf("failed to delete inactive users: %w", err)
	}
	
	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	
	if deleted > 0 {
		log.Printf("[Postgres] Cleaned up %d inactive inventory records (threshold: %v)", deleted, threshold)
	}
	
	return deleted, nil
}

// Close closes the database connection pool.
func (r *PostgresInventoryRepository) Close() error {
	return r.db.Close()
}

// Ensure PostgresInventoryRepository implements InventoryRepository
var _ InventoryRepository = (*PostgresInventoryRepository)(nil)
