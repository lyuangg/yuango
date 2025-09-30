package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestNewFromConfig_AllConfigs 测试所有可能的配置组合
func TestNewFromConfig_AllConfigs(t *testing.T) {
	tmpDir := t.TempDir()
	testCases := []struct {
		name   string
		config interface{}
	}{
		{
			name: "full config struct",
			config: &struct {
				Level  string
				Format string
				Output string
				Daily  bool
			}{
				Level:  "debug",
				Format: "json",
				Output: filepath.Join(tmpDir, "log1.log"),
				Daily:  true,
			},
		},
		{
			name: "different type fields",
			config: struct {
				Level  int
				Format []byte
				Output bool
				Daily  string
			}{
				Level:  123,
				Format: []byte("text"),
				Output: true,
				Daily:  "true",
			},
		},
		{
			name:   "nil config",
			config: nil,
		},
		{
			name:   "non-struct type",
			config: "invalid",
		},
		{
			name: "empty struct",
			config: struct {
			}{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logger, err := NewFromConfig(tc.config)
			if err != nil {
				t.Logf("Config %s produced error (might be expected): %v", tc.name, err)
			}
			if logger == nil && tc.config != nil {
				t.Errorf("Expected non-nil logger for non-nil config: %s", tc.name)
			}
		})
	}
}

// TestWrite_Comprehensive 全面测试写入功能
func TestWrite_Comprehensive(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "write_test.log")

	writer, err := NewDailyRotateWriter(logPath)
	if err != nil {
		t.Fatal(err)
	}

	testData := []struct {
		name string
		data []byte
		size int
	}{
		{"empty", []byte{}, 0},
		{"small", []byte("small test"), 10},
		{"medium", make([]byte, 1024), 1024},      // 1KB
		{"large", make([]byte, 1024*1024), 1024*1024}, // 1MB
	}

	for _, tc := range testData {
		t.Run(tc.name, func(t *testing.T) {
			n, err := writer.Write(tc.data)
			if err != nil {
				t.Errorf("Write failed for %s: %v", tc.name, err)
			}
			if n != len(tc.data) {
				t.Errorf("Expected %d bytes written, got %d", len(tc.data), n)
			}
		})
	}

	// 测试文件关闭后的写入
	writer.Close()
	_, err = writer.Write([]byte("test"))
	if err == nil {
		t.Error("Expected error when writing to closed file")
	}
}

// TestRotation_Comprehensive 全面测试日志轮转
func TestRotation_Comprehensive(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "rotation_test.log")

	// 创建带最大文件数限制的写入器
	writer, err := NewDailyRotateWriter(logPath, 3)
	if err != nil {
		t.Fatal(err)
	}

	// 创建多个日期的日志文件
	dates := []string{
		time.Now().AddDate(0, 0, -3).Format("2006-01-02"),
		time.Now().AddDate(0, 0, -2).Format("2006-01-02"),
		time.Now().AddDate(0, 0, -1).Format("2006-01-02"),
		time.Now().Format("2006-01-02"),
	}

	for _, date := range dates {
		filename := fmt.Sprintf("%s-%s.log", logPath[:len(logPath)-4], date)
		err := os.WriteFile(filename, []byte("test"), 0644)
		if err != nil {
			t.Fatal(err)
		}

		// 设置文件的修改时间
		modTime := time.Now().AddDate(0, 0, -len(dates))
		err = os.Chtimes(filename, modTime, modTime)
		if err != nil {
			t.Fatal(err)
		}
	}

	// 触发清理
	writer.cleanOldLogFiles()

	// 验证剩余文件数
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	logFiles := 0
	for _, f := range files {
		if !f.IsDir() && filepath.Ext(f.Name()) == ".log" {
			logFiles++
		}
	}

	if logFiles != 3 {
		t.Errorf("Expected 3 log files after cleanup, got %d", logFiles)
	}

	// 测试并发写入
	for i := 0; i < 100; i++ {
		go func(i int) {
			_, err := writer.Write([]byte(fmt.Sprintf("concurrent write %d\n", i)))
			if err != nil {
				t.Errorf("Concurrent write failed: %v", err)
			}
		}(i)
	}

	// 等待所有写入完成
	time.Sleep(100 * time.Millisecond)

	// 关闭写入器
	writer.Close()
}