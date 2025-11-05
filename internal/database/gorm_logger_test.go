package database

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/lyuangg/yuango/internal/logging"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// mockLogger is a mock implementation of logging.Logger for testing.
type mockLogger struct {
	mu          sync.RWMutex
	infoCalled  bool
	warnCalled  bool
	errorCalled bool
	debugCalled bool
	lastMessage string
	lastArgs    []interface{}
	callCount   map[string]int
}

func newMockLogger() *mockLogger {
	return &mockLogger{
		callCount: make(map[string]int),
	}
}

func (m *mockLogger) With(args ...any) logging.Logger {
	return m
}

func (m *mockLogger) WithContext(ctx context.Context) logging.Logger {
	return m
}

func (m *mockLogger) Enabled(ctx context.Context, level logging.Level) bool {
	return true
}

func (m *mockLogger) Info(ctx context.Context, msg string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.infoCalled = true
	m.lastMessage = msg
	m.lastArgs = args
	m.callCount["info"]++
}

func (m *mockLogger) Warn(ctx context.Context, msg string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.warnCalled = true
	m.lastMessage = msg
	m.lastArgs = args
	m.callCount["warn"]++
}

func (m *mockLogger) Error(ctx context.Context, msg string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorCalled = true
	m.lastMessage = msg
	m.lastArgs = args
	m.callCount["error"]++
}

func (m *mockLogger) Debug(ctx context.Context, msg string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.debugCalled = true
	m.lastMessage = msg
	m.lastArgs = args
	m.callCount["debug"]++
}

func (m *mockLogger) reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.infoCalled = false
	m.warnCalled = false
	m.errorCalled = false
	m.debugCalled = false
	m.lastMessage = ""
	m.lastArgs = nil
	m.callCount = make(map[string]int)
}

func (m *mockLogger) getInfoCalled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.infoCalled
}

func (m *mockLogger) getWarnCalled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.warnCalled
}

func (m *mockLogger) getErrorCalled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.errorCalled
}

func (m *mockLogger) getLastMessage() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastMessage
}

func (m *mockLogger) getLastArgs() []interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastArgs
}

// TestParseLogLevel tests the parseLogLevel function.
func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected logger.LogLevel
	}{
		{"silent", "silent", logger.Silent},
		{"error", "error", logger.Error},
		{"warn", "warn", logger.Warn},
		{"warning", "warning", logger.Warn},
		{"info", "info", logger.Info},
		{"empty", "", logger.Info},
		{"invalid", "invalid", logger.Info},
		{"uppercase", "INFO", logger.Info},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseLogLevel(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestNewGormLoggerAdapter tests the NewGormLoggerAdapter function.
func TestNewGormLoggerAdapter(t *testing.T) {
	mockLog := newMockLogger()

	t.Run("creates_adapter_with_info_level", func(t *testing.T) {
		adapter := NewGormLoggerAdapter(mockLog, "info")
		assert.NotNil(t, adapter)
		assert.IsType(t, &GormLoggerAdapter{}, adapter)
	})

	t.Run("creates_adapter_with_silent_level", func(t *testing.T) {
		adapter := NewGormLoggerAdapter(mockLog, "silent")
		assert.NotNil(t, adapter)
	})

	t.Run("creates_adapter_with_default_level", func(t *testing.T) {
		adapter := NewGormLoggerAdapter(mockLog, "")
		assert.NotNil(t, adapter)
	})
}

// TestGormLoggerAdapter_LogMode tests the LogMode method.
func TestGormLoggerAdapter_LogMode(t *testing.T) {
	mockLog := newMockLogger()
	adapter := NewGormLoggerAdapter(mockLog, "info").(*GormLoggerAdapter)

	t.Run("changes_log_level", func(t *testing.T) {
		newAdapter := adapter.LogMode(logger.Warn)
		assert.NotNil(t, newAdapter)
		assert.NotSame(t, adapter, newAdapter) // Should return a new instance

		warnAdapter := newAdapter.(*GormLoggerAdapter)
		assert.Equal(t, logger.Warn, warnAdapter.logLevel)
		assert.Equal(t, logger.Info, adapter.logLevel) // Original should be unchanged
	})
}

// TestGormLoggerAdapter_Info tests the Info method.
func TestGormLoggerAdapter_Info(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		logLevel       string
		expectedCalled bool
	}{
		{"info_level_logs", "info", true},
		{"warn_level_no_logs", "warn", false},   // Warn level doesn't log Info
		{"error_level_no_logs", "error", false}, // Error level doesn't log Info
		{"silent_level_no_logs", "silent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLog := newMockLogger()
			adapter := NewGormLoggerAdapter(mockLog, tt.logLevel)

			adapter.Info(ctx, "test message", "key", "value")

			if tt.expectedCalled {
				assert.True(t, mockLog.getInfoCalled(), "Info should be called")
				assert.Equal(t, "test message", mockLog.getLastMessage())
			} else {
				assert.False(t, mockLog.getInfoCalled(), "Info should not be called")
			}
		})
	}
}

