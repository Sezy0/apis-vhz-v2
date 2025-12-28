package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"vinzhub-rest-api-v2/pkg/apierror"
	"vinzhub-rest-api-v2/pkg/response"
)

// ObfuscationHandler handles Lua script obfuscation
type ObfuscationHandler struct {
	FoxzyPath       string // Path to Foxzy-Obfuscator directory
	FileUploaderURL string // URL of file-uploader service
}

// NewObfuscationHandler creates a new obfuscation handler
func NewObfuscationHandler(foxzyPath, fileUploaderURL string) *ObfuscationHandler {
	return &ObfuscationHandler{
		FoxzyPath:       foxzyPath,
		FileUploaderURL: fileUploaderURL,
	}
}

// ObfuscateRequest is the request body for obfuscation
type ObfuscateRequest struct {
	Content  string `json:"content"`
	Preset   string `json:"preset"`   // FoxzyLight, FoxzyBalanced, FoxzyMax
	Filename string `json:"filename"` // Original filename
}

// ObfuscateResponse is the response for obfuscation
type ObfuscateResponse struct {
	Success     bool   `json:"success"`
	Slug        string `json:"slug,omitempty"`
	ResultURL   string `json:"result_url,omitempty"`
	Content     string `json:"content,omitempty"`
	Error       string `json:"error,omitempty"`
	ProcessTime int64  `json:"process_time_ms,omitempty"`
}

// Obfuscate handles POST /api/v1/obfuscate
func (h *ObfuscationHandler) Obfuscate(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Parse request
	var req ObfuscateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierror.BadRequest("Failed to parse request body"))
		return
	}
	
	if req.Content == "" {
		response.Error(w, apierror.BadRequest("Content is required"))
		return
	}
	
	// Validate preset
	validPresets := map[string]bool{
		"Minify": true,
		"FoxzyLight": true,
		"FoxzyBalanced": true,
		"FoxzyMax": true,
		"FoxzyMaxCF": true,
	}
	
	if req.Preset == "" {
		req.Preset = "FoxzyBalanced"
	}
	
	if !validPresets[req.Preset] {
		response.Error(w, apierror.BadRequest("Invalid preset. Use: Minify, FoxzyLight, FoxzyBalanced, FoxzyMax, FoxzyMaxCF"))
		return
	}
	
	// Create temp files
	tmpDir := os.TempDir()
	inputFile := filepath.Join(tmpDir, fmt.Sprintf("foxzy_input_%d.lua", time.Now().UnixNano()))
	outputFile := filepath.Join(tmpDir, fmt.Sprintf("foxzy_output_%d.lua", time.Now().UnixNano()))
	
	// Write input
	if err := os.WriteFile(inputFile, []byte(req.Content), 0644); err != nil {
		response.Error(w, apierror.InternalError("Failed to create temp file"))
		return
	}
	defer os.Remove(inputFile)
	defer os.Remove(outputFile)
	
	// Run Foxzy
	cmd := exec.Command("lua", "cli.lua", "--preset", req.Preset, "--Lua51", "--out", outputFile, inputFile)
	cmd.Dir = h.FoxzyPath
	
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	
	if err := cmd.Run(); err != nil {
		response.Error(w, apierror.InternalError(fmt.Sprintf("Obfuscation failed: %s", stderr.String())))
		return
	}
	
	// Read output
	obfuscated, err := os.ReadFile(outputFile)
	if err != nil {
		response.Error(w, apierror.InternalError("Failed to read obfuscated output"))
		return
	}

	processTime := time.Since(startTime).Milliseconds()
	
	// Upload to file-uploader if URL is configured
	var slug string
	var resultURL string
	if h.FileUploaderURL != "" {
		slug, resultURL, err = h.uploadToFileUploader(string(obfuscated), req.Filename)
		if err != nil {
			// Log error but don't fail - return content directly
			fmt.Printf("Failed to upload to file-uploader: %v\n", err)
		}
	}
	
	resp := ObfuscateResponse{
		Success:     true,
		Slug:        slug,
		ResultURL:   resultURL,
		Content:     string(obfuscated),
		ProcessTime: processTime,
	}
	
	response.OK(w, resp)
}

// uploadToFileUploader uploads obfuscated content to file-uploader service
func (h *ObfuscationHandler) uploadToFileUploader(content string, filename string) (string, string, error) {
	payload := map[string]string{
		"content":  content,
		"type":     "obfuscated",
		"filename": filename,
	}
	
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", "", err
	}
	
	resp, err := http.Post(h.FileUploaderURL+"/api/obs/upload", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	
	var result struct {
		Success bool   `json:"success"`
		Slug    string `json:"slug"`
		URL     string `json:"url"`
	}
	
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", err
	}
	
	if !result.Success {
		return "", "", fmt.Errorf("upload failed")
	}
	
	return result.Slug, result.URL, nil
}
