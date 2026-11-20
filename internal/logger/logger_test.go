package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"mybank/internal/config"
)

func decodeRecord(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var record map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(data), &record); err != nil {
		t.Fatalf("invalid JSON log record %q: %v", data, err)
	}
	return record
}

func TestNewProducesStructuredJSON(t *testing.T) {
	var buf bytes.Buffer
	log, err := New(Options{Level: "INFO", Out: &buf})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	log.Info("account created", slog.Int("account_id", 42), slog.Bool("active", true))

	record := decodeRecord(t, buf.Bytes())
	if record["msg"] != "account created" {
		t.Errorf("msg: got %v, want %q", record["msg"], "account created")
	}
	if record["level"] != "INFO" {
		t.Errorf("level: got %v, want INFO", record["level"])
	}
	if record["account_id"] != float64(42) {
		t.Errorf("account_id: got %v, want 42", record["account_id"])
	}
	if record["active"] != true {
		t.Errorf("active: got %v, want true", record["active"])
	}
	if _, ok := record["time"]; !ok {
		t.Error("expected a time field")
	}
}

func TestParseLevel(t *testing.T) {
	cases := []struct {
		raw  string
		want slog.Level
	}{
		{raw: "DEBUG", want: slog.LevelDebug},
		{raw: " debug ", want: slog.LevelDebug},
		{raw: "INFO", want: slog.LevelInfo},
		{raw: "WARN", want: slog.LevelWarn},
		{raw: "warning", want: slog.LevelWarn},
		{raw: "ERROR", want: slog.LevelError},
	}

	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			got, err := ParseLevel(tc.raw)
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.raw, err)
			}
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestParseLevelRejectsInvalidValues(t *testing.T) {
	for _, raw := range []string{"", "TRACE", "verbose", "1"} {
		t.Run(raw, func(t *testing.T) {
			if _, err := ParseLevel(raw); err == nil {
				t.Fatalf("expected an error for level %q", raw)
			}
		})
	}
}

func TestNewRejectsInvalidLevel(t *testing.T) {
	var buf bytes.Buffer
	log, err := New(Options{Level: "TRACE", Out: &buf})
	if err == nil {
		t.Fatal("expected an error for an unsupported level")
	}
	if log != nil {
		t.Fatalf("expected a nil logger on error, got %v", log)
	}
	if buf.Len() != 0 {
		t.Fatalf("expected no output, got %q", buf.String())
	}
	if !strings.Contains(err.Error(), "supported: DEBUG, INFO, WARN, ERROR") {
		t.Fatalf("expected the error to list supported levels, got: %v", err)
	}
}

func TestNewDefaultsAreSafe(t *testing.T) {
	log, err := New(Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if log == nil {
		t.Fatal("expected a logger")
	}
	ctx := context.Background()
	if !log.Enabled(ctx, slog.LevelInfo) {
		t.Error("expected INFO to be enabled by default")
	}
	if log.Enabled(ctx, slog.LevelDebug) {
		t.Error("expected DEBUG to be disabled by default")
	}
}

func TestLevelFiltering(t *testing.T) {
	cases := []struct {
		level    string
		expected []string
	}{
		{level: "DEBUG", expected: []string{"DEBUG", "INFO", "WARN", "ERROR"}},
		{level: "INFO", expected: []string{"INFO", "WARN", "ERROR"}},
		{level: "WARN", expected: []string{"WARN", "ERROR"}},
		{level: "ERROR", expected: []string{"ERROR"}},
	}

	for _, tc := range cases {
		t.Run(tc.level, func(t *testing.T) {
			var buf bytes.Buffer
			log, err := New(Options{Level: tc.level, Out: &buf})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			log.Debug("debug message")
			log.Info("info message")
			log.Warn("warn message")
			log.Error("error message")

			lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
			if len(lines) != len(tc.expected) {
				t.Fatalf("expected %d records, got %d: %q", len(tc.expected), len(lines), buf.String())
			}
			for i, line := range lines {
				record := decodeRecord(t, []byte(line))
				if record["level"] != tc.expected[i] {
					t.Errorf("record %d: got level %v, want %s", i, record["level"], tc.expected[i])
				}
			}
		})
	}
}

func TestContextualFieldsIncludeRequestID(t *testing.T) {
	var buf bytes.Buffer
	log, err := New(Options{Level: "INFO", Out: &buf})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	requestLog := log.With(slog.String("request_id", "req-123"), slog.Int("customer_id", 7))
	requestLog.Info("balance checked")

	record := decodeRecord(t, buf.Bytes())
	if record["request_id"] != "req-123" {
		t.Errorf("request_id: got %v, want req-123", record["request_id"])
	}
	if record["customer_id"] != float64(7) {
		t.Errorf("customer_id: got %v, want 7", record["customer_id"])
	}
}

func TestCustomOutputWriter(t *testing.T) {
	var buf bytes.Buffer
	log, err := New(Options{Level: "INFO", Out: &buf})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	log.Info("written to custom writer")

	if buf.Len() == 0 {
		t.Fatal("expected the supplied writer to receive the record")
	}
	record := decodeRecord(t, buf.Bytes())
	if record["msg"] != "written to custom writer" {
		t.Errorf("unexpected message: %v", record["msg"])
	}
}

func TestSensitiveAttributesAreRedacted(t *testing.T) {
	var buf bytes.Buffer
	log, err := New(Options{Level: "INFO", Out: &buf})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	const dsn = "postgres://app:hunter2@db.internal:5432/mybank"
	log.Info("connection setup",
		slog.String("password", "hunter2"),
		slog.String("dsn", dsn),
		slog.String("host", "db.internal"),
	)

	output := buf.String()
	if strings.Contains(output, "hunter2") || strings.Contains(output, dsn) {
		t.Fatalf("sensitive value leaked: %s", output)
	}
	record := decodeRecord(t, buf.Bytes())
	if record["password"] != "[REDACTED]" || record["dsn"] != "[REDACTED]" {
		t.Fatalf("expected redacted fields, got: %v", record)
	}
	if record["host"] != "db.internal" {
		t.Errorf("expected non-sensitive field to survive, got: %v", record["host"])
	}
}

func TestConfigIsRedactedWhenLoggedAsAny(t *testing.T) {
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_USER", "mybank_app")
	t.Setenv("DB_PASSWORD", "not-a-real-password-9f3a")
	t.Setenv("DB_NAME", "mybank")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected config error: %v", err)
	}

	var buf bytes.Buffer
	log, err := New(Options{Level: "INFO", Out: &buf})
	if err != nil {
		t.Fatalf("unexpected logger error: %v", err)
	}
	log.Info("configuration loaded", slog.Any("config", cfg))

	output := buf.String()
	if strings.Contains(output, "not-a-real-password-9f3a") {
		t.Fatalf("database password leaked: %s", output)
	}
	record := decodeRecord(t, buf.Bytes())
	fields, ok := record["config"].(map[string]any)
	if !ok {
		t.Fatalf("expected structured config fields, got: %v", record["config"])
	}
	if fields["password"] != "[REDACTED]" {
		t.Fatalf("expected redacted password, got: %v", fields["password"])
	}
	if fields["host"] != "localhost" {
		t.Errorf("expected host to be present, got: %v", fields["host"])
	}
}