// TestGormLoggerAdapter_Warn tests the Warn method.
func TestGormLoggerAdapter_Warn(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		logLevel       string
		expectedCalled bool
	}{
		{"info_level_logs", "info", true}, // Info level logs Warn
		{"warn_level_logs", "warn", true},
		{"error_level_no_logs", "error", false}, // Error level doesn't log Warn
		{"silent_level_no_logs", "silent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLog := newMockLogger()
			adapter := NewGormLoggerAdapter(mockLog, tt.logLevel)

			adapter.Warn(ctx, "test warning", "key", "value")

			if tt.expectedCalled {
				assert.True(t, mockLog.getWarnCalled(), "Warn should be called")
				assert.Equal(t, "test warning", mockLog.getLastMessage())
			} else {
				assert.False(t, mockLog.getWarnCalled(), "Warn should not be called")
			}
		})
	}
}

// TestGormLoggerAdapter_Error tests the Error method.
func TestGormLoggerAdapter_Error(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		logLevel       string
		expectedCalled bool
	}{
		{"info_level_logs", "info", true}, // Info level logs Error
		{"warn_level_logs", "warn", true}, // Warn level logs Error
		{"error_level_logs", "error", true},
		{"silent_level_no_logs", "silent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLog := newMockLogger()
			adapter := NewGormLoggerAdapter(mockLog, tt.logLevel)

			adapter.Error(ctx, "test error", "key", "value")

			if tt.expectedCalled {
				assert.True(t, mockLog.getErrorCalled(), "Error should be called")
				assert.Equal(t, "test error", mockLog.getLastMessage())
			} else {
				assert.False(t, mockLog.getErrorCalled(), "Error should not be called")
			}
		})
	}
}

