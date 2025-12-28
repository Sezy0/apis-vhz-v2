package handler

import (
	"net/http"
	"runtime"
	"time"

	"vinzhub-rest-api-v2/pkg/response"
)

// StartTime tracks when the server started for uptime calculation
var StartTime = time.Now()

// Handler contains shared HTTP handlers and their dependencies.
type Handler struct{}

// New creates a new handler.
func New() *Handler {
	return &Handler{}
}

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
}

// Health handles GET /api/v1/health
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC(),
		Version:   "2.0.0",
	}
	response.OK(w, resp)
}

// ReadyResponse represents the readiness check response.
type ReadyResponse struct {
	Ready     bool      `json:"ready"`
	Timestamp time.Time `json:"timestamp"`
	Checks    []Check   `json:"checks"`
}

// Check represents an individual readiness check.
type Check struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// Ready handles GET /api/v1/ready
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	checks := []Check{
		{Name: "api", Status: "ok"},
	}

	allReady := true
	for _, check := range checks {
		if check.Status != "ok" {
			allReady = false
			break
		}
	}

	resp := ReadyResponse{
		Ready:     allReady,
		Timestamp: time.Now().UTC(),
		Checks:    checks,
	}

	if !allReady {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	response.OK(w, resp)
}

// StatusChecks represents the checks in status response
type StatusChecks struct {
	Database string  `json:"database"`
	MemoryMB float64 `json:"memory_mb"`
}

// StatusResponse represents the unified status response for bot monitoring
type StatusResponse struct {
	Service       string       `json:"service"`
	Status        string       `json:"status"`
	Timestamp     string       `json:"timestamp"`
	UptimeSeconds int64        `json:"uptime_seconds"`
	PingMS        int64        `json:"ping_ms"`
	Checks        StatusChecks `json:"checks"`
}

// Status handles GET /api/status - unified health check for bot monitoring
func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	requestStart := time.Now()

	// Get memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	memoryMB := float64(memStats.Alloc) / 1024 / 1024

	// Calculate metrics
	pingMS := time.Since(requestStart).Milliseconds()
	uptimeSeconds := int64(time.Since(StartTime).Seconds())

	resp := StatusResponse{
		Service:       "vinzhub-rest-api",
		Status:        "ok",
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		UptimeSeconds: uptimeSeconds,
		PingMS:        pingMS,
		Checks: StatusChecks{
			Database: "ok", // API doesn't have direct DB - always ok
			MemoryMB: float64(int(memoryMB*100)) / 100,
		},
	}

	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	response.OK(w, resp)
}
