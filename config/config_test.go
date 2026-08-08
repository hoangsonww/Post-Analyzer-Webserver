package config

import (
	"testing"
	"time"
)

func TestLoad_Defaults(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Server.Port != "8080" {
		t.Errorf("expected default port 8080, got %s", cfg.Server.Port)
	}
	if cfg.Database.Type != "file" {
		t.Errorf("expected default db type file, got %s", cfg.Database.Type)
	}
	if cfg.Logging.Level != "info" {
		t.Errorf("expected default log level info, got %s", cfg.Logging.Level)
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("DB_TYPE", "postgres")
	t.Setenv("DB_HOST", "dbhost")
	t.Setenv("DB_NAME", "dbname")
	t.Setenv("ALLOWED_ORIGINS", "https://a.example,https://b.example")
	t.Setenv("READ_TIMEOUT", "30s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Server.Port != "9090" {
		t.Errorf("expected PORT override, got %s", cfg.Server.Port)
	}
	if cfg.Database.Type != "postgres" {
		t.Errorf("expected DB_TYPE override, got %s", cfg.Database.Type)
	}
	if len(cfg.Security.AllowedOrigins) != 2 || cfg.Security.AllowedOrigins[0] != "https://a.example" {
		t.Errorf("expected ALLOWED_ORIGINS to split on comma, got %v", cfg.Security.AllowedOrigins)
	}
	if cfg.Server.ReadTimeout != 30*time.Second {
		t.Errorf("expected READ_TIMEOUT override, got %v", cfg.Server.ReadTimeout)
	}
}

func TestLoad_InvalidNumericEnvFallsBackToDefault(t *testing.T) {
	t.Setenv("RATE_LIMIT_REQUESTS", "not-a-number")
	t.Setenv("MAX_BODY_SIZE", "not-a-number-either")
	t.Setenv("READ_TIMEOUT", "not-a-duration")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Security.RateLimitRequests != 100 {
		t.Errorf("expected malformed RATE_LIMIT_REQUESTS to fall back to default, got %d", cfg.Security.RateLimitRequests)
	}
}

func TestConfig_Validate(t *testing.T) {
	base := func() *Config {
		return &Config{
			Server:   ServerConfig{Port: "8080", Environment: "development"},
			Database: DatabaseConfig{Type: "file", FilePath: "posts.json"},
			Security: SecurityConfig{RateLimitRequests: 100, MaxBodySize: 1024},
			Logging:  LoggingConfig{Level: "info", Format: "json"},
		}
	}

	t.Run("valid config passes", func(t *testing.T) {
		if err := base().Validate(); err != nil {
			t.Errorf("expected a valid config to pass, got %v", err)
		}
	})
	t.Run("empty port fails", func(t *testing.T) {
		c := base()
		c.Server.Port = ""
		if err := c.Validate(); err == nil {
			t.Error("expected an error for empty port")
		}
	})
	t.Run("invalid environment fails", func(t *testing.T) {
		c := base()
		c.Server.Environment = "sandbox"
		if err := c.Validate(); err == nil {
			t.Error("expected an error for an invalid environment")
		}
	})
	t.Run("invalid db type fails", func(t *testing.T) {
		c := base()
		c.Database.Type = "mongodb"
		if err := c.Validate(); err == nil {
			t.Error("expected an error for an invalid database type")
		}
	})
	t.Run("file db without file path fails", func(t *testing.T) {
		c := base()
		c.Database.FilePath = ""
		if err := c.Validate(); err == nil {
			t.Error("expected an error for file db type with no file path")
		}
	})
	t.Run("postgres without host fails", func(t *testing.T) {
		c := base()
		c.Database.Type = "postgres"
		c.Database.DBName = "mydb"
		if err := c.Validate(); err == nil {
			t.Error("expected an error for postgres db type with no host")
		}
	})
	t.Run("postgres with host and name passes", func(t *testing.T) {
		c := base()
		c.Database.Type = "postgres"
		c.Database.Host = "localhost"
		c.Database.DBName = "mydb"
		if err := c.Validate(); err != nil {
			t.Errorf("expected a valid postgres config to pass, got %v", err)
		}
	})
	t.Run("non-positive rate limit fails", func(t *testing.T) {
		c := base()
		c.Security.RateLimitRequests = 0
		if err := c.Validate(); err == nil {
			t.Error("expected an error for a non-positive rate limit")
		}
	})
	t.Run("non-positive max body size fails", func(t *testing.T) {
		c := base()
		c.Security.MaxBodySize = 0
		if err := c.Validate(); err == nil {
			t.Error("expected an error for a non-positive max body size")
		}
	})
	t.Run("invalid log level fails", func(t *testing.T) {
		c := base()
		c.Logging.Level = "verbose"
		if err := c.Validate(); err == nil {
			t.Error("expected an error for an invalid log level")
		}
	})
	t.Run("invalid log format fails", func(t *testing.T) {
		c := base()
		c.Logging.Format = "xml"
		if err := c.Validate(); err == nil {
			t.Error("expected an error for an invalid log format")
		}
	})
}

func TestConfig_IsDevelopmentIsProduction(t *testing.T) {
	dev := &Config{Server: ServerConfig{Environment: "development"}}
	if !dev.IsDevelopment() || dev.IsProduction() {
		t.Error("expected IsDevelopment=true, IsProduction=false for environment=development")
	}

	prod := &Config{Server: ServerConfig{Environment: "production"}}
	if prod.IsDevelopment() || !prod.IsProduction() {
		t.Error("expected IsDevelopment=false, IsProduction=true for environment=production")
	}
}
