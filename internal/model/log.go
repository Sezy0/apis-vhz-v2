package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ObfuscationLog represents a record in the obfuscation_logs collection
type ObfuscationLog struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID          *int64             `bson:"user_id,omitempty" json:"user_id,omitempty"`
	IPAddress       string             `bson:"ip_address" json:"ip_address"`
	FileName        string             `bson:"file_name" json:"file_name"`
	FileSizeIn      int64              `bson:"file_size_in" json:"file_size_in"`
	FileSizeOut     int64              `bson:"file_size_out" json:"file_size_out"`
	PresetUsed      string             `bson:"preset_used" json:"preset_used"`
	Status          string             `bson:"status" json:"status"` // 'success' or 'failed'
	ErrorMessage    string             `bson:"error_message,omitempty" json:"error_message,omitempty"`
	ExecutionTimeMs int64              `bson:"execution_time_ms" json:"execution_time_ms"`
	CreatedAt       time.Time          `bson:"created_at" json:"created_at"`
}
