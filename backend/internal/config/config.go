package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	AppEnv                 string   `mapstructure:"APP_ENV"`
	ServerPort             string   `mapstructure:"SERVER_PORT"`
	PublicBaseURL          string   `mapstructure:"PUBLIC_BASE_URL"`
	FrontendOrigin         string   `mapstructure:"FRONTEND_ORIGIN"`
	CookieSecure           bool     `mapstructure:"COOKIE_SECURE"`
	TrustProxyHeaders      bool     `mapstructure:"TRUST_PROXY_HEADERS"`
	AnalyticsSalt          string   `mapstructure:"ANALYTICS_SALT"`
	ClickSpoolPath         string   `mapstructure:"CLICK_SPOOL_PATH"`
	MetricsToken           string   `mapstructure:"METRICS_TOKEN"`
	ShutdownTimeout        int      `mapstructure:"SHUTDOWN_TIMEOUT_SECONDS"`
	AnalyticsRetentionDays int      `mapstructure:"ANALYTICS_RETENTION_DAYS"`
	ReportRetentionDays    int      `mapstructure:"REPORT_RETENTION_DAYS"`
	Database               Database `mapstructure:",squash"`
}

type Database struct {
	Host     string `mapstructure:"DATABASE_HOST"`
	Port     string `mapstructure:"DATABASE_PORT"`
	User     string `mapstructure:"DATABASE_USER"`
	Password string `mapstructure:"DATABASE_PASSWORD"`
	Name     string `mapstructure:"DATABASE_NAME"`
	SSLMode  string `mapstructure:"DATABASE_SSLMODE"`
}

func InitConfig() (*Config, error) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("resolve working directory: %w", err)
	}

	v := viper.New()
	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AutomaticEnv()

	v.SetDefault("APP_ENV", "development")
	v.SetDefault("SERVER_PORT", "43178")
	v.SetDefault("PUBLIC_BASE_URL", "")
	v.SetDefault("FRONTEND_ORIGIN", "http://localhost:43177")
	v.SetDefault("COOKIE_SECURE", false)
	v.SetDefault("TRUST_PROXY_HEADERS", false)
	v.SetDefault("ANALYTICS_SALT", "shortli-local-development")
	v.SetDefault("CLICK_SPOOL_PATH", ".runtime/click-spool")
	v.SetDefault("METRICS_TOKEN", "")
	v.SetDefault("SHUTDOWN_TIMEOUT_SECONDS", 20)
	v.SetDefault("ANALYTICS_RETENTION_DAYS", 365)
	v.SetDefault("REPORT_RETENTION_DAYS", 180)
	v.SetDefault("DATABASE_PORT", "5432")
	v.SetDefault("DATABASE_SSLMODE", "disable")
	v.SetDefault("DATABASE_HOST", "")
	v.SetDefault("DATABASE_USER", "")
	v.SetDefault("DATABASE_PASSWORD", "")
	v.SetDefault("DATABASE_NAME", "")

	if configFile := strings.TrimSpace(os.Getenv("CONFIG_FILE")); configFile != "" {
		v.SetConfigFile(configFile)
	} else {
		v.AddConfigPath(workingDirectory)
		v.AddConfigPath(filepath.Join(workingDirectory, "backend"))
		v.AddConfigPath(filepath.Dir(workingDirectory))
	}

	if err = v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	cfg.AppEnv = strings.ToLower(strings.TrimSpace(cfg.AppEnv))
	cfg.PublicBaseURL = strings.TrimRight(strings.TrimSpace(cfg.PublicBaseURL), "/")
	cfg.FrontendOrigin = strings.TrimRight(strings.TrimSpace(cfg.FrontendOrigin), "/")
	cfg.Database.SSLMode = strings.TrimSpace(cfg.Database.SSLMode)

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) validate() error {
	required := map[string]string{
		"DATABASE_HOST": c.Database.Host,
		"DATABASE_USER": c.Database.User,
		"DATABASE_NAME": c.Database.Name,
	}
	for name, value := range required {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	if _, err := strconv.Atoi(c.ServerPort); err != nil {
		return fmt.Errorf("SERVER_PORT must be numeric")
	}
	if c.ShutdownTimeout < 5 || c.ShutdownTimeout > 120 {
		return fmt.Errorf("SHUTDOWN_TIMEOUT_SECONDS must be between 5 and 120")
	}
	if c.AnalyticsRetentionDays < 30 || c.AnalyticsRetentionDays > 3650 {
		return fmt.Errorf("ANALYTICS_RETENTION_DAYS must be between 30 and 3650")
	}
	if c.ReportRetentionDays < 30 || c.ReportRetentionDays > 3650 {
		return fmt.Errorf("REPORT_RETENTION_DAYS must be between 30 and 3650")
	}

	if c.AppEnv == "production" {
		if len(c.Database.Password) < 12 || c.Database.Password == "12345" {
			return fmt.Errorf("DATABASE_PASSWORD must be a strong secret in production")
		}
		if len(c.AnalyticsSalt) < 32 || c.AnalyticsSalt == "shortli-local-development" {
			return fmt.Errorf("ANALYTICS_SALT must be at least 32 characters in production")
		}
		if c.PublicBaseURL == "" {
			return fmt.Errorf("PUBLIC_BASE_URL is required in production")
		}
		if c.FrontendOrigin == "" {
			return fmt.Errorf("FRONTEND_ORIGIN is required in production")
		}
		if len(c.MetricsToken) < 24 {
			return fmt.Errorf("METRICS_TOKEN must be at least 24 characters in production")
		}
	}
	return nil
}
