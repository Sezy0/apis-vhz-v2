package handler

import (
	"net/http"
	"time"

	"vinzhub-rest-api-v2/pkg/response"
)

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
