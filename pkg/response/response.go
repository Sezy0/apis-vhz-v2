package response

import (
	"encoding/json"
	"net/http"

	"vinzhub-rest-api-v2/pkg/apierror"
)

// Response represents a standard API response.
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// Meta contains pagination metadata.
type Meta struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
}

// JSON sends a JSON response with the given status code.
func JSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := Response{
		Success: true,
		Data:    data,
	}

	_ = json.NewEncoder(w).Encode(response)
}

// JSONWithMeta sends a JSON response with pagination metadata.
func JSONWithMeta(w http.ResponseWriter, statusCode int, data interface{}, page, limit int, total int64) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := Response{
		Success: true,
		Data:    data,
		Meta: &Meta{
			Page:  page,
			Limit: limit,
			Total: total,
		},
	}

	_ = json.NewEncoder(w).Encode(response)
}

// Error sends an error response.
func Error(w http.ResponseWriter, err error) {
	// Check if it's an APIError
	if apiErr, ok := err.(*apierror.Error); ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(apiErr.StatusCode)
		w.Write(apiErr.ToJSON())
		return
	}

	// Default to internal server error
	internalErr := apierror.InternalError("an unexpected error occurred")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(internalErr.StatusCode)
	w.Write(internalErr.ToJSON())
}

// NoContent sends a 204 No Content response.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// Created sends a 201 Created response with the created resource.
func Created(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusCreated, data)
}

// OK sends a 200 OK response.
func OK(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, data)
}