// TestGormLoggerAdapter_Trace tests the Trace method.
func TestGormLoggerAdapter_Trace(t *testing.T) {
	ctx := context.Background()

	t.Run("silent_level_no_logs", func(t *testing.T) {
		mockLog := newMockLogger()
		adapter := NewGormLoggerAdapter(mockLog, "silent")

		begin := time.Now()
		fc := func() (string, int64) {
			return "SELECT * FROM users", 1
		}

		adapter.Trace(ctx, begin, fc, nil)

		assert.False(t, mockLog.getInfoCalled())
		assert.False(t, mockLog.getWarnCalled())
		assert.False(t, mockLog.getErrorCalled())
	})

	t.Run("info_level_logs_normal_query", func(t *testing.T) {
		mockLog := newMockLogger()
		adapter := NewGormLoggerAdapter(mockLog, "info")

		begin := time.Now()
		fc := func() (string, int64) {
			return "SELECT * FROM users WHERE id = ?", 1
		}

		adapter.Trace(ctx, begin, fc, nil)

		assert.True(t, mockLog.getInfoCalled())
		assert.Equal(t, "SQL query", mockLog.getLastMessage())

		args := mockLog.getLastArgs()
		assert.Contains(t, args, "sql")
		assert.Contains(t, args, "duration")
		assert.Contains(t, args, "rows")
	})

	t.Run("warn_level_logs_slow_query", func(t *testing.T) {
		mockLog := newMockLogger()
		adapter := NewGormLoggerAdapter(mockLog, "warn")
		adapterImpl := adapter.(*GormLoggerAdapter)
		adapterImpl.slowThreshold = 100 * time.Millisecond

		begin := time.Now().Add(-200 * time.Millisecond) // Query started 200ms ago
		fc := func() (string, int64) {
			return "SELECT * FROM users", 1
		}

		adapter.Trace(ctx, begin, fc, nil)

		assert.True(t, mockLog.getWarnCalled())
		assert.Contains(t, mockLog.getLastMessage(), "slow SQL query")
	})

	t.Run("error_level_logs_sql_error", func(t *testing.T) {
		mockLog := newMockLogger()
		adapter := NewGormLoggerAdapter(mockLog, "error")

		begin := time.Now()
		fc := func() (string, int64) {
			return "SELECT * FROM users", 0
		}
		testErr := errors.New("connection failed")

		adapter.Trace(ctx, begin, fc, testErr)

		assert.True(t, mockLog.getErrorCalled())
		assert.Equal(t, "SQL error", mockLog.getLastMessage())

		args := mockLog.getLastArgs()
		assert.Contains(t, args, "error")
	})

	t.Run("error_level_ignores_record_not_found", func(t *testing.T) {
		mockLog := newMockLogger()
		adapter := NewGormLoggerAdapter(mockLog, "error")

		begin := time.Now()
		fc := func() (string, int64) {
			return "SELECT * FROM users WHERE id = ?", 0
		}

		adapter.Trace(ctx, begin, fc, gorm.ErrRecordNotFound)

		// RecordNotFound error should not be logged at error level
		assert.False(t, mockLog.getErrorCalled())
	})

	t.Run("info_level_logs_record_not_found_as_info", func(t *testing.T) {
		mockLog := newMockLogger()
		adapter := NewGormLoggerAdapter(mockLog, "info")

		begin := time.Now()
		fc := func() (string, int64) {
			return "SELECT * FROM users WHERE id = ?", 0
		}

		adapter.Trace(ctx, begin, fc, gorm.ErrRecordNotFound)

		// RecordNotFound should be logged as info query, not error
		assert.True(t, mockLog.getInfoCalled())
		assert.Equal(t, "SQL query", mockLog.getLastMessage())
		assert.False(t, mockLog.getErrorCalled())
	})

	t.Run("trace_includes_sql_and_timing_info", func(t *testing.T) {
		mockLog := newMockLogger()
		adapter := NewGormLoggerAdapter(mockLog, "info")

		begin := time.Now()
		expectedSQL := "INSERT INTO users (name) VALUES (?)"
		expectedRows := int64(5)

		fc := func() (string, int64) {
			return expectedSQL, expectedRows
		}

		adapter.Trace(ctx, begin, fc, nil)

		args := mockLog.getLastArgs()
		// Check that args contain sql, duration, and rows
		foundSQL := false
		foundRows := false
		foundDuration := false

		for i := 0; i < len(args); i += 2 {
			if i+1 < len(args) {
				key := args[i]
				value := args[i+1]

				if key == "sql" {
					assert.Equal(t, expectedSQL, value)
					foundSQL = true
				}
				if key == "rows" {
					assert.Equal(t, expectedRows, value)
					foundRows = true
				}
				if key == "duration" {
					assert.NotEmpty(t, value)
					foundDuration = true
				}
			}
		}

		assert.True(t, foundSQL, "SQL should be in args")
		assert.True(t, foundRows, "Rows should be in args")
		assert.True(t, foundDuration, "Duration should be in args")
	})

	t.Run("warn_level_no_logs_fast_query", func(t *testing.T) {
		mockLog := newMockLogger()
		adapter := NewGormLoggerAdapter(mockLog, "warn")
		adapterImpl := adapter.(*GormLoggerAdapter)
		adapterImpl.slowThreshold = 100 * time.Millisecond

		begin := time.Now() // Fast query
		fc := func() (string, int64) {
			return "SELECT * FROM users", 1
		}

		adapter.Trace(ctx, begin, fc, nil)

		// Fast query should not be logged at warn level
		assert.False(t, mockLog.getWarnCalled())
		assert.False(t, mockLog.getInfoCalled())
	})

	t.Run("error_level_no_logs_slow_query", func(t *testing.T) {
		mockLog := newMockLogger()
		adapter := NewGormLoggerAdapter(mockLog, "error")
		adapterImpl := adapter.(*GormLoggerAdapter)
		adapterImpl.slowThreshold = 100 * time.Millisecond

		begin := time.Now().Add(-200 * time.Millisecond)
		fc := func() (string, int64) {
			return "SELECT * FROM users", 1
		}

		adapter.Trace(ctx, begin, fc, nil)

		// At error level, slow queries are not logged (only errors are logged)
		assert.False(t, mockLog.getWarnCalled())
		assert.False(t, mockLog.getInfoCalled())
	})
}

