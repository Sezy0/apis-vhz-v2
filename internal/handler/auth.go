package handler

import (
	"encoding/json"
	"net/http"

	"vinzhub-rest-api-v2/internal/model"
	"vinzhub-rest-api-v2/internal/repository"
	"vinzhub-rest-api-v2/internal/service"
	"vinzhub-rest-api-v2/pkg/apierror"
	"vinzhub-rest-api-v2/pkg/response"
)

// AuthHandler handles authentication-related HTTP requests.
type AuthHandler struct {
	tokenService   *service.TokenService
	keyAccountRepo repository.KeyAccountRepository // Interface, not concrete type
}

// NewAuthHandler creates a new auth handler.
func NewAuthHandler(tokenService *service.TokenService, keyAccountRepo repository.KeyAccountRepository) *AuthHandler {
	return &AuthHandler{
		tokenService:   tokenService,
		keyAccountRepo: keyAccountRepo,
	}
}

// TokenRequest represents the request body for token generation.
type TokenRequest struct {
	Key      string `json:"key"`
	HWID     string `json:"hwid"`
	RobloxID string `json:"roblox_id"`
}

// TokenResponse represents the response for token generation.
type TokenResponse struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expires_in"`
}

// GenerateToken handles POST /auth/token
func (h *AuthHandler) GenerateToken(w http.ResponseWriter, r *http.Request) {
	var req TokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierror.BadRequest("invalid request body"))
		return
	}
	defer r.Body.Close()

	if req.Key == "" {
		response.Error(w, apierror.BadRequest("key is required"))
		return
	}
	if req.RobloxID == "" {
		response.Error(w, apierror.BadRequest("roblox_id is required"))
		return
	}

	validation, err := h.keyAccountRepo.ValidateKeyAndHWID(r.Context(), req.Key, req.HWID, req.RobloxID)
	if err != nil {
		response.Error(w, apierror.Unauthorized(err.Error()))
		return
	}

	tokenData := model.TokenData{
		KeyAccountID:   validation.KeyAccountID,
		KeyID:          validation.KeyID,
		RobloxUserID:   validation.RobloxUserID,
		RobloxUsername: validation.RobloxUsername,
		HWID:           validation.HWID,
	}

	token, err := h.tokenService.GenerateToken(r.Context(), tokenData)
	if err != nil {
		response.Error(w, apierror.InternalError("failed to generate token"))
		return
	}

	response.OK(w, TokenResponse{
		Token:     token,
		ExpiresIn: 3600,
	})
}

// RevokeToken handles POST /auth/revoke
func (h *AuthHandler) RevokeToken(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Token")
	if token == "" {
		response.Error(w, apierror.BadRequest("X-Token header required"))
		return
	}

	if err := h.tokenService.RevokeToken(r.Context(), token); err != nil {
		response.Error(w, apierror.InternalError("failed to revoke token"))
		return
	}

	response.OK(w, map[string]string{"status": "revoked"})
}

// RefreshToken handles POST /auth/refresh
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Token")
	if token == "" {
		response.Error(w, apierror.BadRequest("X-Token header required"))
		return
	}

	if err := h.tokenService.RefreshToken(r.Context(), token); err != nil {
		response.Error(w, apierror.Unauthorized(err.Error()))
		return
	}

	response.OK(w, map[string]interface{}{
		"status":     "refreshed",
		"expires_in": 3600,
	})
}
