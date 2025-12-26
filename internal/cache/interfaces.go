package cache

import (
	"context"
	"time"
)

// Cache defines the interface for caching operations.
// This abstraction allows swapping between memory cache (development)
// and Redis cache (production) without changing business logic.
type Cache interface {
	// Get retrieves a value by key. Returns ErrCacheMiss if not found.
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores a value with the given TTL.
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Delete removes a value by key.
	Delete(ctx context.Context, key string) error

	// Exists checks if a key exists in the cache.
	Exists(ctx context.Context, key string) (bool, error)

	// GetOrSet retrieves a value or computes and stores it if missing.
	GetOrSet(ctx context.Context, key string, ttl time.Duration, fn func() ([]byte, error)) ([]byte, error)

	// Clear removes all entries from the cache.
	Clear(ctx context.Context) error
}

// Common cache errors
type CacheError string

func (e CacheError) Error() string { return string(e) }

const (
	// ErrCacheMiss indicates the key was not found in cache.
	ErrCacheMiss CacheError = "cache miss"
)
