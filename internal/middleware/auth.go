package middleware

import (
	"context"
	"net/http"
	"os"
	"strings"

	"vinzhub-rest-api-v2/internal/model"
	"vinzhub-rest-api-v2/internal/service"
	"vinzhub-rest-api-v2/pkg/apierror"
)

// TokenDataKey is the key for storing token data in request context.
const TokenDataKey contextKey = "token_data"

// AuthConfig holds configuration for the auth middleware.
type AuthConfig struct {
	TokenService *service.TokenService
	APIKeys      []string
}

// NewAuthMiddleware creates an authentication middleware with injected dependencies.
// NO GLOBAL STATE - token service is passed via closure.
func NewAuthMiddleware(cfg AuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for health check endpoints
			if r.URL.Path == "/api/v1/health" || r.URL.Path == "/api/v1/ready" {
				next.ServeHTTP(w, r)
				return
			}

			// Skip auth for admin dashboard and static files
			if r.URL.Path == "/admin" || strings.HasPrefix(r.URL.Path, "/static/") {
				next.ServeHTTP(w, r)
				return
			}

			// Skip auth for docs
			if strings.HasPrefix(r.URL.Path, "/docs") {
				next.ServeHTTP(w, r)
				return
			}

			// Skip auth for token generation endpoint
			if r.URL.Path == "/api/v1/auth/token" && r.Method == "POST" {
				next.ServeHTTP(w, r)
				return
			}
			
			// Allow admin endpoints with X-Login-Key (handled by AdminHandler)
			if strings.HasPrefix(r.URL.Path, "/api/v1/admin") {
				// /admin/login and /admin/stats with X-Login-Key bypass regular auth
				loginKey := r.Header.Get("X-Login-Key")
				if loginKey != "" || r.URL.Path == "/api/v1/admin/login" {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Try X-Token first (session tokens)
			token := r.Header.Get("X-Token")
			if token != "" && cfg.TokenService != nil {
				tokenData, err := cfg.TokenService.ValidateToken(r.Context(), token)
				if err != nil {
					writeError(w, apierror.Unauthorized("Invalid or expired token"))
					return
				}

				ctx := context.WithValue(r.Context(), TokenDataKey, tokenData)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Fall back to X-API-Key
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				auth := r.Header.Get("Authorization")
				if strings.HasPrefix(auth, "Bearer ") {
					apiKey = strings.TrimPrefix(auth, "Bearer ")
				}
			}

			if apiKey == "" {
				writeError(w, apierror.Unauthorized("Authentication required. Use X-Token or X-API-Key header."))
				return
			}

			// Validate API key
			validKeys := cfg.APIKeys
			if len(validKeys) == 0 {
				validKeys = getAPIKeysFromEnv()
			}

			if !isValidKey(apiKey, validKeys) {
				writeError(w, apierror.Unauthorized("Invalid API key"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// writeError writes an API error response.
func writeError(w http.ResponseWriter, err *apierror.Error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.StatusCode)
	w.Write(err.ToJSON())
}

// getAPIKeysFromEnv returns API keys from environment variables.
func getAPIKeysFromEnv() []string {
	keysEnv := os.Getenv("API_KEYS")
	if keysEnv == "" {
		singleKey := os.Getenv("API_KEY")
		if singleKey != "" {
			return []string{singleKey}
		}
		return nil
	}

	keys := strings.Split(keysEnv, ",")
	for i := range keys {
		keys[i] = strings.TrimSpace(keys[i])
	}
	return keys
}

// isValidKey checks if the provided key is in the valid keys list.
func isValidKey(key string, validKeys []string) bool {
	for _, valid := range validKeys {
		if key == valid {
			return true
		}
	}
	return false
}

// GetTokenDataFromContext retrieves token data from request context.
func GetTokenDataFromContext(ctx context.Context) *model.TokenData {
	if data, ok := ctx.Value(TokenDataKey).(*model.TokenData); ok {
		return data
	}
	return nil
}
