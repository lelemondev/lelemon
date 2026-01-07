package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"time"
)

// Config holds logger configuration
type Config struct {
	Level  string // "debug", "info", "warn", "error"
	Format string // "json", "text"
	Output io.Writer
}

// DefaultConfig returns a production-ready configuration
func DefaultConfig() Config {
	return Config{
		Level:  "info",
		Format: "json",
		Output: os.Stdout,
	}
}

// New creates a new structured logger
func New(cfg Config) *slog.Logger {
	level := parseLevel(cfg.Level)

	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: level == slog.LevelDebug,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Customize time format
			if a.Key == slog.TimeKey {
				if t, ok := a.Value.Any().(time.Time); ok {
					a.Value = slog.StringValue(t.Format(time.RFC3339))
				}
			}
			return a
		},
	}

	output := cfg.Output
	if output == nil {
		output = os.Stdout
	}

	switch cfg.Format {
	case "text":
		handler = slog.NewTextHandler(output, opts)
	default:
		handler = slog.NewJSONHandler(output, opts)
	}

	return slog.New(handler)
}

// Setup configures the default logger
func Setup(cfg Config) {
	logger := New(cfg)
	slog.SetDefault(logger)
}

func parseLevel(level string) slog.Level {
	switch level {
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

// Context keys for request-scoped logging
type ctxKey string

const (
	requestIDKey ctxKey = "request_id"
	projectIDKey ctxKey = "project_id"
	userIDKey    ctxKey = "user_id"
)

// WithRequestID adds a request ID to the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// WithProjectID adds a project ID to the context
func WithProjectID(ctx context.Context, projectID string) context.Context {
	return context.WithValue(ctx, projectIDKey, projectID)
}

// WithUserID adds a user ID to the context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// FromContext returns a logger with context values as attributes
func FromContext(ctx context.Context) *slog.Logger {
	logger := slog.Default()

	if requestID, ok := ctx.Value(requestIDKey).(string); ok && requestID != "" {
		logger = logger.With("request_id", requestID)
	}
	if projectID, ok := ctx.Value(projectIDKey).(string); ok && projectID != "" {
		logger = logger.With("project_id", projectID)
	}
	if userID, ok := ctx.Value(userIDKey).(string); ok && userID != "" {
		logger = logger.With("user_id", userID)
	}

	return logger
}

// Helper functions for common log patterns

// LogRequest logs an incoming HTTP request
func LogRequest(ctx context.Context, method, path string, statusCode int, duration time.Duration) {
	FromContext(ctx).Info("http_request",
		"method", method,
		"path", path,
		"status", statusCode,
		"duration_ms", duration.Milliseconds(),
	)
}

// LogError logs an error with context
func LogError(ctx context.Context, msg string, err error, attrs ...any) {
	args := append([]any{"error", err.Error()}, attrs...)
	FromContext(ctx).Error(msg, args...)
}

// LogIngest logs ingestion events
func LogIngest(ctx context.Context, eventCount int, duration time.Duration) {
	FromContext(ctx).Info("ingest",
		"events", eventCount,
		"duration_ms", duration.Milliseconds(),
	)
}

// LogDBQuery logs database queries (only in debug mode)
func LogDBQuery(ctx context.Context, query string, duration time.Duration) {
	FromContext(ctx).Debug("db_query",
		"query", query,
		"duration_ms", duration.Milliseconds(),
	)
}
