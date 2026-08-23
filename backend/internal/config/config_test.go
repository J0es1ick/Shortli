package config

import (
	"strings"
	"testing"
)

func TestInitConfigReadsContainerEnvironment(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("SERVER_PORT", "43178")
	t.Setenv("DATABASE_HOST", "postgres")
	t.Setenv("DATABASE_PORT", "5432")
	t.Setenv("DATABASE_USER", "shortli")
	t.Setenv("DATABASE_PASSWORD", "a-strong-database-password")
	t.Setenv("DATABASE_NAME", "shortli")
	t.Setenv("DATABASE_SSLMODE", "disable")
	t.Setenv("PUBLIC_BASE_URL", "https://go.example.com")
	t.Setenv("FRONTEND_ORIGIN", "https://go.example.com")
	t.Setenv("COOKIE_SECURE", "true")
	t.Setenv("ANALYTICS_SALT", "0123456789abcdef0123456789abcdef")
	t.Setenv("METRICS_TOKEN", "0123456789abcdef01234567")

	cfg, err := InitConfig()
	if err != nil {
		t.Fatalf("InitConfig: %v", err)
	}
	if cfg.Database.Host != "postgres" || cfg.Database.Name != "shortli" {
		t.Fatalf("database environment not loaded: %+v", cfg.Database)
	}
	if cfg.PublicBaseURL != "https://go.example.com" {
		t.Fatalf("public base URL = %q", cfg.PublicBaseURL)
	}
}

func TestInitConfigRejectsInsecurePublicProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("SERVER_PORT", "43178")
	t.Setenv("DATABASE_HOST", "postgres")
	t.Setenv("DATABASE_USER", "shortli")
	t.Setenv("DATABASE_PASSWORD", "a-strong-database-password")
	t.Setenv("DATABASE_NAME", "shortli")
	t.Setenv("PUBLIC_BASE_URL", "http://go.example.com")
	t.Setenv("FRONTEND_ORIGIN", "http://go.example.com")
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("ANALYTICS_SALT", "0123456789abcdef0123456789abcdef")
	t.Setenv("METRICS_TOKEN", "0123456789abcdef01234567")

	if _, err := InitConfig(); err == nil {
		t.Fatal("InitConfig() accepted an insecure public production origin")
	}
}

func TestInitConfigRejectsUnsupportedProductionURLScheme(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("DATABASE_HOST", "postgres")
	t.Setenv("DATABASE_USER", "shortli")
	t.Setenv("DATABASE_PASSWORD", "strong-password-value")
	t.Setenv("DATABASE_NAME", "shortli")
	t.Setenv("ANALYTICS_SALT", "analytics-salt-value-with-32-characters")
	t.Setenv("METRICS_TOKEN", "metrics-token-value-24-chars")
	t.Setenv("PUBLIC_BASE_URL", "ftp://localhost:43176")
	t.Setenv("FRONTEND_ORIGIN", "http://localhost:43176")

	_, err := InitConfig()
	if err == nil || !strings.Contains(err.Error(), "HTTP or HTTPS") {
		t.Fatalf("expected unsupported scheme error, got %v", err)
	}
}
