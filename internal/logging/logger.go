package logging

import "context"

// Level represents logging verbosity.
type Level int

const (
	LevelDebug Level = iota - 4 // align roughly with slog levels
	LevelInfo                   // 0
	LevelWarn                   // +4
	LevelError                  // +8
)

// Logger is a generic logging interface supporting leveled, structured, contextual logs.
// All methods accept context for propagation (trace ids, request ids, etc.).
// Key/value pairs are passed in variadic form: key1, value1, key2, value2, ...
type Logger interface {
	// With returns a child logger with preset fields.
	With(args ...any) Logger

	// WithContext binds default context to the logger; subsequent calls may pass nil ctx.
	WithContext(ctx context.Context) Logger

	// Enabled reports whether the specified level is enabled.
	Enabled(ctx context.Context, level Level) bool

	// Leveled logging with message and key/value pairs.
	Debug(ctx context.Context, msg string, args ...any)
	Info(ctx context.Context, msg string, args ...any)
	Warn(ctx context.Context, msg string, args ...any)
	Error(ctx context.Context, msg string, args ...any)
}
