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

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
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
	Redis           *redis.Client
}

// NewObfuscationHandler creates a new obfuscation handler
func NewObfuscationHandler(foxzyPath, fileUploaderURL string, logRepo repository.LogRepository, redisClient *redis.Client) *ObfuscationHandler {
	return &ObfuscationHandler{
		FoxzyPath:       foxzyPath,
		FileUploaderURL: fileUploaderURL,
		LogRepo:         logRepo,
		Redis:           redisClient,
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

	// Generate Job ID
	jobID := fmt.Sprintf("job_%d_%s", time.Now().UnixNano(), "foxzy") // Simple ID
	// Verify Redis connection
	if h.Redis == nil {
		response.Error(w, apierror.InternalError("Redis unavailable for async processing"))
		return
	}

	// Set initial status
	initialStatus := map[string]interface{}{
		"status": "processing",
		"ts":     time.Now().Unix(),
	}
	statusJSON, _ := json.Marshal(initialStatus)
	h.Redis.Set(r.Context(), "obs_job:"+jobID, statusJSON, 15*time.Minute)

	// Launch background process
	go func() {
		// Create Background Context for Redis operations inside goroutine
		bgCtx := context.Background()

		// Create temp files
		tmpDir := os.TempDir()
		inputFile := filepath.Join(tmpDir, fmt.Sprintf("foxzy_input_%s.lua", jobID))
		outputFile := filepath.Join(tmpDir, fmt.Sprintf("foxzy_output_%s.lua", jobID))
		
		defer os.Remove(inputFile)
		defer os.Remove(outputFile)

		// Write input
		if err := os.WriteFile(inputFile, []byte(req.Content), 0644); err != nil {
			h.updateJobStatus(bgCtx, jobID, "failed", "", "", fmt.Sprintf("Write error: %v", err))
			return
		}

		var cmd *exec.Cmd
		var configFile string

		if req.CustomConfig != nil {
			// Custom Configuration Mode
			var err error
			configFile, err = h.generateCustomConfig(req.CustomConfig)
			if err != nil {
				h.updateJobStatus(bgCtx, jobID, "failed", "", "", fmt.Sprintf("Config error: %v", err))
				return
			}
			defer os.Remove(configFile)

			cmd = exec.Command("lua", "cli.lua", "--config", configFile, "--Lua51", "--out", outputFile, inputFile)
		} else {
			// Preset Mode
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
				h.updateJobStatus(bgCtx, jobID, "failed", "", "", "Invalid preset")
				return
			}

			cmd = exec.Command("lua", "cli.lua", "--preset", req.Preset, "--Lua51", "--out", outputFile, inputFile)
		}
		

		// Run Foxzy
		cmd.Dir = h.FoxzyPath
		var stderr bytes.Buffer
		var stdout bytes.Buffer
		cmd.Stderr = &stderr
		cmd.Stdout = &stdout
		
		fmt.Printf("Running Foxzy: %v \nInput: %s\nOutput: %s\n", cmd.Args, inputFile, outputFile)

		if err := cmd.Run(); err != nil {
			ip := r.Header.Get("X-Forwarded-For")
			if ip == "" { ip = r.RemoteAddr }
			h.insertLog(req, ip, int64(len(req.Content)), 0, "failed", stderr.String(), time.Since(startTime).Milliseconds())

			// Log detailed error
			fmt.Printf("Foxzy Failed: %v\nStderr: %s\nStdout: %s\n", err, stderr.String(), stdout.String())
			
			h.updateJobStatus(bgCtx, jobID, "failed", "", "", fmt.Sprintf("Obfuscation error: %s", stderr.String()))
			return
		}

		// Read output
		obfuscated, err := os.ReadFile(outputFile)
		if err != nil {
			h.updateJobStatus(bgCtx, jobID, "failed", "", "", "Failed to read output file")
			return
		}

		// Check if output is empty
		if len(obfuscated) == 0 {
			fmt.Printf("Foxzy output file is empty!\nStderr: %s\nStdout: %s\n", stderr.String(), stdout.String())
			h.updateJobStatus(bgCtx, jobID, "failed", "", "", "Obfuscation generated empty file")
			return
		}

		processTime := time.Since(startTime).Milliseconds()
		
		// Upload to file-uploader if URL is configured
		var resultURL string
		if h.FileUploaderURL != "" {
			var errUpload error
			_, resultURL, errUpload = h.uploadToFileUploader(string(obfuscated), req.Filename)
			if errUpload != nil {
				fmt.Printf("File upload failed: %v\n", errUpload)
			}
		}
		
		// Log success
		ip := r.Header.Get("X-Forwarded-For")
		if ip == "" { ip = r.RemoteAddr }
		h.insertLog(req, ip, int64(len(req.Content)), int64(len(obfuscated)), "success", "", processTime)

		// Update Job to Success
		h.updateJobStatus(bgCtx, jobID, "success", string(obfuscated), resultURL, "")
	}()

	response.OK(w, map[string]string{
		"message": "Obfuscation started",
		"job_id":  jobID,
		"status":  "processing",
	})
}

// updateJobStatus helper to update Redis
func (h *ObfuscationHandler) updateJobStatus(ctx context.Context, jobID, status, content, resultURL, errorMsg string) {
	data := map[string]interface{}{
		"status":     status,
		"content":    content,
		"result_url": resultURL,
		"error":      errorMsg,
		"ts":         time.Now().Unix(),
	}
	jsonBytes, _ := json.Marshal(data)
	h.Redis.Set(ctx, "obs_job:"+jobID, jsonBytes, 15*time.Minute)
}

// GetObfuscationStatus handles GET /api/v1/obfuscate/status/{jobID}
func (h *ObfuscationHandler) GetObfuscationStatus(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobID")
	
	val, err := h.Redis.Get(r.Context(), "obs_job:"+jobID).Result()
	if err == redis.Nil {
		response.Error(w, apierror.NotFound("Job not found"))
		return
	} else if err != nil {
		response.Error(w, apierror.InternalError("Redis error"))
		return
	}

	var data map[string]interface{}
	json.Unmarshal([]byte(val), &data)
	
	response.OK(w, data)
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


