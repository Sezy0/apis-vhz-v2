package cache

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"vinzhub-rest-api-v2/internal/model"

	"github.com/redis/go-redis/v9"
)

// Buffer configuration
const (
	MaxBatchSize       = 20  // Reduced for slow cloud DB (Aiven)
	FlushTimeout       = 120 * time.Second  // 2 minutes for slow connections
	StaleDataThreshold = 1 * time.Hour
	CleanupInterval    = 5 * time.Minute
)

// FlushFunc is called to persist buffered data to database.
type FlushFunc func(ctx context.Context, items []*model.BufferedInventory) error

var deleteIfUnchangedScript = redis.NewScript(`
	if redis.call("HGET", KEYS[1], ARGV[1]) == ARGV[2] then
		redis.call("HDEL", KEYS[1], ARGV[1])
		redis.call("SREM", KEYS[2], ARGV[1])
		return 1
	else
		return 0
	end
`)

// RedisInventoryBuffer uses Redis for write-behind caching.
type RedisInventoryBuffer struct {
	client        *redis.Client
	flushFunc     FlushFunc
	flushTicker   *time.Ticker
	cleanupTicker *time.Ticker
	stopFlush     chan struct{}
	stopOnce      sync.Once
	keyPrefix     string
}

// RedisBufferConfig holds configuration for Redis buffer.
type RedisBufferConfig struct {
	Addr          string
	Password      string
	DB            int
	FlushInterval time.Duration
	KeyPrefix     string
}

// NewRedisInventoryBuffer creates a Redis-backed inventory buffer.
func NewRedisInventoryBuffer(cfg RedisBufferConfig, flushFunc FlushFunc) (*RedisInventoryBuffer, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     20,
		MinIdleConns: 5,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	keyPrefix := cfg.KeyPrefix
	if keyPrefix == "" {
		keyPrefix = "vinzhub:fishit:inventory"
	}

	b := &RedisInventoryBuffer{
		client:        client,
		flushFunc:     flushFunc,
		flushTicker:   time.NewTicker(cfg.FlushInterval),
		cleanupTicker: time.NewTicker(CleanupInterval),
		stopFlush:     make(chan struct{}),
		keyPrefix:     keyPrefix,
	}

	go b.backgroundFlush()
	go b.backgroundCleanup()

	log.Printf("[RedisInventoryBuffer] Started - DB:%d, prefix:%s, flush:%v, batch:%d",
		cfg.DB, keyPrefix, cfg.FlushInterval, MaxBatchSize)
	return b, nil
}

func (b *RedisInventoryBuffer) bufferKey() string {
	return b.keyPrefix + ":buffer"
}

func (b *RedisInventoryBuffer) pendingKey() string {
	return b.keyPrefix + ":pending"
}

