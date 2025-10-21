package logging

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// TestSlogLogger_WithContextFields tests the WithContextFields functionality.
func TestSlogLogger_WithContextFields(t *testing.T) {
	var buf bytes.Buffer
	logger, err := NewSlogLogger(LevelInfo, "json", &buf, "", 0)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Configure logger to extract custom fields from context
	type contextKey string
	const traceIDKey contextKey = "trace_id"

	logger = logger.WithContextFields(func(ctx context.Context) []any {
		if traceID := ctx.Value(traceIDKey); traceID != nil {
			if id, ok := traceID.(string); ok && id != "" {
				return []any{"trace_id", id}
			}
		}
		return nil
	})

	// Test with trace_id in context
	ctx := context.WithValue(context.Background(), traceIDKey, "test-trace-123")
	logger.Info(ctx, "test message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "test-trace-123") {
		t.Errorf("Log output should contain trace_id, got: %s", output)
	}
	if !strings.Contains(output, "test message") {
		t.Errorf("Log output should contain message, got: %s", output)
	}
	if !strings.Contains(output, "\"key\":\"value\"") {
		t.Errorf("Log output should contain key-value pair, got: %s", output)
	}
}

// TestSlogLogger_WithContextFields_NoContext tests logging without context fields.
func TestSlogLogger_WithContextFields_NoContext(t *testing.T) {
	var buf bytes.Buffer
	logger, err := NewSlogLogger(LevelInfo, "json", &buf, "", 0)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Configure logger but don't provide context with fields
	type contextKey string
	const traceIDKey contextKey = "trace_id"

	logger = logger.WithContextFields(func(ctx context.Context) []any {
		if traceID := ctx.Value(traceIDKey); traceID != nil {
			if id, ok := traceID.(string); ok && id != "" {
				return []any{"trace_id", id}
			}
		}
		return nil
	})

	// Log without trace_id in context
	logger.Info(context.Background(), "test message")

	output := buf.String()
	if strings.Contains(output, "trace_id") {
		t.Errorf("Log output should not contain trace_id when not in context, got: %s", output)
	}
	if !strings.Contains(output, "test message") {
		t.Errorf("Log output should contain message, got: %s", output)
	}
}

// TestSlogLogger_WithContextFields_MultipleFields tests extracting multiple fields.
func TestSlogLogger_WithContextFields_MultipleFields(t *testing.T) {
	var buf bytes.Buffer
	logger, err := NewSlogLogger(LevelInfo, "json", &buf, "", 0)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Configure logger to extract multiple fields
	type contextKey string
	const (
		traceIDKey contextKey = "trace_id"
		userIDKey  contextKey = "user_id"
	)

	logger = logger.WithContextFields(func(ctx context.Context) []any {
		var fields []any

		if traceID := ctx.Value(traceIDKey); traceID != nil {
			if id, ok := traceID.(string); ok && id != "" {
				fields = append(fields, "trace_id", id)
			}
		}

		if userID := ctx.Value(userIDKey); userID != nil {
			if id, ok := userID.(int); ok && id > 0 {
				fields = append(fields, "user_id", id)
			}
		}

		return fields
	})

	// Test with multiple fields in context
	ctx := context.Background()
	ctx = context.WithValue(ctx, traceIDKey, "trace-456")
	ctx = context.WithValue(ctx, userIDKey, 789)

	logger.Info(ctx, "user action")

	output := buf.String()
	if !strings.Contains(output, "trace-456") {
		t.Errorf("Log output should contain trace_id, got: %s", output)
	}
	if !strings.Contains(output, "789") {
		t.Errorf("Log output should contain user_id, got: %s", output)
	}
	if !strings.Contains(output, "user action") {
		t.Errorf("Log output should contain message, got: %s", output)
	}
}

