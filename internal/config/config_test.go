package config

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"
)

const testPassword = "not-a-real-password-9f3a"

func clearDBEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		envAppEnv, envLogLevel,
		envDBHost, envDBPort, envDBUser, envDBPassword, envDBName,
		envDBSSLMode, envDBMaxConns, envDBMinConns, envDBConnTimeout,
	} {
		t.Setenv(name, "")
	}
}

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv(envDBHost, "localhost")
	t.Setenv(envDBUser, "mybank_app")
	t.Setenv(envDBPassword, testPassword)
	t.Setenv(envDBName, "mybank")
}

func assertNoSecret(t *testing.T, err error) {
	t.Helper()
	if err != nil && strings.Contains(err.Error(), testPassword) {
		t.Fatalf("error leaked the database password: %v", err)
	}
}

func applicationField(cfg *Config, env string) string {
	if env == envAppEnv {
		return cfg.AppEnv
	}
	return cfg.LogLevel
}

func TestLoadConfigMissingRequiredVariables(t *testing.T) {
	for _, name := range []string{envDBHost, envDBUser, envDBPassword, envDBName} {
		t.Run(name, func(t *testing.T) {
			clearDBEnv(t)
			setRequiredEnv(t)
			t.Setenv(name, "")

			cfg, err := LoadConfig()
			assertNoSecret(t, err)
			if err == nil {
				t.Fatalf("expected an error when %s is missing", name)
			}
			if cfg != nil {
				t.Fatalf("expected a nil config on error, got %v", cfg)
			}
			if !strings.Contains(err.Error(), name) {
				t.Fatalf("expected error to name %s, got: %v", name, err)
			}
		})
	}
}

func TestLoadConfigValidConfiguration(t *testing.T) {
	clearDBEnv(t)
	t.Setenv(envAppEnv, "Production")
	t.Setenv(envLogLevel, "debug")
	t.Setenv(envDBHost, "db.internal.example")
	t.Setenv(envDBPort, "6543")
	t.Setenv(envDBUser, "mybank_app")
	t.Setenv(envDBPassword, testPassword)
	t.Setenv(envDBName, "mybank_prod")
	t.Setenv(envDBSSLMode, "verify-full")
	t.Setenv(envDBMaxConns, "40")
	t.Setenv(envDBMinConns, "10")
	t.Setenv(envDBConnTimeout, "12s")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected valid configuration to load, got: %v", err)
	}

	want := Config{
		AppEnv:        "production",
		LogLevel:      "DEBUG",
		DBHost:        "db.internal.example",
		DBPort:        6543,
		DBUser:        "mybank_app",
		DBPassword:    testPassword,
		DBName:        "mybank_prod",
		DBSSLMode:     "verify-full",
		DBMaxConns:    40,
		DBMinConns:    10,
		DBConnTimeout: 12 * time.Second,
	}
	if *cfg != want {
		t.Errorf("unexpected config:\n got: %v\nwant: %v", cfg, &want)
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	clearDBEnv(t)
	setRequiredEnv(t)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected defaults to load, got: %v", err)
	}
	if cfg.AppEnv != DefaultAppEnv {
		t.Errorf("AppEnv: got %q, want %q", cfg.AppEnv, DefaultAppEnv)
	}
	if cfg.LogLevel != DefaultLogLevel {
		t.Errorf("LogLevel: got %q, want %q", cfg.LogLevel, DefaultLogLevel)
	}
	if cfg.DBPort != DefaultDBPort {
		t.Errorf("DBPort: got %d, want %d", cfg.DBPort, DefaultDBPort)
	}
	if cfg.DBSSLMode != DefaultDBSSLMode {
		t.Errorf("DBSSLMode: got %q, want %q", cfg.DBSSLMode, DefaultDBSSLMode)
	}
	if cfg.DBMaxConns != DefaultDBMaxConns {
		t.Errorf("DBMaxConns: got %d, want %d", cfg.DBMaxConns, DefaultDBMaxConns)
	}
	if cfg.DBMinConns != DefaultDBMinConns {
		t.Errorf("DBMinConns: got %d, want %d", cfg.DBMinConns, DefaultDBMinConns)
	}
	if cfg.DBConnTimeout != DefaultDBConnTimeout {
		t.Errorf("DBConnTimeout: got %s, want %s", cfg.DBConnTimeout, DefaultDBConnTimeout)
	}
}

