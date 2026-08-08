package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the application
type Config struct {
	Server      ServerConfig
	Database    DatabaseConfig
	Security    SecurityConfig
	Logging     LoggingConfig
	External    ExternalConfig
	RPC         RPCConfig
	Auth        AuthConfig
	Redis       RedisConfig
	Messaging   MessagingConfig
	ObjectStore ObjectStoreConfig
}

// ObjectStoreConfig contains connection info for the local MinIO
// (S3-API-compatible) object store used to persist generated exports.
type ObjectStoreConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	UseSSL    bool
	Enabled   bool
}

// MessagingConfig contains connection info for the three message brokers.
// Each is independently enabled so a partial local stack (e.g. no
// RocketMQ running) doesn't block the rest of the system.
type MessagingConfig struct {
	KafkaBrokers    []string
	KafkaEnabled    bool
	RabbitMQURL     string
	RabbitMQEnabled bool
	RocketMQNsAddrs []string
	RocketMQEnabled bool
}

// RPCConfig contains addresses for the Kitex RPC microservices
type RPCConfig struct {
	PostServiceAddr string // host:port the postsvc Kitex server listens on / gateway dials
	AuthServiceAddr string // host:port the authsvc Kitex server listens on / gateway dials
	MuxTransport    bool   // enable Kitex connection multiplexing
}

// AuthConfig contains ABAC / JWT auth configuration
type AuthConfig struct {
	JWTSecret     string
	TokenTTL      time.Duration
	AdminUsername string
	AdminPassword string
}

// RedisConfig contains Redis cache configuration
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
	Enabled  bool
}

// ServerConfig contains server-related configuration
type ServerConfig struct {
	Port            string
	Host            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	Environment     string
}

// DatabaseConfig contains database-related configuration
type DatabaseConfig struct {
	Type     string // "file" or "postgres"
	FilePath string
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
	MaxConns int
	MinConns int
}

// SecurityConfig contains security-related configuration
type SecurityConfig struct {
	RateLimitRequests int
	RateLimitWindow   time.Duration
	MaxBodySize       int64
	AllowedOrigins    []string
	TrustedProxies    []string
}

// LoggingConfig contains logging-related configuration
type LoggingConfig struct {
	Level      string
	Format     string // "json" or "text"
	Output     string // "stdout" or file path
	TimeFormat string
}

// ExternalConfig contains external service configuration
type ExternalConfig struct {
	JSONPlaceholderURL string
	HTTPTimeout        time.Duration
}

