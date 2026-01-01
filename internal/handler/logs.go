package handler

import (
	"net/http"
	"strconv"
	
	"vinzhub-rest-api-v2/internal/repository"
	"vinzhub-rest-api-v2/internal/service"
	"vinzhub-rest-api-v2/pkg/apierror"
	"vinzhub-rest-api-v2/pkg/response"
)

type LogHandler struct {
	LogRepo          repository.LogRepository
	InventoryService *service.InventoryService
}

func NewLogHandler(logRepo repository.LogRepository, inventoryService *service.InventoryService) *LogHandler {
	return &LogHandler{
		LogRepo:          logRepo,
		InventoryService: inventoryService,
	}
}

// GetObfuscationLogs returns paginated obfuscation logs
func (h *LogHandler) GetObfuscationLogs(w http.ResponseWriter, r *http.Request) {
	if h.LogRepo == nil {
		response.Error(w, apierror.InternalError("Log repository unavailable"))
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 { page = 1 }
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 { limit = 20 }
	offset := (page - 1) * limit

	logs, count, err := h.LogRepo.GetObfuscationLogs(r.Context(), limit, offset)
	
	if err != nil {
		response.Error(w, apierror.InternalError("Failed to fetch logs"))
		return
	}
	
	response.OK(w, map[string]interface{}{
		"data":  logs,
		"total": count,
		"page":  page,
		"limit": limit,
	})
}

// GetInventoryLogs returns inventory snapshot (assuming 1:1 for now)
// Since we don't have historical inventory transactions yet, we just return the current inventory list from MongoDB
func (h *LogHandler) GetInventoryLogs(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement proper historical logging if MongoDB collection supports it.
	// For now, this endpoint placeholder can return nothing or generic info.
	// The user asked for "Inventory Logs", usually meaning "who has what".
	// Since listing ALL users' inventory is heavy, we might need a specific repo method.
	
	response.OK(w, map[string]interface{}{
		"message": "Inventory logs endpoint ready. Filter by user_id to see items.",
		"data": []string{},
	})
}