// TestSlogLogger_WithContextFields_NoFunction tests logging without context fields function.
func TestSlogLogger_WithContextFields_NoFunction(t *testing.T) {
	var buf bytes.Buffer
	logger, err := NewSlogLogger(LevelInfo, "json", &buf, "", 0)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Don't configure any context fields function
	type contextKey string
	const traceIDKey contextKey = "trace_id"
	ctx := context.WithValue(context.Background(), traceIDKey, "should-not-appear")

	logger.Info(ctx, "test message")

	output := buf.String()
	if strings.Contains(output, "should-not-appear") {
		t.Errorf("Log output should not contain trace_id when no extractor is configured, got: %s", output)
	}
	if !strings.Contains(output, "test message") {
		t.Errorf("Log output should contain message, got: %s", output)
	}
}

// TestSlogLogger_WithContextFields_AllLogLevels tests all log levels.
func TestSlogLogger_WithContextFields_AllLogLevels(t *testing.T) {
	type contextKey string
	const traceIDKey contextKey = "trace_id"

	tests := []struct {
		name     string
		logLevel Level
		logFunc  func(Logger, context.Context, string)
	}{
		{
			name:     "Debug",
			logLevel: LevelDebug,
			logFunc: func(l Logger, ctx context.Context, msg string) {
				l.Debug(ctx, msg)
			},
		},
		{
			name:     "Info",
			logLevel: LevelInfo,
			logFunc: func(l Logger, ctx context.Context, msg string) {
				l.Info(ctx, msg)
			},
		},
		{
			name:     "Warn",
			logLevel: LevelWarn,
			logFunc: func(l Logger, ctx context.Context, msg string) {
				l.Warn(ctx, msg)
			},
		},
		{
			name:     "Error",
			logLevel: LevelError,
			logFunc: func(l Logger, ctx context.Context, msg string) {
				l.Error(ctx, msg)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger, err := NewSlogLogger(LevelDebug, "json", &buf, "", 0)
			if err != nil {
				t.Fatalf("Failed to create logger: %v", err)
			}

			logger = logger.WithContextFields(func(ctx context.Context) []any {
				if traceID := ctx.Value(traceIDKey); traceID != nil {
					if id, ok := traceID.(string); ok && id != "" {
						return []any{"trace_id", id}
					}
				}
				return nil
			})

			ctx := context.WithValue(context.Background(), traceIDKey, "trace-"+tt.name)
			tt.logFunc(logger, ctx, "test message")

			output := buf.String()
			if !strings.Contains(output, "trace-"+tt.name) {
				t.Errorf("Log output should contain trace_id for %s, got: %s", tt.name, output)
			}
		})
	}
}

// TestSlogLogger_WithContextFields_Propagation tests that context fields are propagated through With and WithContext.
func TestSlogLogger_WithContextFields_Propagation(t *testing.T) {
	var buf bytes.Buffer
	logger, err := NewSlogLogger(LevelInfo, "json", &buf, "", 0)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	type contextKey string
	const traceIDKey contextKey = "trace_id"

	// Configure context fields on the original logger
	logger = logger.WithContextFields(func(ctx context.Context) []any {
		if traceID := ctx.Value(traceIDKey); traceID != nil {
			if id, ok := traceID.(string); ok && id != "" {
				return []any{"trace_id", id}
			}
		}
		return nil
	})

	// Test With() propagation
	childLogger := logger.With("service", "test-service")
	ctx := context.WithValue(context.Background(), traceIDKey, "trace-with")
	childLogger.Info(ctx, "test with")

	output := buf.String()
	if !strings.Contains(output, "trace-with") {
		t.Errorf("Child logger (With) should extract trace_id, got: %s", output)
	}
	if !strings.Contains(output, "test-service") {
		t.Errorf("Child logger (With) should have preset field, got: %s", output)
	}

	// Test WithContext() propagation
	buf.Reset()
	ctx2 := context.WithValue(context.Background(), traceIDKey, "trace-context")
	contextLogger := logger.WithContext(ctx2)
	// When logging with the same context that was bound, the context fields should be extracted
	contextLogger.Info(ctx2, "test context")

	output = buf.String()
	if !strings.Contains(output, "trace-context") {
		t.Errorf("Child logger (WithContext) should extract trace_id, got: %s", output)
	}
}
