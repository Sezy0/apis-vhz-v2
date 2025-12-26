package handler

import (
	"net/http"
	"runtime"
	"time"

	"vinzhub-rest-api-v2/internal/cache"
	"vinzhub-rest-api-v2/internal/repository"
	"vinzhub-rest-api-v2/pkg/response"
)

// AdminHandler handles admin-related HTTP requests.
type AdminHandler struct {
	redisBuffer   *cache.RedisInventoryBuffer
	inventoryRepo repository.InventoryRepository // Interface instead of concrete type
	dbType        string                          // Database type: sqlite, postgres, mongodb
	startTime     time.Time
}

// NewAdminHandler creates a new admin handler.
func NewAdminHandler(
	redisBuffer *cache.RedisInventoryBuffer,
	inventoryRepo repository.InventoryRepository,
	dbType string,
) *AdminHandler {
	return &AdminHandler{
		redisBuffer:   redisBuffer,
		inventoryRepo: inventoryRepo,
		dbType:        dbType,
		startTime:     time.Now(),
	}
}

// GetStats handles GET /api/v1/admin/stats
func (h *AdminHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	stats := make(map[string]interface{})

	// System info
	stats["uptime_seconds"] = int64(time.Since(h.startTime).Seconds())
	stats["uptime_human"] = time.Since(h.startTime).Round(time.Second).String()
	stats["server_time"] = time.Now().Format(time.RFC3339)
	stats["db_type"] = h.dbType // sqlite, postgres, or mongodb

	// Memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	stats["memory"] = map[string]interface{}{
		"alloc_mb":       float64(memStats.Alloc) / 1024 / 1024,
		"total_alloc_mb": float64(memStats.TotalAlloc) / 1024 / 1024,
		"sys_mb":         float64(memStats.Sys) / 1024 / 1024,
		"heap_alloc_mb":  float64(memStats.HeapAlloc) / 1024 / 1024,
		"heap_inuse_mb":  float64(memStats.HeapInuse) / 1024 / 1024,
		"num_gc":         memStats.NumGC,
		"goroutines":     runtime.NumGoroutine(),
	}

	// Redis buffer stats
	if h.redisBuffer != nil {
		count, err := h.redisBuffer.Count(ctx)
		if err == nil {
			stats["redis_buffer"] = map[string]interface{}{
				"pending_items": count,
				"status":        "connected",
			}
		} else {
			stats["redis_buffer"] = map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			}
		}
	} else {
		stats["redis_buffer"] = map[string]interface{}{
			"status": "not_configured",
		}
	}

	// SQLite stats
	if h.inventoryRepo != nil {
		sqliteStats, err := h.inventoryRepo.GetStats(ctx)
		if err == nil {
			stats["sqlite"] = sqliteStats
			stats["sqlite"].(map[string]interface{})["status"] = "connected"
		} else {
			stats["sqlite"] = map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			}
		}
	} else {
		stats["sqlite"] = map[string]interface{}{
			"status": "not_configured",
		}
	}

	// Runtime info
	stats["runtime"] = map[string]interface{}{
		"go_version": runtime.Version(),
		"os":         runtime.GOOS,
		"arch":       runtime.GOARCH,
		"cpus":       runtime.NumCPU(),
	}

	response.OK(w, stats)
}

// GetHealth handles GET /api/v1/admin/health
func (h *AdminHandler) GetHealth(w http.ResponseWriter, r *http.Request) {
	response.OK(w, map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}
