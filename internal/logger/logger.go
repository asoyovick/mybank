// Package logger provides a reusable structured logging foundation built on
// the standard library's log/slog package.
package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// Options configures the logger returned by New.
type Options struct {
	// Level is one of DEBUG, INFO, WARN or ERROR. An empty value means INFO.
	Level string
	// Out is the destination for log records. A nil value means os.Stdout.
	Out io.Writer
}

// New builds a JSON logger that writes to opts.Out at opts.Level.
// It returns an error rather than a partially configured logger when the
// level is invalid.
func New(opts Options) (*slog.Logger, error) {
	level := slog.LevelInfo
	if opts.Level != "" {
		parsed, err := ParseLevel(opts.Level)
		if err != nil {
			return nil, err
		}
		level = parsed
	}

	out := opts.Out
	if out == nil {
		out = os.Stdout
	}

	return slog.New(slog.NewJSONHandler(out, &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: redactSensitiveAttrs,
	})), nil
}

// ParseLevel converts a case-insensitive level name into a slog.Level.
// Empty input is not a valid level; New treats an empty Options.Level as INFO.
func ParseLevel(raw string) (slog.Level, error) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "DEBUG":
		return slog.LevelDebug, nil
	case "INFO":
		return slog.LevelInfo, nil
	case "WARN", "WARNING":
		return slog.LevelWarn, nil
	case "ERROR":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("logger: unsupported level %q (supported: DEBUG, INFO, WARN, ERROR)", raw)
	}
}

func redactSensitiveAttrs(_ []string, a slog.Attr) slog.Attr {
	switch strings.ToLower(a.Key) {
	case "password", "passwd", "pwd", "db_password", "database_url",
		"secret", "client_secret", "token", "access_token", "refresh_token",
		"dsn", "connection_string", "conn_string", "credentials",
		"api_key", "authorization":
		return slog.String(a.Key, "[REDACTED]")
	default:
		return a
	}
}
