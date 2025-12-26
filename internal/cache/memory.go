package cache

import (
	"context"
	"sync"
	"time"
)

// cacheEntry represents a cached value with expiration.
type cacheEntry struct {
	value     []byte
	expiresAt time.Time
}

// isExpired checks if the entry has expired.
func (e *cacheEntry) isExpired() bool {
	return time.Now().After(e.expiresAt)
}

// MemoryCache is an in-memory implementation of Cache.
// Use this for development/testing or single-instance deployments.
type MemoryCache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry

	cleanupInterval time.Duration
	stopCleanup     chan struct{}
}

// NewMemoryCache creates a new in-memory cache with automatic cleanup.
func NewMemoryCache() *MemoryCache {
	c := &MemoryCache{
		entries:         make(map[string]*cacheEntry),
		cleanupInterval: time.Minute,
		stopCleanup:     make(chan struct{}),
	}

	go c.cleanup()

	return c
}

// Get retrieves a value by key.
func (c *MemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists || entry.isExpired() {
		return nil, ErrCacheMiss
	}

	result := make([]byte, len(entry.value))
	copy(result, entry.value)
	return result, nil
}

// Set stores a value with the given TTL.
func (c *MemoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)

	c.entries[key] = &cacheEntry{
		value:     valueCopy,
		expiresAt: time.Now().Add(ttl),
	}

	return nil
}

// Delete removes a value by key.
func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, key)
	return nil
}

// Exists checks if a key exists and is not expired.
func (c *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists || entry.isExpired() {
		return false, nil
	}

	return true, nil
}

// GetOrSet retrieves a value or computes and stores it if missing.
func (c *MemoryCache) GetOrSet(ctx context.Context, key string, ttl time.Duration, fn func() ([]byte, error)) ([]byte, error) {
	if value, err := c.Get(ctx, key); err == nil {
		return value, nil
	}

	value, err := fn()
	if err != nil {
		return nil, err
	}

	if err := c.Set(ctx, key, value, ttl); err != nil {
		return nil, err
	}

	return value, nil
}

// Clear removes all entries from the cache.
func (c *MemoryCache) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*cacheEntry)
	return nil
}

// Close stops the background cleanup goroutine.
func (c *MemoryCache) Close() error {
	close(c.stopCleanup)
	return nil
}

// cleanup periodically removes expired entries.
func (c *MemoryCache) cleanup() {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.removeExpired()
		case <-c.stopCleanup:
			return
		}
	}
}

// removeExpired removes all expired entries.
func (c *MemoryCache) removeExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, entry := range c.entries {
		if entry.isExpired() {
			delete(c.entries, key)
		}
	}
}