// Load reads configuration from environment variables with sensible defaults
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:            getEnv("PORT", "8080"),
			Host:            getEnv("HOST", "0.0.0.0"),
			ReadTimeout:     getDurationEnv("READ_TIMEOUT", 15*time.Second),
			WriteTimeout:    getDurationEnv("WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:     getDurationEnv("IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: getDurationEnv("SHUTDOWN_TIMEOUT", 30*time.Second),
			Environment:     getEnv("ENVIRONMENT", "development"),
		},
		Database: DatabaseConfig{
			Type:     getEnv("DB_TYPE", "file"),
			FilePath: getEnv("DB_FILE_PATH", "posts.json"),
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			DBName:   getEnv("DB_NAME", "postanalyzer"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
			MaxConns: getIntEnv("DB_MAX_CONNS", 25),
			MinConns: getIntEnv("DB_MIN_CONNS", 5),
		},
		Security: SecurityConfig{
			RateLimitRequests: getIntEnv("RATE_LIMIT_REQUESTS", 100),
			RateLimitWindow:   getDurationEnv("RATE_LIMIT_WINDOW", 1*time.Minute),
			MaxBodySize:       getInt64Env("MAX_BODY_SIZE", 1*1024*1024), // 1MB
			AllowedOrigins:    getSliceEnv("ALLOWED_ORIGINS", []string{"*"}),
			TrustedProxies:    getSliceEnv("TRUSTED_PROXIES", []string{}),
		},
		Logging: LoggingConfig{
			Level:      getEnv("LOG_LEVEL", "info"),
			Format:     getEnv("LOG_FORMAT", "json"),
			Output:     getEnv("LOG_OUTPUT", "stdout"),
			TimeFormat: getEnv("LOG_TIME_FORMAT", time.RFC3339),
		},
		External: ExternalConfig{
			JSONPlaceholderURL: getEnv("JSONPLACEHOLDER_URL", "https://jsonplaceholder.typicode.com/posts"),
			HTTPTimeout:        getDurationEnv("HTTP_TIMEOUT", 30*time.Second),
		},
		RPC: RPCConfig{
			PostServiceAddr: getEnv("POSTSVC_ADDR", "127.0.0.1:9001"),
			AuthServiceAddr: getEnv("AUTHSVC_ADDR", "127.0.0.1:9002"),
			MuxTransport:    getEnv("RPC_MUX", "true") == "true",
		},
		Auth: AuthConfig{
			JWTSecret:     getEnv("JWT_SECRET", "dev-only-change-me-in-production"),
			TokenTTL:      getDurationEnv("JWT_TTL", 24*time.Hour),
			AdminUsername: getEnv("ADMIN_USERNAME", "admin"),
			AdminPassword: getEnv("ADMIN_PASSWORD", "admin123"),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "127.0.0.1:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getIntEnv("REDIS_DB", 0),
			Enabled:  getEnv("REDIS_ENABLED", "true") == "true",
		},
		Messaging: MessagingConfig{
			KafkaBrokers:    getSliceEnv("KAFKA_BROKERS", []string{"127.0.0.1:9092"}),
			KafkaEnabled:    getEnv("KAFKA_ENABLED", "false") == "true",
			RabbitMQURL:     getEnv("RABBITMQ_URL", "amqp://guest:guest@127.0.0.1:5672/"),
			RabbitMQEnabled: getEnv("RABBITMQ_ENABLED", "false") == "true",
			RocketMQNsAddrs: getSliceEnv("ROCKETMQ_NAMESRV_ADDRS", []string{"127.0.0.1:9876"}),
			RocketMQEnabled: getEnv("ROCKETMQ_ENABLED", "false") == "true",
		},
		ObjectStore: ObjectStoreConfig{
			Endpoint:  getEnv("MINIO_ENDPOINT", "127.0.0.1:9000"),
			AccessKey: getEnv("MINIO_ACCESS_KEY", "minioadmin"),
			SecretKey: getEnv("MINIO_SECRET_KEY", "minioadmin"),
			UseSSL:    getEnv("MINIO_USE_SSL", "false") == "true",
			Enabled:   getEnv("MINIO_ENABLED", "false") == "true",
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate server config
	if c.Server.Port == "" {
		return fmt.Errorf("server port cannot be empty")
	}
	if c.Server.Environment != "development" && c.Server.Environment != "staging" && c.Server.Environment != "production" {
		return fmt.Errorf("environment must be one of: development, staging, production")
	}

	// Validate database config
	if c.Database.Type != "file" && c.Database.Type != "postgres" {
		return fmt.Errorf("database type must be 'file' or 'postgres'")
	}
	if c.Database.Type == "file" && c.Database.FilePath == "" {
		return fmt.Errorf("database file path cannot be empty when using file storage")
	}
	if c.Database.Type == "postgres" {
		if c.Database.Host == "" || c.Database.DBName == "" {
			return fmt.Errorf("database host and name are required for postgres")
		}
	}

	// Validate security config
	if c.Security.RateLimitRequests <= 0 {
		return fmt.Errorf("rate limit requests must be positive")
	}
	if c.Security.MaxBodySize <= 0 {
		return fmt.Errorf("max body size must be positive")
	}

	// Validate logging config
	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLogLevels[c.Logging.Level] {
		return fmt.Errorf("log level must be one of: debug, info, warn, error")
	}
	if c.Logging.Format != "json" && c.Logging.Format != "text" {
		return fmt.Errorf("log format must be 'json' or 'text'")
	}

	return nil
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Server.Environment == "development"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.Server.Environment == "production"
}

// Helper functions for reading environment variables

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getInt64Env(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getSliceEnv(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// Simple comma-separated parsing
		result := []string{}
		current := ""
		for _, char := range value {
			if char == ',' {
				if current != "" {
					result = append(result, current)
					current = ""
				}
			} else {
				current += string(char)
			}
		}
		if current != "" {
			result = append(result, current)
		}
		return result
	}
	return defaultValue
}
