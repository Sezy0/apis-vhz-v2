package service

import (
	"context"
	"log"
	"sync"
	"time"

	"vinzhub-rest-api-v2/internal/repository"
)

// CleanupConfig holds configuration for the cleanup scheduler.
type CleanupConfig struct {
	// InactiveThreshold is the duration after which inactive users are deleted.
	// Default: 30 days
	InactiveThreshold time.Duration

	// CleanupInterval is how often the cleanup runs.
	// Default: 24 hours
	CleanupInterval time.Duration
}

// DefaultCleanupConfig returns default cleanup configuration.
// Aggressive cleanup for limited MongoDB storage.
func DefaultCleanupConfig() CleanupConfig {
	return CleanupConfig{
		InactiveThreshold: 1 * time.Hour,      // 1 hour - user inactive for 1 hour gets deleted
		CleanupInterval:   10 * time.Minute,   // Run every 10 minutes
	}
}

// CleanupScheduler runs periodic cleanup of inactive inventory data.
type CleanupScheduler struct {
	repo      repository.InventoryRepository
	config    CleanupConfig
	ticker    *time.Ticker
	stopCh    chan struct{}
	stopOnce  sync.Once
	isRunning bool
	mu        sync.Mutex
}

// NewCleanupScheduler creates a new cleanup scheduler.
func NewCleanupScheduler(repo repository.InventoryRepository, config CleanupConfig) *CleanupScheduler {
	if config.InactiveThreshold == 0 {
		config.InactiveThreshold = 30 * 24 * time.Hour
	}
	if config.CleanupInterval == 0 {
		config.CleanupInterval = 24 * time.Hour
	}

	return &CleanupScheduler{
		repo:   repo,
		config: config,
		stopCh: make(chan struct{}),
	}
}

// Start begins the cleanup scheduler.
func (s *CleanupScheduler) Start() {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return
	}
	s.isRunning = true
	s.ticker = time.NewTicker(s.config.CleanupInterval)
	s.mu.Unlock()

	log.Printf("[CleanupScheduler] Started - Interval: %v, Threshold: %v",
		s.config.CleanupInterval, s.config.InactiveThreshold)

	// Run initial cleanup after a short delay
	go func() {
		time.Sleep(1 * time.Minute) // Wait 1 minute after startup
		s.runCleanup()
	}()

	go s.run()
}

// run is the main cleanup loop.
func (s *CleanupScheduler) run() {
	for {
		select {
		case <-s.ticker.C:
			s.runCleanup()
		case <-s.stopCh:
			log.Printf("[CleanupScheduler] Stopped")
			return
		}
	}
}

// runCleanup performs the actual cleanup.
func (s *CleanupScheduler) runCleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log.Printf("[CleanupScheduler] Running cleanup for inactive users (threshold: %v)", s.config.InactiveThreshold)

	deleted, err := s.repo.DeleteInactiveUsers(ctx, s.config.InactiveThreshold)
	if err != nil {
		log.Printf("[CleanupScheduler] Error during cleanup: %v", err)
		return
	}

	if deleted > 0 {
		log.Printf("[CleanupScheduler] Cleaned up %d inactive inventory records", deleted)
	} else {
		log.Printf("[CleanupScheduler] No inactive records to clean up")
	}
}

// Stop stops the cleanup scheduler.
func (s *CleanupScheduler) Stop() {
	s.stopOnce.Do(func() {
		s.mu.Lock()
		defer s.mu.Unlock()

		if s.ticker != nil {
			s.ticker.Stop()
		}
		close(s.stopCh)
		s.isRunning = false
	})
}

// RunNow triggers an immediate cleanup run.
func (s *CleanupScheduler) RunNow() (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	return s.repo.DeleteInactiveUsers(ctx, s.config.InactiveThreshold)
}
