package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultAppEnv        = "development"
	DefaultLogLevel      = "INFO"
	DefaultDBPort        = 5432
	DefaultDBSSLMode     = "require"
	DefaultDBMaxConns    = 25
	DefaultDBMinConns    = 5
	DefaultDBConnTimeout = 5 * time.Second
)

const (
	envAppEnv        = "APP_ENV"
	envLogLevel      = "LOG_LEVEL"
	envDBHost        = "DB_HOST"
	envDBPort        = "DB_PORT"
	envDBUser        = "DB_USER"
	envDBPassword    = "DB_PASSWORD"
	envDBName        = "DB_NAME"
	envDBSSLMode     = "DB_SSLMODE"
	envDBMaxConns    = "DB_MAX_CONNS"
	envDBMinConns    = "DB_MIN_CONNS"
	envDBConnTimeout = "DB_CONN_TIMEOUT"
)

type Config struct {
	AppEnv        string
	LogLevel      string
	DBHost        string
	DBPort        int
	DBUser        string
	DBPassword    string
	DBName        string
	DBSSLMode     string
	DBMaxConns    int
	DBMinConns    int
	DBConnTimeout time.Duration
}

func (c Config) String() string {
	return fmt.Sprintf(
		"Config{AppEnv:%q LogLevel:%q DBHost:%q DBPort:%d DBUser:%q DBName:%q DBSSLMode:%q DBMaxConns:%d DBMinConns:%d DBConnTimeout:%s DBPassword:[REDACTED]}",
		c.AppEnv, c.LogLevel, c.DBHost, c.DBPort, c.DBUser, c.DBName, c.DBSSLMode,
		c.DBMaxConns, c.DBMinConns, c.DBConnTimeout,
	)
}

func (c Config) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("app_env", c.AppEnv),
		slog.String("log_level", c.LogLevel),
		slog.String("host", c.DBHost),
		slog.Int("port", c.DBPort),
		slog.String("user", c.DBUser),
		slog.String("database", c.DBName),
		slog.String("sslmode", c.DBSSLMode),
		slog.Int("max_conns", c.DBMaxConns),
		slog.Int("min_conns", c.DBMinConns),
		slog.Duration("conn_timeout", c.DBConnTimeout),
		slog.String("password", "[REDACTED]"),
	)
}

func LoadConfig() (*Config, error) {
	cfg := &Config{
		AppEnv:        DefaultAppEnv,
		LogLevel:      DefaultLogLevel,
		DBHost:        strings.TrimSpace(os.Getenv(envDBHost)),
		DBPort:        DefaultDBPort,
		DBUser:        strings.TrimSpace(os.Getenv(envDBUser)),
		DBPassword:    os.Getenv(envDBPassword),
		DBName:        strings.TrimSpace(os.Getenv(envDBName)),
		DBSSLMode:     DefaultDBSSLMode,
		DBMaxConns:    DefaultDBMaxConns,
		DBMinConns:    DefaultDBMinConns,
		DBConnTimeout: DefaultDBConnTimeout,
	}

	if err := loadOptional(cfg); err != nil {
		return nil, err
	}
	if err := validate(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func loadOptional(cfg *Config) error {
	if raw, ok := lookup(envAppEnv); ok {
		cfg.AppEnv = strings.ToLower(raw)
	}
	if raw, ok := lookup(envLogLevel); ok {
		level := strings.ToUpper(raw)
		if level == "WARNING" {
			level = "WARN"
		}
		cfg.LogLevel = level
	}
	if err := parseInt(envDBPort, &cfg.DBPort); err != nil {
		return err
	}
	if err := parseInt(envDBMaxConns, &cfg.DBMaxConns); err != nil {
		return err
	}
	if err := parseInt(envDBMinConns, &cfg.DBMinConns); err != nil {
		return err
	}
	if err := parseDuration(envDBConnTimeout, &cfg.DBConnTimeout); err != nil {
		return err
	}
	if raw, ok := lookup(envDBSSLMode); ok {
		cfg.DBSSLMode = strings.ToLower(raw)
	}
	return nil
}

func lookup(name string) (string, bool) {
	raw, ok := os.LookupEnv(name)
	if !ok {
		return "", false
	}
	raw = strings.TrimSpace(raw)
	return raw, raw != ""
}

func parseInt(name string, target *int) error {
	raw, ok := lookup(name)
	if !ok {
		return nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fmt.Errorf("config: %s must be an integer, got %q", name, raw)
	}
	*target = value
	return nil
}

func parseDuration(name string, target *time.Duration) error {
	raw, ok := lookup(name)
	if !ok {
		return nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return fmt.Errorf("config: %s must be a duration such as 5s, got %q", name, raw)
	}
	*target = value
	return nil
}

func validate(cfg *Config) error {
	switch cfg.AppEnv {
	case "development", "staging", "production":
	default:
		return fmt.Errorf("config: %s must be one of development, staging, production, got %q", envAppEnv, cfg.AppEnv)
	}

	switch cfg.LogLevel {
	case "DEBUG", "INFO", "WARN", "ERROR":
	default:
		return fmt.Errorf("config: %s must be one of DEBUG, INFO, WARN, ERROR, got %q", envLogLevel, cfg.LogLevel)
	}

	for _, field := range []struct{ name, value string }{
		{envDBHost, cfg.DBHost},
		{envDBUser, cfg.DBUser},
		{envDBPassword, cfg.DBPassword},
		{envDBName, cfg.DBName},
	} {
		if field.value == "" {
			return fmt.Errorf("config: required environment variable %s is not set", field.name)
		}
	}

	if cfg.DBPort < 1 || cfg.DBPort > 65535 {
		return fmt.Errorf("config: %s must be between 1 and 65535, got %d", envDBPort, cfg.DBPort)
	}

	switch cfg.DBSSLMode {
	case "disable", "allow", "prefer", "require", "verify-ca", "verify-full":
	default:
		return fmt.Errorf("config: %s must be one of disable, allow, prefer, require, verify-ca, verify-full, got %q", envDBSSLMode, cfg.DBSSLMode)
	}

	if cfg.DBMaxConns < 1 {
		return fmt.Errorf("config: %s must be greater than 0, got %d", envDBMaxConns, cfg.DBMaxConns)
	}
	if cfg.DBMinConns < 0 {
		return fmt.Errorf("config: %s cannot be negative, got %d", envDBMinConns, cfg.DBMinConns)
	}
	if cfg.DBMinConns > cfg.DBMaxConns {
		return fmt.Errorf("config: %s (%d) cannot exceed %s (%d)", envDBMinConns, cfg.DBMinConns, envDBMaxConns, cfg.DBMaxConns)
	}
	if cfg.DBConnTimeout <= 0 {
		return fmt.Errorf("config: %s must be greater than 0, got %s", envDBConnTimeout, cfg.DBConnTimeout)
	}
	return nil
}
