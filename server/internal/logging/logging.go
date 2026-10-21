// Package logging configures structured server logging and privacy-safe token
// redaction.
package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// New returns a JSON logger writing to stdout and, optionally, a log file.
// The returned closer must be closed on shutdown.
func New(path, level string) (*slog.Logger, io.Closer, error) {
	var writers []io.Writer
	writers = append(writers, os.Stdout)
	var closer io.Closer = nopCloser{}
	if strings.TrimSpace(path) != "" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
			return nil, nil, fmt.Errorf("create log directory: %w", err)
		}
		file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, nil, fmt.Errorf("open log file: %w", err)
		}
		writers = append(writers, file)
		closer = file
	}
	handler := slog.NewJSONHandler(io.MultiWriter(writers...), &slog.HandlerOptions{
		Level: parseLevel(level),
	})
	return slog.New(handler), closer, nil
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// RedactToken returns a short, non-reversible diagnostic prefix for a device
// or game token. Full tokens must never be logged.
func RedactToken(token string) string {
	if token == "" {
		return ""
	}
	if len(token) <= 8 {
		return "***"
	}
	return token[:4] + "..." + token[len(token)-2:]
}

type nopCloser struct{}

func (nopCloser) Close() error { return nil }