func TestExtendedSensitiveAttributeNamesAreRedacted(t *testing.T) {
	keys := []string{
		"password", "passwd", "pwd", "db_password", "database_url",
		"secret", "client_secret", "token", "access_token", "refresh_token",
		"dsn", "connection_string", "conn_string", "credentials",
		"api_key", "authorization", "DB_PASSWORD", "Authorization", "Access_Token",
	}

	var buf bytes.Buffer
	log, err := New(Options{Level: "INFO", Out: &buf})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	attrs := make([]slog.Attr, 0, len(keys))
	values := make(map[string]string, len(keys))
	for _, key := range keys {
		value := "do-not-log-" + key
		values[key] = value
		attrs = append(attrs, slog.String(key, value))
	}
	log.LogAttrs(context.Background(), slog.LevelInfo, "sensitive fields", attrs...)

	output := buf.String()
	for key, value := range values {
		if strings.Contains(output, value) {
			t.Errorf("secret for %q leaked: %s", key, output)
		}
	}
	record := decodeRecord(t, buf.Bytes())
	for _, key := range keys {
		if record[key] != "[REDACTED]" {
			t.Errorf("%q: got %v, want [REDACTED]", key, record[key])
		}
	}
}

func TestNestedGroupSecretsAreRedacted(t *testing.T) {
	var buf bytes.Buffer
	log, err := New(Options{Level: "INFO", Out: &buf})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	log.Info("nested secrets",
		slog.Group("db",
			slog.String("host", "db.internal"),
			slog.Group("credentials",
				slog.String("password", "nested-password"),
				slog.String("access_token", "nested-token"),
			),
		),
	)

	output := buf.String()
	for _, secret := range []string{"nested-password", "nested-token"} {
		if strings.Contains(output, secret) {
			t.Fatalf("nested secret leaked: %s", output)
		}
	}

	record := decodeRecord(t, buf.Bytes())
	db, ok := record["db"].(map[string]any)
	if !ok {
		t.Fatalf("expected db group, got: %v", record["db"])
	}
	creds, ok := db["credentials"].(map[string]any)
	if !ok {
		t.Fatalf("expected credentials group, got: %v", db["credentials"])
	}
	if creds["password"] != "[REDACTED]" || creds["access_token"] != "[REDACTED]" {
		t.Fatalf("expected nested fields to be redacted, got: %v", creds)
	}
	if db["host"] != "db.internal" {
		t.Errorf("expected non-sensitive field to survive, got: %v", db["host"])
	}
}

func TestRedactionAppliesToWithAttributes(t *testing.T) {
	var buf bytes.Buffer
	log, err := New(Options{Level: "INFO", Out: &buf})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	requestLog := log.With(slog.Group("auth", slog.String("authorization", "Bearer with-attr-token")))
	requestLog.Info("authenticated")

	if strings.Contains(buf.String(), "with-attr-token") {
		t.Fatalf("secret in With attributes leaked: %s", buf.String())
	}
	record := decodeRecord(t, buf.Bytes())
	auth, ok := record["auth"].(map[string]any)
	if !ok {
		t.Fatalf("expected auth group, got: %v", record["auth"])
	}
	if auth["authorization"] != "[REDACTED]" {
		t.Fatalf("expected redacted authorization, got: %v", auth["authorization"])
	}
}

func TestRedactionIsKeyBased(t *testing.T) {
	var buf bytes.Buffer
	log, err := New(Options{Level: "INFO", Out: &buf})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	const value = "caller-must-not-log-this"
	log.Info("unrecognized key", slog.String("custom_secret_field", value))

	if !strings.Contains(buf.String(), value) {
		t.Fatalf("expected an unrecognized key to pass through: %s", buf.String())
	}
}
