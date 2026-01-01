package config

import (
	"fmt"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

func init() {
	// Load .env file if it exists (silent fail if not)
	_ = godotenv.Load()
}

// Config holds all application configuration loaded from environment variables.
type Config struct {
	Server      ServerConfig
	App         AppConfig
	Cache       CacheConfig
	Database    DatabaseConfig
	InventoryDB InventoryDBConfig
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host            string        `envconfig:"SERVER_HOST" default:"0.0.0.0"`
	Port            int           `envconfig:"SERVER_PORT" default:"8080"`
	ReadTimeout     time.Duration `envconfig:"SERVER_READ_TIMEOUT" default:"15s"`
	WriteTimeout    time.Duration `envconfig:"SERVER_WRITE_TIMEOUT" default:"3000s"`
	ShutdownTimeout time.Duration `envconfig:"SERVER_SHUTDOWN_TIMEOUT" default:"30s"`
}

// AppConfig holds application-level settings.
type AppConfig struct {
	Name        string `envconfig:"APP_NAME" default:"vinzhub-api"`
	Environment string `envconfig:"APP_ENV" default:"development"`
	Debug       bool   `envconfig:"APP_DEBUG" default:"false"`
	Version     string `envconfig:"APP_VERSION" default:"2.0.0"`
	LoginKey    string `envconfig:"LOGIN_KEY" default:""`  // Admin dashboard login key
}

// CacheConfig holds cache settings.
type CacheConfig struct {
	Type string        `envconfig:"CACHE_TYPE" default:"memory"`
	TTL  time.Duration `envconfig:"CACHE_TTL" default:"5m"`

	RedisHost     string `envconfig:"REDIS_HOST" default:"localhost"`
	RedisPort     int    `envconfig:"REDIS_PORT" default:"6379"`
	RedisPassword string `envconfig:"REDIS_PASSWORD" default:""`
	RedisDB       int    `envconfig:"REDIS_DB" default:"0"`
}

// DatabaseConfig holds MySQL connection settings (for key_accounts).
type DatabaseConfig struct {
	Host     string `envconfig:"DB_HOST" default:"localhost"`
	Port     int    `envconfig:"DB_PORT" default:"3306"`
	Name     string `envconfig:"DB_NAME" default:"vinzhub"`
	User     string `envconfig:"DB_USER" default:"root"`
	Password string `envconfig:"DB_PASS" default:""`
}

// InventoryDBConfig holds inventory database settings.
type InventoryDBConfig struct {
	Type     string `envconfig:"INVENTORY_DB_TYPE" default:"sqlite"` // sqlite, postgres, or mongodb
	Path     string `envconfig:"INVENTORY_DB_PATH" default:"./data/inventory.db"`
	// PostgreSQL settings
	Host     string `envconfig:"INVENTORY_DB_HOST" default:"localhost"`
	Port     int    `envconfig:"INVENTORY_DB_PORT" default:"5432"`
	Name     string `envconfig:"INVENTORY_DB_NAME" default:"vinzhub"`
	User     string `envconfig:"INVENTORY_DB_USER" default:"postgres"`
	Password string `envconfig:"INVENTORY_DB_PASS" default:""`
	SSLMode  string `envconfig:"INVENTORY_DB_SSLMODE" default:"disable"`
	// MongoDB settings
	MongoURI        string `envconfig:"MONGODB_URI" default:""`
	MongoDatabase   string `envconfig:"MONGODB_DATABASE" default:"vinzhub"`
	MongoCollection string `envconfig:"MONGODB_COLLECTION" default:"fishit_inventory"`
}

// PostgresDSN returns the PostgreSQL connection string.
func (i *InventoryDBConfig) PostgresDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		i.User, i.Password, i.Host, i.Port, i.Name, i.SSLMode)
}

// Address returns the server address in host:port format.
func (s *ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// RedisAddress returns the Redis address in host:port format.
func (c *CacheConfig) RedisAddress() string {
	return fmt.Sprintf("%s:%d", c.RedisHost, c.RedisPort)
}

// DSN returns the MySQL data source name.
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		d.User, d.Password, d.Host, d.Port, d.Name)
}

// IsDevelopment returns true if running in development mode.
func (a *AppConfig) IsDevelopment() bool {
	return a.Environment == "development"
}

// IsProduction returns true if running in production mode.
func (a *AppConfig) IsProduction() bool {
	return a.Environment == "production"
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	var cfg Config

	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return &cfg, nil
}

// MustLoad loads configuration or panics on error.
func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		panic(err)
	}
	return cfg
}
