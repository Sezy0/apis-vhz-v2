package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"vinzhub-rest-api-v2/internal/model"
	"vinzhub-rest-api-v2/internal/repository"
	"vinzhub-rest-api-v2/pkg/apierror"
	"vinzhub-rest-api-v2/pkg/response"
)

// ObfuscationHandler handles Lua script obfuscation
type ObfuscationHandler struct {
	FoxzyPath       string // Path to Foxzy-Obfuscator directory
	FileUploaderURL string // URL of file-uploader service
	LogRepo         repository.LogRepository
}

// NewObfuscationHandler creates a new obfuscation handler
func NewObfuscationHandler(foxzyPath, fileUploaderURL string, logRepo repository.LogRepository) *ObfuscationHandler {
	return &ObfuscationHandler{
		FoxzyPath:       foxzyPath,
		FileUploaderURL: fileUploaderURL,
		LogRepo:         logRepo,
	}
}

// ... existing code ...

// insertLog records the obfuscation attempt asynchronously
func (h *ObfuscationHandler) insertLog(req ObfuscateRequest, ip string, sizeIn, sizeOut int64, status, errorMsg string, durationMs int64) {
	if h.LogRepo == nil {
		return
	}

	go func() {
		preset := req.Preset
		if req.CustomConfig != nil {
			preset = "Custom"
		}

		logEntry := &model.ObfuscationLog{
			IPAddress:       ip,
			FileName:        req.Filename,
			FileSizeIn:      sizeIn,
			FileSizeOut:     sizeOut,
			PresetUsed:      preset,
			Status:          status,
			ErrorMessage:    errorMsg,
			ExecutionTimeMs: durationMs,
			CreatedAt:       time.Now(),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := h.LogRepo.InsertObfuscationLog(ctx, logEntry); err != nil {
			fmt.Printf("Failed to insert obfuscation log: %v\n", err)
		}
	}()
}

// CustomConfig defines custom obfuscation settings
type CustomConfig struct {
	NameGenerator string   `json:"NameGenerator"`
	Steps         []string `json:"Steps"`
}

// ObfuscateRequest is the request body for obfuscation
type ObfuscateRequest struct {
	Content      string        `json:"content"`
	Preset       string        `json:"preset"` // FoxzyLight, FoxzyBalanced, FoxzyMax
	Filename     string        `json:"filename"` // Original filename
	CustomConfig *CustomConfig `json:"customConfig,omitempty"`
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

// generateCustomConfig creates a temporary Lua config file
func (h *ObfuscationHandler) generateCustomConfig(config *CustomConfig) (string, error) {
	tmpDir := os.TempDir()
	configFile := filepath.Join(tmpDir, fmt.Sprintf("foxzy_config_%d.lua", time.Now().UnixNano()))

	var stepsLua string
	for _, step := range config.Steps {
		stepsLua += fmt.Sprintf(`{ Name = "%s", Settings = {} },`, step)
	}

	luaConfig := fmt.Sprintf(`
return {
    LuaVersion = "Lua51",
    VarNamePrefix = "",
    NameGenerator = "%s",
    PrettyPrint = false,
    Seed = 0,
    Steps = {
        %s
    }
}
`, config.NameGenerator, stepsLua)

	if err := os.WriteFile(configFile, []byte(luaConfig), 0644); err != nil {
		return "", err
	}

	return configFile, nil
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

	var cmd *exec.Cmd
	var configFile string

	if req.CustomConfig != nil {
		// Custom Configuration Mode
		var err error
		configFile, err = h.generateCustomConfig(req.CustomConfig)
		if err != nil {
			response.Error(w, apierror.InternalError("Failed to generate custom config"))
			return
		}
		defer os.Remove(configFile)

		cmd = exec.Command("lua", "cli.lua", "--config", configFile, "--Lua51", "--out", outputFile, inputFile)
	} else {
		// Preset Mode
		// Validate preset - available Foxzy presets
		validPresets := map[string]bool{
			"Minify":        true,
			"FoxzyLight":    true,
			"FoxzyBalanced": true,
			"FoxzyMax":      true,
			"FoxzyMaxCF":    true,
		}
		
		if req.Preset == "" {
			req.Preset = "FoxzyBalanced"
		}
		
		if !validPresets[req.Preset] {
			response.Error(w, apierror.BadRequest("Invalid preset"))
			return
		}

		cmd = exec.Command("lua", "cli.lua", "--preset", req.Preset, "--Lua51", "--out", outputFile, inputFile)
	}
	
	// Run Foxzy
	cmd.Dir = h.FoxzyPath
	
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	
	if err := cmd.Run(); err != nil {
		ip := r.Header.Get("X-Forwarded-For")
		if ip == "" { ip = r.RemoteAddr }
		h.insertLog(req, ip, int64(len(req.Content)), 0, "failed", stderr.String(), time.Since(startTime).Milliseconds())

		response.Error(w, apierror.InternalError(fmt.Sprintf("Obfuscation failed: %s", stderr.String())))
		return
	}
		// Read output
	obfuscated, err := os.ReadFile(outputFile)
	if err != nil {
		ip := r.Header.Get("X-Forwarded-For")
		if ip == "" { ip = r.RemoteAddr }
		h.insertLog(req, ip, int64(len(req.Content)), 0, "failed", "Read output failed", time.Since(startTime).Milliseconds())

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
	
	// Log success
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" { ip = r.RemoteAddr }
	h.insertLog(req, ip, int64(len(req.Content)), int64(len(obfuscated)), "success", "", processTime)

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


