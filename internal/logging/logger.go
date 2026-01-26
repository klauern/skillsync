// Package logging provides structured logging for skillsync using slog.
package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
)

// Level aliases for convenience.
const (
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
)

var (
	defaultLogger *slog.Logger
	defaultOnce   sync.Once
)

// Options configures the logger behavior.
type Options struct {
	// Level sets the minimum log level. Defaults to LevelInfo.
	Level slog.Level
	// Output sets the output destination. Defaults to os.Stderr.
	Output io.Writer
	// JSON enables JSON output format. Defaults to false (text format).
	JSON bool
	// AddSource includes source file and line in log output.
	AddSource bool
}

// DefaultOptions returns options suitable for CLI usage.
func DefaultOptions() Options {
	return Options{
		Level:     LevelInfo,
		Output:    os.Stderr,
		JSON:      false,
		AddSource: false,
	}
}

// New creates a new logger with the given options.
func New(opts Options) *slog.Logger {
	if opts.Output == nil {
		opts.Output = os.Stderr
	}

	handlerOpts := &slog.HandlerOptions{
		Level:     opts.Level,
		AddSource: opts.AddSource,
	}

	var handler slog.Handler
	if opts.JSON {
		handler = slog.NewJSONHandler(opts.Output, handlerOpts)
	} else {
		handler = slog.NewTextHandler(opts.Output, handlerOpts)
	}

	return slog.New(handler)
}

// Default returns the default logger, creating it if necessary.
// The default logger writes text output to stderr at Info level.
func Default() *slog.Logger {
	defaultOnce.Do(func() {
		defaultLogger = New(DefaultOptions())
	})
	return defaultLogger
}

// SetDefault sets the default logger and also sets it as slog's default.
// This also ensures the sync.Once is triggered so Default() won't override the logger.
func SetDefault(logger *slog.Logger) {
	// Trigger the once to prevent Default() from overwriting our logger
	defaultOnce.Do(func() {})
	defaultLogger = logger
	slog.SetDefault(logger)
}

// With returns a logger that includes the given attributes in every output.
func With(args ...any) *slog.Logger {
	if defaultLogger != nil {
		return defaultLogger.With(args...)
	}
	return Default().With(args...)
}

// WithContext returns a logger with context-derived attributes.
func WithContext(ctx context.Context) *slog.Logger {
	if l := FromContext(ctx); l != nil {
		return l
	}
	if defaultLogger != nil {
		return defaultLogger
	}
	return Default()
}

// Debug logs at debug level using the default logger.
func Debug(msg string, args ...any) {
	Default().Debug(msg, args...)
}

// Info logs at info level using the default logger.
func Info(msg string, args ...any) {
	Default().Info(msg, args...)
}

// Warn logs at warn level using the default logger.
func Warn(msg string, args ...any) {
	Default().Warn(msg, args...)
}

// Error logs at error level using the default logger.
func Error(msg string, args ...any) {
	Default().Error(msg, args...)
}

// Context key for logger storage.
type loggerKey struct{}

// NewContext returns a context with the logger attached.
func NewContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// FromContext retrieves the logger from context, or nil if not present.
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
		return l
	}
	return nil
}

// Common attribute keys for consistent logging across the codebase.
const (
	// KeyPlatform identifies the AI platform (claude-code, cursor, codex).
	KeyPlatform = "platform"
	// KeySkill identifies a skill by name.
	KeySkill = "skill"
	// KeyPath identifies a file path.
	KeyPath = "path"
	// KeyOperation identifies the operation being performed.
	KeyOperation = "operation"
	// KeyStrategy identifies the sync strategy.
	KeyStrategy = "strategy"
	// KeyCount provides a count of items.
	KeyCount = "count"
	// KeyError attaches an error value.
	KeyError = "error"
	// KeyDuration records operation duration.
	KeyDuration = "duration"
)

// Platform returns a slog attribute for platform logging.
func Platform(p string) slog.Attr {
	return slog.String(KeyPlatform, p)
}

// Skill returns a slog attribute for skill logging.
func Skill(name string) slog.Attr {
	return slog.String(KeySkill, name)
}

// Path returns a slog attribute for file path logging.
func Path(p string) slog.Attr {
	return slog.String(KeyPath, p)
}

// Operation returns a slog attribute for operation logging.
func Operation(op string) slog.Attr {
	return slog.String(KeyOperation, op)
}

// Err returns a slog attribute for error logging.
func Err(err error) slog.Attr {
	if err == nil {
		return slog.Attr{}
	}
	return slog.Any(KeyError, err)
}

// Count returns a slog attribute for item counts.
func Count(n int) slog.Attr {
	return slog.Int(KeyCount, n)
}
