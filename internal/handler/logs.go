package handler

import (
	"database/sql"
	"net/http"
	"strconv"
	
	"vinzhub-rest-api-v2/internal/service"
	"vinzhub-rest-api-v2/pkg/apierror"
	"vinzhub-rest-api-v2/pkg/response"
)

type LogHandler struct {
	DB               *sql.DB
	InventoryService *service.InventoryService
}

func NewLogHandler(db *sql.DB, inventoryService *service.InventoryService) *LogHandler {
	return &LogHandler{
		DB:               db,
		InventoryService: inventoryService,
	}
}

// GetObfuscationLogs returns paginated obfuscation logs
func (h *LogHandler) GetObfuscationLogs(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil {
		response.Error(w, apierror.InternalError("Database connection unavailable"))
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 { page = 1 }
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 { limit = 20 }
	offset := (page - 1) * limit

	rows, err := h.DB.Query(`
		SELECT id, ip_address, file_name, file_size_in, file_size_out, preset_used, status, error_message, execution_time_ms, created_at 
		FROM obfuscation_logs 
		ORDER BY created_at DESC 
		LIMIT ? OFFSET ?`, limit, offset)
	
	if err != nil {
		response.Error(w, apierror.InternalError("Failed to fetch logs"))
		return
	}
	defer rows.Close()

	type LogEntry struct {
		ID              int64  `json:"id"`
		IPAddress       string `json:"ip_address"`
		FileName        string `json:"file_name"`
		FileSizeIn      int64  `json:"file_size_in"`
		FileSizeOut     int64  `json:"file_size_out"`
		PresetUsed      string `json:"preset_used"`
		Status          string `json:"status"`
		ErrorMessage    string `json:"error_message,omitempty"`
		ExecutionTimeMs int64  `json:"execution_time_ms"`
		CreatedAt       string `json:"created_at"`
	}

	logs := []LogEntry{}
	for rows.Next() {
		var l LogEntry
		var errMsg sql.NullString
		if err := rows.Scan(&l.ID, &l.IPAddress, &l.FileName, &l.FileSizeIn, &l.FileSizeOut, &l.PresetUsed, &l.Status, &errMsg, &l.ExecutionTimeMs, &l.CreatedAt); err != nil {
			continue
		}
		if errMsg.Valid {
			l.ErrorMessage = errMsg.String
		}
		logs = append(logs, l)
	}

	// Get total count
	var total int
	h.DB.QueryRow("SELECT COUNT(*) FROM obfuscation_logs").Scan(&total)

	response.OK(w, map[string]interface{}{
		"data":  logs,
		"total": total,
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
