package logging

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/lyuangg/yuango/internal/config"
)

// TestNewFromConfig tests the NewFromConfig factory function.
func TestNewFromConfig(t *testing.T) {
	tempDir := t.TempDir()

	testCases := []struct {
		name    string
		config  config.LogConfig
		wantErr bool
	}{
		{
			name: "default config",
			config: config.LogConfig{
				Level:  "info",
				Format: "text",
				Output: "stdout",
				Daily:  false,
			},
			wantErr: false,
		},
		{
			name: "json format",
			config: config.LogConfig{
				Level:  "debug",
				Format: "json",
				Output: "stderr",
				Daily:  false,
			},
			wantErr: false,
		},
		{
			name: "daily rotation enabled",
			config: config.LogConfig{
				Level:  "warn",
				Format: "text",
				Output: filepath.Join(tempDir, "daily.log"),
				Daily:  true,
			},
			wantErr: false,
		},
		{
			name: "error log level",
			config: config.LogConfig{
				Level:  "error",
				Format: "text",
				Output: filepath.Join(tempDir, "error.log"),
				Daily:  false,
			},
			wantErr: false,
		},
		{
			name: "invalid log level",
			config: config.LogConfig{
				Level:  "invalid",
				Format: "text",
				Output: "stdout",
				Daily:  false,
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logger, err := NewFromConfig(tc.config)
			if tc.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if logger == nil {
				t.Error("Expected logger but got nil")
				return
			}

			// Test basic functionality
			ctx := context.Background()
			if tc.config.Output != "stdout" && tc.config.Output != "stderr" {
				logger.Info(ctx, "factory test message", "test", tc.name)
			}
		})
	}
}

// TestNewFromConfig_CustomConfigs tests NewFromConfig with various custom config types
func TestNewFromConfig_NativeConfig(t *testing.T) {
	testCases := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "native config with all fields",
			config: &Config{
				Level:    "debug",
				Format:   "json",
				Output:   "stdout",
				Daily:    true,
				MaxFiles: 5,
			},
			wantErr: false,
		},
		{
			name:    "native config with empty fields",
			config:  &Config{},
			wantErr: false,
		},
		{
			name: "native config with invalid level",
			config: &Config{
				Level: "invalid",
			},
			wantErr: true,
		},
		{
			name: "native config with maxfiles",
			config: &Config{
				Level:    "info",
				Format:   "text",
				Output:   "stderr",
				MaxFiles: 10,
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logger, err := NewFromConfig(tc.config)
			if tc.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if logger == nil {
				t.Error("Expected non-nil logger")
				return
			}

			// 测试基本功能
			ctx := context.Background()
			logger.Info(ctx, "test message from native config")
		})
	}
}

func TestNewFromConfig_CustomConfigs(t *testing.T) {
	tempDir := t.TempDir()

	// 测试用例：Config 结构体
	type CustomConfig1 struct {
		Level  string
		Format string
		Output string
		Daily  bool
	}

	// 测试用例：不同类型的字段
	type CustomConfig2 struct {
		Level  interface{} // 测试接口类型
		Format *string     // 测试指针类型
		Output []byte      // 测试切片类型
		Daily  *bool       // 测试布尔指针
	}

	// 测试用例：带有额外字段的配置
	type CustomConfig3 struct {
		Level    string
		Format   string
		Output   string
		Daily    bool
		MaxFiles int    // 额外字段
		LogID    string // 额外字段
	}

	// 测试用例：带有未导出字段的配置
	type CustomConfig4 struct {
		level  string // 未导出字段
		format string // 未导出字段
		Level  string // 导出字段
		Format string // 导出字段
	}

	// 准备测试数据
	textFormat := "text"
	jsonFormat := "json"
	isDaily := true
	debugLevel := "debug"
	outputPath := filepath.Join(tempDir, "test.log")

	testCases := []struct {
		name    string
		config  interface{}
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: false,
		},
		{
			name: "basic custom config",
			config: &CustomConfig1{
				Level:  "debug",
				Format: "json",
				Output: outputPath,
				Daily:  true,
			},
			wantErr: false,
		},
		{
			name: "config with interface and pointer fields",
			config: &CustomConfig2{
				Level:  debugLevel,
				Format: &textFormat,
				Output: []byte("stdout"),
				Daily:  &isDaily,
			},
			wantErr: false,
		},
		{
			name: "config with extra fields",
			config: &CustomConfig3{
				Level:    "info",
				Format:   "text",
				Output:   "stderr",
				Daily:    false,
				MaxFiles: 10,
				LogID:    "test-log",
			},
			wantErr: false,
		},
		{
			name: "config with unexported fields",
			config: &CustomConfig4{
				level:  "debug", // 未导出字段
				format: "json",  // 未导出字段
				Level:  "info",  // 导出字段
				Format: "text",  // 导出字段
			},
			wantErr: false,
		},
		{
			name: "string map config",
			config: map[string]interface{}{
				"Level":  "debug",
				"Format": "json",
				"Output": "stdout",
				"Daily":  true,
			},
			wantErr: false,
		},
		{
			name:    "empty struct config",
			config:  struct{}{},
			wantErr: false,
		},
		{
			name:    "invalid type config",
			config:  42,
			wantErr: false,
		},
		{
			name: "pointer to pointer config",
			config: func() interface{} {
				cfg := &CustomConfig1{
					Level:  "debug",
					Format: "json",
					Output: "stdout",
				}
				return &cfg
			}(),
			wantErr: false,
		},
		{
			name: "config with all pointer fields",
			config: func() interface{} {
				return &struct {
					Level  *string
					Format *string
					Output *string
					Daily  *bool
				}{
					Level:  &debugLevel,
					Format: &jsonFormat,
					Output: &outputPath,
					Daily:  &isDaily,
				}
			}(),
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logger, err := NewFromConfig(tc.config)
			if tc.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// 对于非 nil 的配置，logger 不应该为 nil
			if tc.config != nil && logger == nil {
				t.Error("Expected non-nil logger for non-nil config")
				return
			}

			// 如果成功创建了 logger，测试基本功能
			if logger != nil {
				ctx := context.Background()
				logger.Info(ctx, "test message", "test_case", tc.name)
			}
		})
	}
}

