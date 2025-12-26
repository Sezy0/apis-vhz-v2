package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"vinzhub-rest-api-v2/internal/cache"
	"vinzhub-rest-api-v2/internal/config"
	"vinzhub-rest-api-v2/internal/handler"
	"vinzhub-rest-api-v2/internal/middleware"
	"vinzhub-rest-api-v2/internal/repository"
	"vinzhub-rest-api-v2/internal/router"
	"vinzhub-rest-api-v2/internal/service"

	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting VinzHub API v2...")

	// Load configuration
	cfg := config.MustLoad()
	log.Printf("Environment: %s", cfg.App.Environment)

	// Initialize inventory repository based on config
	var inventoryRepo repository.InventoryRepository
	switch cfg.InventoryDB.Type {
	case "mongodb", "mongo":
		mongoRepo, err := repository.NewMongoDBInventoryRepository(
			cfg.InventoryDB.MongoURI,
			cfg.InventoryDB.MongoDatabase,
			cfg.InventoryDB.MongoCollection,
		)
		if err != nil {
			log.Fatalf("Failed to initialize MongoDB: %v", err)
		}
		defer mongoRepo.Close()
		inventoryRepo = mongoRepo
		log.Println("MongoDB inventory repository initialized")
	case "postgres", "postgresql":
		pgRepo, err := repository.NewPostgresInventoryRepository(cfg.InventoryDB.PostgresDSN())
		if err != nil {
			log.Fatalf("Failed to initialize PostgreSQL: %v", err)
		}
		defer pgRepo.Close()
		inventoryRepo = pgRepo
		log.Println("PostgreSQL inventory repository initialized")
	default: // sqlite
		sqliteRepo, err := repository.NewSQLiteInventoryRepository(cfg.InventoryDB.Path)
		if err != nil {
			log.Fatalf("Failed to initialize SQLite: %v", err)
		}
		defer sqliteRepo.Close()
		inventoryRepo = sqliteRepo
		log.Println("SQLite inventory repository initialized")
	}

	// Initialize MySQL connection for key accounts (optional)
	var err error
	var mysqlDB *sql.DB
	var keyAccountRepo *repository.MySQLKeyAccountRepository

	mysqlDSN := cfg.Database.DSN()
	mysqlDB, err = sql.Open("mysql", mysqlDSN)
	if err != nil {
		log.Printf("Warning: MySQL connection failed: %v", err)
	} else {
		mysqlDB.SetMaxOpenConns(10)
		mysqlDB.SetMaxIdleConns(5)
		mysqlDB.SetConnMaxLifetime(5 * time.Minute)

		if err := mysqlDB.Ping(); err != nil {
			log.Printf("Warning: MySQL ping failed: %v", err)
			mysqlDB.Close()
			mysqlDB = nil
		} else {
			keyAccountRepo = repository.NewMySQLKeyAccountRepository(mysqlDB)
			log.Println("MySQL repository initialized")
		}
	}
	if mysqlDB != nil {
		defer mysqlDB.Close()
	}

	// Initialize Redis client
	redisAddr := cfg.Cache.RedisAddress()
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: cfg.Cache.RedisPassword,
		DB:       cfg.Cache.RedisDB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
		redisClient = nil
	} else {
		log.Println("Redis client initialized")
	}
	cancel()

	// Initialize Redis inventory buffer
	var redisBuffer *cache.RedisInventoryBuffer
	if redisClient != nil {
		bufferCfg := cache.RedisBufferConfig{
			Addr:          redisAddr,
			Password:      cfg.Cache.RedisPassword,
			DB:            cfg.Cache.RedisDB,
			FlushInterval: 30 * time.Second,
		}
		flushFunc := service.CreateFlushFunc(inventoryRepo)
		redisBuffer, err = cache.NewRedisInventoryBuffer(bufferCfg, flushFunc)
		if err != nil {
			log.Printf("Warning: Redis buffer initialization failed: %v", err)
		} else {
			log.Println("Redis inventory buffer initialized")
		}
	}

	// Initialize services
	var inventoryService *service.InventoryService
	if redisBuffer != nil {
		inventoryService = service.NewInventoryServiceWithBuffer(inventoryRepo, keyAccountRepo, redisBuffer)
	} else {
		inventoryService = service.NewInventoryService(inventoryRepo, keyAccountRepo)
	}

	var tokenService *service.TokenService
	if redisClient != nil {
		tokenService = service.NewTokenService(redisClient)
	}

	// Initialize handlers
	healthHandler := handler.New()
	inventoryHandler := handler.NewInventoryHandler(inventoryService)
	adminHandler := handler.NewAdminHandler(redisBuffer, inventoryRepo, cfg.InventoryDB.Type)

	var authHandler *handler.AuthHandler
	if tokenService != nil && keyAccountRepo != nil {
		authHandler = handler.NewAuthHandler(tokenService, keyAccountRepo)
	}

	// Create auth middleware with injected dependencies (NO GLOBALS!)
	authMiddleware := middleware.NewAuthMiddleware(middleware.AuthConfig{
		TokenService: tokenService,
	})

	// Create router
	r := router.New(router.Config{
		Handler:          healthHandler,
		InventoryHandler: inventoryHandler,
		AdminHandler:     adminHandler,
		AuthHandler:      authHandler,
		AuthMiddleware:   authMiddleware,
	})

	// Create HTTP server
	srv := &http.Server{
		Addr:         cfg.Server.Address(),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server listening on %s", cfg.Server.Address())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel = context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	// Close Redis buffer first (flushes pending data)
	if redisBuffer != nil {
		log.Println("Closing Redis buffer...")
		redisBuffer.Close()
	}

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
	fmt.Println("Goodbye!")
}
