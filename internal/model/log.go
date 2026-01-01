package model

import "time"

// ObfuscationLog represents a record in the obfuscation_logs table
type ObfuscationLog struct {
	ID              int64     `json:"id"`
	UserID          *int64    `json:"user_id,omitempty"`
	IPAddress       string    `json:"ip_address"`
	FileName        string    `json:"file_name"`
	FileSizeIn      int64     `json:"file_size_in"`
	FileSizeOut     int64     `json:"file_size_out"`
	PresetUsed      string    `json:"preset_used"`
	Status          string    `json:"status"` // 'success' or 'failed'
	ErrorMessage    string    `json:"error_message,omitempty"`
	ExecutionTimeMs int64     `json:"execution_time_ms"`
	CreatedAt       time.Time `json:"created_at"`
}
