package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"vinzhub-rest-api-v2/internal/service"
	"vinzhub-rest-api-v2/pkg/apierror"
	"vinzhub-rest-api-v2/pkg/response"

	"github.com/go-chi/chi/v5"
)

// InventoryHandler handles inventory-related HTTP requests.
type InventoryHandler struct {
	inventoryService *service.InventoryService
}

// NewInventoryHandler creates a new inventory handler.
func NewInventoryHandler(inventoryService *service.InventoryService) *InventoryHandler {
	return &InventoryHandler{
		inventoryService: inventoryService,
	}
}

// SyncRawInventory handles POST /api/v1/inventory/{roblox_user_id}/sync
func (h *InventoryHandler) SyncRawInventory(w http.ResponseWriter, r *http.Request) {
	robloxUserID := chi.URLParam(r, "roblox_user_id")
	if robloxUserID == "" {
		response.Error(w, apierror.BadRequest("roblox_user_id is required"))
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		response.Error(w, apierror.BadRequest("failed to read request body"))
		return
	}
	defer r.Body.Close()

	var jsonData json.RawMessage
	if err := json.Unmarshal(body, &jsonData); err != nil {
		response.Error(w, apierror.BadRequest("invalid JSON"))
		return
	}

	err = h.inventoryService.SyncRawInventory(r.Context(), robloxUserID, body)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.OK(w, map[string]interface{}{
		"status":  "synced",
		"user_id": robloxUserID,
		"size":    len(body),
	})
}

// GetRawInventory handles GET /api/v1/inventory/{roblox_user_id}
func (h *InventoryHandler) GetRawInventory(w http.ResponseWriter, r *http.Request) {
	robloxUserID := chi.URLParam(r, "roblox_user_id")
	if robloxUserID == "" {
		response.Error(w, apierror.BadRequest("roblox_user_id is required"))
		return
	}

	data, syncedAt, err := h.inventoryService.GetRawInventory(r.Context(), robloxUserID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.OK(w, map[string]interface{}{
		"roblox_user_id": robloxUserID,
		"inventory":      json.RawMessage(data),
		"synced_at":      syncedAt,
	})
}