// TestGormLoggerAdapter_Integration tests the adapter with real GORM logger interface.
func TestGormLoggerAdapter_Integration(t *testing.T) {
	ctx := context.Background()
	mockLog := newMockLogger()
	adapter := NewGormLoggerAdapter(mockLog, "info")

	// Verify it implements the logger.Interface
	var _ logger.Interface = adapter

	// Test all methods are callable
	adapter.Info(ctx, "info message")
	assert.True(t, mockLog.getInfoCalled())

	mockLog.reset()
	adapter.Warn(ctx, "warn message")
	assert.True(t, mockLog.getWarnCalled())

	mockLog.reset()
	adapter.Error(ctx, "error message")
	assert.True(t, mockLog.getErrorCalled())

	mockLog.reset()
	begin := time.Now()
	fc := func() (string, int64) {
		return "SELECT 1", 1
	}
	adapter.Trace(ctx, begin, fc, nil)
	assert.True(t, mockLog.getInfoCalled())
}

// TestGormLoggerAdapter_LogLevelBoundaries tests edge cases for log levels.
func TestGormLoggerAdapter_LogLevelBoundaries(t *testing.T) {
	ctx := context.Background()

	t.Run("error_level_only_logs_errors", func(t *testing.T) {
		mockLog := newMockLogger()
		adapter := NewGormLoggerAdapter(mockLog, "error")

		adapter.Info(ctx, "info")
		assert.False(t, mockLog.getInfoCalled())

		adapter.Warn(ctx, "warn")
		assert.False(t, mockLog.getWarnCalled())

		adapter.Error(ctx, "error")
		assert.True(t, mockLog.getErrorCalled())
	})

	t.Run("warn_level_logs_warn_and_error", func(t *testing.T) {
		mockLog := newMockLogger()
		adapter := NewGormLoggerAdapter(mockLog, "warn")

		adapter.Info(ctx, "info")
		assert.False(t, mockLog.getInfoCalled())

		mockLog.reset()
		adapter.Warn(ctx, "warn")
		assert.True(t, mockLog.getWarnCalled())

		mockLog.reset()
		adapter.Error(ctx, "error")
		assert.True(t, mockLog.getErrorCalled())
	})
}

// TestGormLoggerAdapter_TraceErrorHandling tests error handling in Trace method.
func TestGormLoggerAdapter_TraceErrorHandling(t *testing.T) {
	ctx := context.Background()

	t.Run("info_level_logs_error_as_error", func(t *testing.T) {
		mockLog := newMockLogger()
		adapter := NewGormLoggerAdapter(mockLog, "info")

		begin := time.Now()
		fc := func() (string, int64) {
			return "SELECT * FROM users", 0
		}
		testErr := errors.New("some error")

		adapter.Trace(ctx, begin, fc, testErr)

		// At info level, errors are logged as error (because info >= error level)
		assert.True(t, mockLog.getErrorCalled())
		assert.Equal(t, "SQL error", mockLog.getLastMessage())
		assert.False(t, mockLog.getInfoCalled())
	})

	t.Run("warn_level_logs_error_as_error", func(t *testing.T) {
		mockLog := newMockLogger()
		adapter := NewGormLoggerAdapter(mockLog, "warn")

		begin := time.Now()
		fc := func() (string, int64) {
			return "SELECT * FROM users", 0
		}
		testErr := errors.New("some error")

		adapter.Trace(ctx, begin, fc, testErr)

		// At warn level, non-RecordNotFound errors are logged as error (because warn >= error level)
		assert.True(t, mockLog.getErrorCalled())
		assert.Equal(t, "SQL error", mockLog.getLastMessage())
		assert.False(t, mockLog.getInfoCalled())
	})
}