func TestLoadConfigApplicationDefaults(t *testing.T) {
	cases := []struct {
		name  string
		env   string
		field func(*Config) string
		want  string
	}{
		{name: "APP_ENV", env: envAppEnv, field: func(c *Config) string { return c.AppEnv }, want: DefaultAppEnv},
		{name: "LOG_LEVEL", env: envLogLevel, field: func(c *Config) string { return c.LogLevel }, want: DefaultLogLevel},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clearDBEnv(t)
			setRequiredEnv(t)

			cfg, err := LoadConfig()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := tc.field(cfg); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestLoadConfigApplicationValidValues(t *testing.T) {
	cases := []struct {
		name  string
		env   string
		value string
		want  string
	}{
		{name: "development", env: envAppEnv, value: "development", want: "development"},
		{name: "uppercase staging", env: envAppEnv, value: "STAGING", want: "staging"},
		{name: "mixed case production", env: envAppEnv, value: "Production", want: "production"},
		{name: "debug", env: envLogLevel, value: "debug", want: "DEBUG"},
		{name: "info", env: envLogLevel, value: "Info", want: "INFO"},
		{name: "warning alias", env: envLogLevel, value: "warning", want: "WARN"},
		{name: "error", env: envLogLevel, value: "ERROR", want: "ERROR"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clearDBEnv(t)
			setRequiredEnv(t)
			t.Setenv(tc.env, tc.value)

			cfg, err := LoadConfig()
			if err != nil {
				t.Fatalf("unexpected error for %s=%q: %v", tc.env, tc.value, err)
			}
			if got := applicationField(cfg, tc.env); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestLoadConfigApplicationInvalidValues(t *testing.T) {
	cases := []struct {
		name  string
		env   string
		value string
	}{
		{name: "unknown environment", env: envAppEnv, value: "qa"},
		{name: "unsupported environment", env: envAppEnv, value: "prod"},
		{name: "unknown log level", env: envLogLevel, value: "TRACE"},
		{name: "numeric log level", env: envLogLevel, value: "1"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clearDBEnv(t)
			setRequiredEnv(t)
			t.Setenv(tc.env, tc.value)

			cfg, err := LoadConfig()
			assertNoSecret(t, err)
			if err == nil {
				t.Fatalf("expected an error for %s=%q", tc.env, tc.value)
			}
			if cfg != nil {
				t.Fatalf("expected a nil config on error, got %v", cfg)
			}
			if !strings.Contains(err.Error(), tc.env) {
				t.Fatalf("expected error to name %s, got: %v", tc.env, err)
			}
		})
	}
}

func TestLoadConfigInvalidPort(t *testing.T) {
	for _, value := range []string{"0", "65536", "-1", "abc"} {
		t.Run(value, func(t *testing.T) {
			clearDBEnv(t)
			setRequiredEnv(t)
			t.Setenv(envDBPort, value)

			cfg, err := LoadConfig()
			assertNoSecret(t, err)
			if err == nil {
				t.Fatalf("expected an error for DB_PORT=%q", value)
			}
			if cfg != nil {
				t.Fatalf("expected a nil config on error, got %v", cfg)
			}
			if !strings.Contains(err.Error(), envDBPort) {
				t.Fatalf("expected error to name %s, got: %v", envDBPort, err)
			}
		})
	}
}

func TestLoadConfigInvalidTimeout(t *testing.T) {
	for _, value := range []string{"not-a-duration", "0s", "-2s"} {
		t.Run(value, func(t *testing.T) {
			clearDBEnv(t)
			setRequiredEnv(t)
			t.Setenv(envDBConnTimeout, value)

			cfg, err := LoadConfig()
			assertNoSecret(t, err)
			if err == nil {
				t.Fatalf("expected an error for DB_CONN_TIMEOUT=%q", value)
			}
			if cfg != nil {
				t.Fatalf("expected a nil config on error, got %v", cfg)
			}
			if !strings.Contains(err.Error(), envDBConnTimeout) {
				t.Fatalf("expected error to name %s, got: %v", envDBConnTimeout, err)
			}
		})
	}
}

func TestLoadConfigInvalidPoolLimits(t *testing.T) {
	cases := []struct {
		name   string
		values map[string]string
	}{
		{name: "max is zero", values: map[string]string{envDBMaxConns: "0"}},
		{name: "max is negative", values: map[string]string{envDBMaxConns: "-5"}},
		{name: "min is negative", values: map[string]string{envDBMinConns: "-1"}},
		{name: "min exceeds max", values: map[string]string{envDBMaxConns: "3", envDBMinConns: "4"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clearDBEnv(t)
			setRequiredEnv(t)
			for name, value := range tc.values {
				t.Setenv(name, value)
			}

			cfg, err := LoadConfig()
			assertNoSecret(t, err)
			if err == nil {
				t.Fatalf("expected an error for %s", tc.name)
			}
			if cfg != nil {
				t.Fatalf("expected a nil config on error, got %v", cfg)
			}
		})
	}
}

func TestLoadConfigInvalidSSLMode(t *testing.T) {
	clearDBEnv(t)
	setRequiredEnv(t)
	t.Setenv(envDBSSLMode, "insecure")

	cfg, err := LoadConfig()
	assertNoSecret(t, err)
	if err == nil {
		t.Fatal("expected an error for an unsupported SSL mode")
	}
	if cfg != nil {
		t.Fatalf("expected a nil config on error, got %v", cfg)
	}
}

func TestConfigStringRedactsPassword(t *testing.T) {
	clearDBEnv(t)
	setRequiredEnv(t)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected config to load, got: %v", err)
	}
	if strings.Contains(cfg.String(), testPassword) {
		t.Fatalf("String() leaked the database password: %s", cfg)
	}
}

func TestConfigLogValueRedactsPassword(t *testing.T) {
	clearDBEnv(t)
	setRequiredEnv(t)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected config to load, got: %v", err)
	}

	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))
	log.Info("configuration", slog.Any("config", cfg))

	if strings.Contains(buf.String(), testPassword) {
		t.Fatalf("slog JSON output leaked the password: %s", buf.String())
	}

	var record map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &record); err != nil {
		t.Fatalf("invalid JSON log output: %v", err)
	}
	fields, ok := record["config"].(map[string]any)
	if !ok {
		t.Fatalf("expected a structured config group, got: %v", record["config"])
	}
	if fields["password"] != "[REDACTED]" {
		t.Fatalf("expected redacted password, got %v", fields["password"])
	}
	if fields["app_env"] != DefaultAppEnv {
		t.Errorf("app_env: got %v, want %q", fields["app_env"], DefaultAppEnv)
	}
	if fields["log_level"] != DefaultLogLevel {
		t.Errorf("log_level: got %v, want %q", fields["log_level"], DefaultLogLevel)
	}
}