// TestParseLevel tests the parseLevel function.
func TestParseLevel(t *testing.T) {
	testCases := []struct {
		input   string
		want    Level
		wantErr bool
	}{
		{"debug", LevelDebug, false},
		{"DEBUG", LevelDebug, false},
		{"  debug  ", LevelDebug, false},
		{"info", LevelInfo, false},
		{"INFO", LevelInfo, false},
		{"", LevelInfo, false},
		{"warn", LevelWarn, false},
		{"warning", LevelWarn, false},
		{"WARN", LevelWarn, false},
		{"error", LevelError, false},
		{"err", LevelError, false},
		{"ERROR", LevelError, false},
		{"invalid", LevelInfo, true},
		{"critical", LevelInfo, true},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result, err := parseLevel(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Errorf("Expected error for input %q but got none", tc.input)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for input %q: %v", tc.input, err)
				return
			}

			if result != tc.want {
				t.Errorf("parseLevel(%q) = %v, want %v", tc.input, result, tc.want)
			}
		})
	}
}

// TestNewFromConfigEdgeCases tests edge cases for NewFromConfig.
func TestNewFromConfigEdgeCases(t *testing.T) {
	tempDir := t.TempDir()

	testCases := []struct {
		name   string
		config config.LogConfig
		test   func(t *testing.T, logger Logger)
	}{
		{
			name: "stdout output",
			config: config.LogConfig{
				Level:  "info",
				Format: "text",
				Output: "stdout",
				Daily:  false,
			},
			test: func(t *testing.T, logger Logger) {
				ctx := context.Background()
				logger.Info(ctx, "stdout test")
			},
		},
		{
			name: "stderr output",
			config: config.LogConfig{
				Level:  "info",
				Format: "text",
				Output: "stderr",
				Daily:  false,
			},
			test: func(t *testing.T, logger Logger) {
				ctx := context.Background()
				logger.Info(ctx, "stderr test")
			},
		},
		{
			name: "empty output defaults to stdout",
			config: config.LogConfig{
				Level:  "info",
				Format: "text",
				Output: "",
				Daily:  false,
			},
			test: func(t *testing.T, logger Logger) {
				ctx := context.Background()
				logger.Info(ctx, "empty output test")
			},
		},
		{
			name: "json format",
			config: config.LogConfig{
				Level:  "info",
				Format: "json",
				Output: filepath.Join(tempDir, "json.log"),
				Daily:  false,
			},
			test: func(t *testing.T, logger Logger) {
				ctx := context.Background()
				logger.Info(ctx, "json test", "key", "value")
			},
		},
		{
			name: "unknown format defaults to text",
			config: config.LogConfig{
				Level:  "info",
				Format: "unknown",
				Output: filepath.Join(tempDir, "text.log"),
				Daily:  false,
			},
			test: func(t *testing.T, logger Logger) {
				ctx := context.Background()
				logger.Info(ctx, "text format test")
			},
		},
		{
			name: "daily rotation",
			config: config.LogConfig{
				Level:  "info",
				Format: "text",
				Output: filepath.Join(tempDir, "daily.log"),
				Daily:  true,
			},
			test: func(t *testing.T, logger Logger) {
				ctx := context.Background()
				logger.Info(ctx, "daily rotation test")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logger, err := NewFromConfig(tc.config)
			if err != nil {
				t.Fatalf("NewFromConfig failed: %v", err)
			}

			if logger == nil {
				t.Fatal("NewFromConfig returned nil logger")
			}

			tc.test(t, logger)
		})
	}
}

// TestNewFromConfigAllLevels tests all log levels with NewFromConfig.
func TestNewFromConfigAllLevels(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "levels.log")

	levels := []struct {
		name  string
		level string
		want  Level
	}{
		{"debug", "debug", LevelDebug},
		{"info", "info", LevelInfo},
		{"warn", "warn", LevelWarn},
		{"error", "error", LevelError},
	}

	for _, tc := range levels {
		t.Run(tc.name, func(t *testing.T) {
			config := config.LogConfig{
				Level:  tc.level,
				Format: "text",
				Output: logPath,
				Daily:  false,
			}

			logger, err := NewFromConfig(config)
			if err != nil {
				t.Fatalf("NewFromConfig failed: %v", err)
			}

			ctx := context.Background()

			// Test all log methods
			logger.Debug(ctx, "debug message")
			logger.Info(ctx, "info message")
			logger.Warn(ctx, "warn message")
			logger.Error(ctx, "error message")

			// Test Enabled method
			if logger.Enabled(ctx, tc.want) {
				t.Logf("Level %s is enabled", tc.level)
			}

			// Test higher levels should be enabled for lower thresholds
			if tc.level == "debug" {
				if !logger.Enabled(ctx, LevelDebug) {
					t.Error("Debug should be enabled when level is debug")
				}
			}
		})
	}
}
