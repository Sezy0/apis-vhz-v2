package router

import (
	"net/http"

	"vinzhub-rest-api-v2/internal/handler"
	"vinzhub-rest-api-v2/internal/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

// Config holds the configuration for creating a router.
type Config struct {
	Handler             *handler.Handler
	InventoryHandler    *handler.InventoryHandler
	AdminHandler        *handler.AdminHandler
	AuthHandler         *handler.AuthHandler
	ObfuscationHandler  *handler.ObfuscationHandler
	AuthMiddleware      func(http.Handler) http.Handler
}

// New creates and configures the HTTP router.
func New(cfg Config) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware stack (applies to ALL routes)
	r.Use(middleware.Recovery)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logging)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID", "X-API-Key", "X-Token", "X-Login-Key"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// PUBLIC routes (no auth required)
	if cfg.Handler != nil {
		r.Get("/api/status", cfg.Handler.Status)
	}

	// Static files (admin dashboard) - public
	fileServer := http.FileServer(http.Dir("./static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	// Admin dashboard redirect - public
	r.Get("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/static/admin.html", http.StatusMovedPermanently)
	})

	// AUTHENTICATED routes (use Group to apply auth middleware only to these)
	r.Group(func(r chi.Router) {
		// Apply auth middleware only to this group
		if cfg.AuthMiddleware != nil {
			r.Use(cfg.AuthMiddleware)
		}

		// API v1 routes
		r.Route("/api/v1", func(r chi.Router) {
			// Health check endpoints
			if cfg.Handler != nil {
				r.Get("/health", cfg.Handler.Health)
				r.Get("/ready", cfg.Handler.Ready)
			}

			// Auth endpoints
			if cfg.AuthHandler != nil {
				r.Route("/auth", func(r chi.Router) {
					r.Post("/token", cfg.AuthHandler.GenerateToken)
					r.Post("/revoke", cfg.AuthHandler.RevokeToken)
					r.Post("/refresh", cfg.AuthHandler.RefreshToken)
				})
			}

			// Inventory endpoints
			if cfg.InventoryHandler != nil {
				r.Route("/inventory/{roblox_user_id}", func(r chi.Router) {
					r.Post("/sync", cfg.InventoryHandler.SyncRawInventory)
					r.Get("/", cfg.InventoryHandler.GetRawInventory)
				})
			}

			// Admin endpoints
			if cfg.AdminHandler != nil {
				r.Route("/admin", func(r chi.Router) {
					r.Get("/stats", cfg.AdminHandler.GetStats)
					r.Get("/health", cfg.AdminHandler.GetHealth)
					r.Post("/login", cfg.AdminHandler.VerifyLogin)
				})
			}

			// Obfuscation endpoint
			if cfg.ObfuscationHandler != nil {
				r.Post("/obfuscate", cfg.ObfuscationHandler.Obfuscate)
			}
		})
	})

	return r
}