// Add buffers an inventory update in Redis.
func (b *RedisInventoryBuffer) Add(ctx context.Context, keyAccountID int64, robloxUserID string, rawJSON []byte) error {
	data := &model.BufferedInventory{
		KeyAccountID: keyAccountID,
		RobloxUserID: robloxUserID,
		RawJSON:      rawJSON,
		UpdatedAt:    time.Now(),
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	pipe := b.client.Pipeline()
	pipe.HSet(ctx, b.bufferKey(), robloxUserID, jsonData)
	pipe.SAdd(ctx, b.pendingKey(), robloxUserID)
	_, err = pipe.Exec(ctx)
	return err
}

// Get retrieves a buffered inventory from Redis.
func (b *RedisInventoryBuffer) Get(ctx context.Context, robloxUserID string) (*model.BufferedInventory, error) {
	data, err := b.client.HGet(ctx, b.bufferKey(), robloxUserID).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var inv model.BufferedInventory
	if err := json.Unmarshal(data, &inv); err != nil {
		return nil, err
	}

	return &inv, nil
}

// Count returns the number of pending items.
func (b *RedisInventoryBuffer) Count(ctx context.Context) (int64, error) {
	return b.client.SCard(ctx, b.pendingKey()).Result()
}

// FlushBatch writes up to MaxBatchSize items to the database.
func (b *RedisInventoryBuffer) FlushBatch(ctx context.Context) (int, error) {
	userIDs, err := b.client.SRandMemberN(ctx, b.pendingKey(), MaxBatchSize).Result()
	if err != nil {
		return 0, err
	}

	if len(userIDs) == 0 {
		return 0, nil
	}

	totalPending, _ := b.Count(ctx)
	log.Printf("[RedisInventoryBuffer] Flushing %d/%d items", len(userIDs), totalPending)

	items := make([]*model.BufferedInventory, 0, len(userIDs))
	originalData := make(map[string]string)

	for _, userID := range userIDs {
		data, err := b.client.HGet(ctx, b.bufferKey(), userID).Bytes()
		if err == redis.Nil {
			b.client.SRem(ctx, b.pendingKey(), userID)
			continue
		}
		if err != nil {
			log.Printf("[RedisInventoryBuffer] Error getting %s: %v", userID, err)
			continue
		}

		originalData[userID] = string(data)

		var inv model.BufferedInventory
		if err := json.Unmarshal(data, &inv); err != nil {
			log.Printf("[RedisInventoryBuffer] Error unmarshaling %s: %v", userID, err)
			b.client.HDel(ctx, b.bufferKey(), userID)
			b.client.SRem(ctx, b.pendingKey(), userID)
			continue
		}
		items = append(items, &inv)
	}

	if len(items) == 0 {
		return 0, nil
	}

	if err := b.flushFunc(ctx, items); err != nil {
		log.Printf("[RedisInventoryBuffer] Flush error: %v", err)
		return 0, err
	}

	pipe := b.client.Pipeline()
	for userID, rawJSON := range originalData {
		deleteIfUnchangedScript.Run(ctx, pipe, []string{b.bufferKey(), b.pendingKey()}, userID, rawJSON)
	}
	_, err = pipe.Exec(ctx)
	if err != nil {
		log.Printf("[RedisInventoryBuffer] Error clearing Redis: %v", err)
	}

	log.Printf("[RedisInventoryBuffer] Successfully flushed %d items", len(items))
	return len(items), nil
}

// Flush writes all buffered items to database.
func (b *RedisInventoryBuffer) Flush(ctx context.Context) error {
	_, err := b.FlushBatch(ctx)
	return err
}

// CleanupStale removes inventory data older than StaleDataThreshold.
func (b *RedisInventoryBuffer) CleanupStale(ctx context.Context) (int, error) {
	userIDs, err := b.client.SMembers(ctx, b.pendingKey()).Result()
	if err != nil {
		return 0, err
	}

	if len(userIDs) == 0 {
		return 0, nil
	}

	staleThreshold := time.Now().Add(-StaleDataThreshold)
	staleCount := 0
	pipe := b.client.Pipeline()

	for _, userID := range userIDs {
		data, err := b.client.HGet(ctx, b.bufferKey(), userID).Bytes()
		if err == redis.Nil {
			pipe.SRem(ctx, b.pendingKey(), userID)
			continue
		}
		if err != nil {
			continue
		}

		var inv model.BufferedInventory
		if err := json.Unmarshal(data, &inv); err != nil {
			pipe.HDel(ctx, b.bufferKey(), userID)
			pipe.SRem(ctx, b.pendingKey(), userID)
			staleCount++
			continue
		}

		if inv.UpdatedAt.Before(staleThreshold) {
			pipe.HDel(ctx, b.bufferKey(), userID)
			pipe.SRem(ctx, b.pendingKey(), userID)
			staleCount++
		}
	}

	if staleCount > 0 {
		_, err = pipe.Exec(ctx)
		if err != nil {
			log.Printf("[RedisInventoryBuffer] Cleanup exec error: %v", err)
			return 0, err
		}
		log.Printf("[RedisInventoryBuffer] Cleaned up %d stale items", staleCount)
	}

	return staleCount, nil
}

func (b *RedisInventoryBuffer) backgroundFlush() {
	for {
		select {
		case <-b.flushTicker.C:
			ctx, cancel := context.WithTimeout(context.Background(), FlushTimeout)
			if _, err := b.FlushBatch(ctx); err != nil {
				log.Printf("[RedisInventoryBuffer] Background flush error: %v", err)
			}
			cancel()
		case <-b.stopFlush:
			log.Printf("[RedisInventoryBuffer] Shutdown: flushing remaining items...")
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			for {
				flushed, err := b.FlushBatch(ctx)
				if err != nil {
					log.Printf("[RedisInventoryBuffer] Shutdown flush error: %v", err)
					break
				}
				if flushed == 0 {
					break
				}
			}
			cancel()
			log.Printf("[RedisInventoryBuffer] Shutdown flush complete")
			return
		}
	}
}

func (b *RedisInventoryBuffer) backgroundCleanup() {
	for {
		select {
		case <-b.cleanupTicker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			b.CleanupStale(ctx)
			cancel()
		case <-b.stopFlush:
			return
		}
	}
}

// Close stops the buffer and performs a final flush.
func (b *RedisInventoryBuffer) Close() error {
	b.stopOnce.Do(func() {
		b.flushTicker.Stop()
		b.cleanupTicker.Stop()
		close(b.stopFlush)
	})
	return b.client.Close()
}
