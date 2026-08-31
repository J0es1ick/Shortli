package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
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
	TrustedProxyCIDRs      string   `mapstructure:"TRUSTED_PROXY_CIDRS"`
	AdminBootstrapToken    string   `mapstructure:"ADMIN_BOOTSTRAP_TOKEN"`
	RequestTimeout         int      `mapstructure:"REQUEST_TIMEOUT_SECONDS"`
	AnalyticsSalt          string   `mapstructure:"ANALYTICS_SALT"`
	ClickSpoolPath         string   `mapstructure:"CLICK_SPOOL_PATH"`
	ClickSpoolMaxBytes     int64    `mapstructure:"CLICK_SPOOL_MAX_BYTES"`
	ClickBufferCapacity    int      `mapstructure:"CLICK_BUFFER_CAPACITY"`
	MetricsToken           string   `mapstructure:"METRICS_TOKEN"`
	ShutdownTimeout        int      `mapstructure:"SHUTDOWN_TIMEOUT_SECONDS"`
	AnalyticsRetentionDays int      `mapstructure:"ANALYTICS_RETENTION_DAYS"`
	ReportRetentionDays    int      `mapstructure:"REPORT_RETENTION_DAYS"`
	Database               Database `mapstructure:",squash"`
}

type Database struct {
	Host             string `mapstructure:"DATABASE_HOST"`
	Port             string `mapstructure:"DATABASE_PORT"`
	User             string `mapstructure:"DATABASE_USER"`
	Password         string `mapstructure:"DATABASE_PASSWORD"`
	Name             string `mapstructure:"DATABASE_NAME"`
	SSLMode          string `mapstructure:"DATABASE_SSLMODE"`
	ConnectTimeout   int    `mapstructure:"DATABASE_CONNECT_TIMEOUT_SECONDS"`
	StatementTimeout int    `mapstructure:"DATABASE_STATEMENT_TIMEOUT_MS"`
	LockTimeout      int    `mapstructure:"DATABASE_LOCK_TIMEOUT_MS"`
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
	v.SetDefault("TRUSTED_PROXY_CIDRS", "")
	v.SetDefault("ADMIN_BOOTSTRAP_TOKEN", "")
	v.SetDefault("REQUEST_TIMEOUT_SECONDS", 10)
	v.SetDefault("ANALYTICS_SALT", "shortli-local-development")
	v.SetDefault("CLICK_SPOOL_PATH", ".runtime/click-spool")
	v.SetDefault("CLICK_SPOOL_MAX_BYTES", 268435456)
	v.SetDefault("CLICK_BUFFER_CAPACITY", 1024)
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
	v.SetDefault("DATABASE_CONNECT_TIMEOUT_SECONDS", 5)
	v.SetDefault("DATABASE_STATEMENT_TIMEOUT_MS", 5000)
	v.SetDefault("DATABASE_LOCK_TIMEOUT_MS", 2000)

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
	if c.RequestTimeout < 1 || c.RequestTimeout > 60 {
		return fmt.Errorf("REQUEST_TIMEOUT_SECONDS must be between 1 and 60")
	}
	if c.Database.ConnectTimeout < 1 || c.Database.ConnectTimeout > 30 {
		return fmt.Errorf("DATABASE_CONNECT_TIMEOUT_SECONDS must be between 1 and 30")
	}
	if c.Database.StatementTimeout < 500 || c.Database.StatementTimeout > 60000 {
		return fmt.Errorf("DATABASE_STATEMENT_TIMEOUT_MS must be between 500 and 60000")
	}
	if c.Database.LockTimeout < 100 || c.Database.LockTimeout > c.Database.StatementTimeout {
		return fmt.Errorf("DATABASE_LOCK_TIMEOUT_MS must be between 100 and DATABASE_STATEMENT_TIMEOUT_MS")
	}
	for _, value := range strings.Split(c.TrustedProxyCIDRs, ",") {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, _, err := net.ParseCIDR(value); err != nil {
			return fmt.Errorf("TRUSTED_PROXY_CIDRS contains invalid CIDR %q", value)
		}
	}
	if c.AnalyticsRetentionDays < 30 || c.AnalyticsRetentionDays > 3650 {
		return fmt.Errorf("ANALYTICS_RETENTION_DAYS must be between 30 and 3650")
	}
	if c.ReportRetentionDays < 30 || c.ReportRetentionDays > 3650 {
		return fmt.Errorf("REPORT_RETENTION_DAYS must be between 30 and 3650")
	}
	if c.ClickSpoolMaxBytes < 1<<20 || c.ClickSpoolMaxBytes > 10<<30 {
		return fmt.Errorf("CLICK_SPOOL_MAX_BYTES must be between 1 MiB and 10 GiB")
	}
	if c.ClickBufferCapacity < 64 || c.ClickBufferCapacity > 65536 {
		return fmt.Errorf("CLICK_BUFFER_CAPACITY must be between 64 and 65536")
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
		publicURL, err := url.Parse(c.PublicBaseURL)
		if err != nil || publicURL.Hostname() == "" || !isHTTPScheme(publicURL.Scheme) {
			return fmt.Errorf("PUBLIC_BASE_URL must be an absolute HTTP or HTTPS URL")
		}
		frontendURL, err := url.Parse(c.FrontendOrigin)
		if err != nil || frontendURL.Hostname() == "" || !isHTTPScheme(frontendURL.Scheme) {
			return fmt.Errorf("FRONTEND_ORIGIN must be an absolute HTTP or HTTPS URL")
		}
		localOnly := isLoopbackHost(publicURL.Hostname()) && isLoopbackHost(frontendURL.Hostname())
		if !localOnly && (publicURL.Scheme != "https" || frontendURL.Scheme != "https") {
			return fmt.Errorf("PUBLIC_BASE_URL and FRONTEND_ORIGIN must use HTTPS in production")
		}
		if !localOnly && !c.CookieSecure {
			return fmt.Errorf("COOKIE_SECURE must be true for a public production deployment")
		}
		if len(c.MetricsToken) < 24 {
			return fmt.Errorf("METRICS_TOKEN must be at least 24 characters in production")
		}
		if c.AdminBootstrapToken != "" && len(c.AdminBootstrapToken) < 32 {
			return fmt.Errorf("ADMIN_BOOTSTRAP_TOKEN must be at least 32 characters in production")
		}
	}
	return nil
}

func isHTTPScheme(scheme string) bool {
	return scheme == "http" || scheme == "https"
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
