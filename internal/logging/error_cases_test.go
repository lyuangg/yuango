package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestRotationWithErrors tests error cases for file rotation
func TestRotationWithErrors(t *testing.T) {
	// 准备临时目录
	tmpDir, err := os.MkdirTemp("", "log_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "app.log")

	// 创建一个ReadOnly目录来触发写入权限错误
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.Mkdir(readOnlyDir, 0o400); err != nil {
		t.Fatal(err)
	}
	readOnlyLog := filepath.Join(readOnlyDir, "app.log")

	tests := []struct {
		name        string
		path        string
		maxFiles    int
		wantError   bool
		setupFunc   func() error
		cleanupFunc func()
	}{
		{
			name:      "rotation_with_invalid_path",
			path:      "/non/existent/path/app.log",
			maxFiles:  1,
			wantError: true,
		},
		{
			name:      "rotation_with_readonly_dir",
			path:      readOnlyLog,
			maxFiles:  1,
			wantError: true,
		},
		{
			name:     "rotation_with_readonly_file",
			path:     logPath,
			maxFiles: 1,
			setupFunc: func() error {
				// First create a file
				if err := os.WriteFile(logPath, []byte("test"), 0o644); err != nil {
					return err
				}
				// Make parent directory readonly
				if err := os.Chmod(filepath.Dir(logPath), 0o500); err != nil {
					return err
				}
				return nil
			},
			wantError: true,
			cleanupFunc: func() {
				// Make parent directory writable again
				os.Chmod(filepath.Dir(logPath), 0o755)
				os.Remove(logPath)
			},
		},
		{
			name:     "rotation_with_maxfiles_error",
			path:     logPath,
			maxFiles: 2,
			setupFunc: func() error {
				// 创建几个日志文件
				for i := 0; i < 3; i++ {
					date := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
					logFile := fmt.Sprintf("%s-%s.log", logPath[:len(logPath)-4], date)
					if err := os.WriteFile(logFile, []byte("test"), 0o644); err != nil {
						return err
					}
					// 设置最后修改时间以便排序
					modTime := time.Now().AddDate(0, 0, -i)
					if err := os.Chtimes(logFile, modTime, modTime); err != nil {
						return err
					}
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				if err := tt.setupFunc(); err != nil {
					t.Fatal(err)
				}
			}

			if tt.cleanupFunc != nil {
				defer tt.cleanupFunc()
			}

			// 创建DailyRotateWriter
			writer, err := NewDailyRotateWriter(tt.path, tt.maxFiles)
			if tt.wantError {
				if err == nil {
					t.Error("Expected error when creating writer, but got nil")
					return
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			// 写入一些数据
			_, err = writer.Write([]byte("test"))
			if err != nil && !tt.wantError {
				t.Fatal(err)
			}

			// 关闭
			err = writer.Close()
			if err != nil && !tt.wantError {
				t.Fatal(err)
			}
		})
	}
}

// TestAutoRotateEdgeCases tests edge cases for auto rotation
func TestAutoRotateEdgeCases(t *testing.T) {
	// 准备临时目录
	tmpDir, err := os.MkdirTemp("", "log_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "app.log")

	tests := []struct {
		name     string
		maxFiles int
		setup    func(w *DailyRotateWriter)
	}{
		{
			name:     "quick_stop",
			maxFiles: 1,
			setup: func(w *DailyRotateWriter) {
				// 立即停止自动轮转
				w.Close()
			},
		},
		{
			name:     "multiple_rotations",
			maxFiles: 1,
			setup: func(w *DailyRotateWriter) {
				// 快速触发多次轮转
				for i := 0; i < 3; i++ {
					w.rotateIfNeeded()
					time.Sleep(10 * time.Millisecond)
				}
			},
		},
		{
			name:     "rotation_during_writes",
			maxFiles: 1,
			setup: func(w *DailyRotateWriter) {
				// 在写入时触发轮转
				go func() {
					for i := 0; i < 100; i++ {
						w.Write([]byte("test\n"))
						time.Sleep(time.Millisecond)
					}
				}()
				time.Sleep(10 * time.Millisecond)
				w.rotateIfNeeded()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建DailyRotateWriter
			writer, err := NewDailyRotateWriter(logPath, tt.maxFiles)
			if err != nil {
				t.Fatal(err)
			}

			// 运行测试特定的设置
			if tt.setup != nil {
				tt.setup(writer)
			}

			// 等待一小段时间以让自动轮转有机会运行
			time.Sleep(50 * time.Millisecond)

			// 关闭
			writer.Close()
		})
	}
}

// TestCleanOldLogFilesEdgeCases tests edge cases for cleaning old log files
func TestCleanOldLogFilesEdgeCases(t *testing.T) {
	// 准备临时目录
	tmpDir, err := os.MkdirTemp("", "log_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "app.log")

	tests := []struct {
		name     string
		maxFiles int
		setup    func() error
	}{
		{
			name:     "no_old_files",
			maxFiles: 5,
			setup: func() error {
				return nil // 不创建任何文件
			},
		},
		{
			name:     "exactly_max_files",
			maxFiles: 3,
			setup: func() error {
				// 创建正好maxFiles个文件
				for i := 0; i < 3; i++ {
					date := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
					logFile := fmt.Sprintf("%s-%s.log", logPath[:len(logPath)-4], date)
					if err := os.WriteFile(logFile, []byte("test"), 0o644); err != nil {
						return err
					}
					// 设置不同的修改时间以便排序
					modTime := time.Now().AddDate(0, 0, -i)
					if err := os.Chtimes(logFile, modTime, modTime); err != nil {
						return err
					}
				}
				return nil
			},
		},
		{
			name:     "mixed_file_types",
			maxFiles: 2,
			setup: func() error {
				// 创建一些日志文件和非日志文件
				files := []struct {
					name    string
					content string
				}{
					{"app-2023-01-01.log", "log1"},
					{"app-2023-01-02.log", "log2"},
					{"app-2023-01-03.txt", "not a log"}, // 不同扩展名
					{"other-2023-01-01.log", "other"},   // 不同前缀
				}

				for _, f := range files {
					path := filepath.Join(tmpDir, f.name)
					if err := os.WriteFile(path, []byte(f.content), 0o644); err != nil {
						return err
					}
				}
				return nil
			},
		},
		{
			name:     "unreadable_files",
			maxFiles: 2,
			setup: func() error {
				// 创建两个旧日志文件
				for i := 0; i < 2; i++ {
					date := time.Now().AddDate(0, 0, -i-1).Format("2006-01-02")
					logFile := fmt.Sprintf("%s-%s.log", logPath[:len(logPath)-4], date)
					if err := os.WriteFile(logFile, []byte("test"), 0o644); err != nil {
						return err
					}
					// 设置不同的修改时间以便排序
					modTime := time.Now().AddDate(0, 0, -i-1)
					if err := os.Chtimes(logFile, modTime, modTime); err != nil {
						return err
					}
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				if err := tt.setup(); err != nil {
					t.Fatal(err)
				}
			}

			// 创建DailyRotateWriter
			writer, err := NewDailyRotateWriter(logPath, tt.maxFiles)
			if err != nil {
				t.Fatal(err)
			}

			if _, err := writer.Write([]byte("new test data\n")); err != nil {
				t.Fatal(err)
			}

			// 直接调用cleanOldLogFiles
			writer.cleanOldLogFiles()

			// 验证文件数量
			files, err := os.ReadDir(tmpDir)
			if err != nil {
				t.Fatal(err)
			}

			var logFiles int
			for _, f := range files {
				name := f.Name()
				if !f.IsDir() && filepath.Ext(name) == ".log" {
					if name == filepath.Base(logPath) || strings.HasPrefix(name, filepath.Base(logPath[:len(logPath)-4])) {
						logFiles++
					}
				}
			}

			// 验证剩余的日志文件数量不超过maxFiles+1(包括当前日志文件)
			if logFiles > tt.maxFiles+1 {
				t.Errorf("Expected at most %d log files, but found %d", tt.maxFiles+1, logFiles)
			}

			writer.Close()
		})
	}
}

// TestCloseWithConcurrentWrite tests closing a writer while concurrently writing
func TestCloseWithConcurrentWrite(t *testing.T) {
	// 准备临时目录
	tmpDir, err := os.MkdirTemp("", "log_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "app.log")

	// 创建目录
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		t.Fatal(err)
	}

	// 创建DailyRotateWriter
	writer, err := NewDailyRotateWriter(logPath, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()

	// 写入初始数据
	testData := "initial data\n"
	if _, err := writer.Write([]byte(testData)); err != nil {
		t.Fatal(err)
	}

	// 确保数据写入到了正确的文件
	today := time.Now().Format("2006-01-02")
	actualLogPath := fmt.Sprintf("%s-%s.log", logPath[:len(logPath)-4], today)

	content, err := os.ReadFile(actualLogPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	if string(content) != testData {
		t.Fatalf("Expected %q, got %q", testData, string(content))
	}

	// 启动并发写入
	const writeCount = 10
	var wg sync.WaitGroup
	errCh := make(chan error, writeCount)
	doneCh := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < writeCount; i++ {
			select {
			case <-doneCh:
				return
			default:
				data := fmt.Sprintf("concurrent write %d\n", i)
				_, err := writer.Write([]byte(data))
				if err != nil && !strings.Contains(err.Error(), "file already closed") {
					errCh <- err
					return
				}
				time.Sleep(time.Millisecond) // 给其他goroutine一些执行机会
			}
		}
	}()

	// 让写入运行一段时间
	time.Sleep(50 * time.Millisecond)

	// 关闭writer并等待goroutine完成
	close(doneCh)
	if err := writer.Close(); err != nil {
		t.Fatalf("Failed to close writer: %v", err)
	}
	wg.Wait()

	// 检查写入错误
	select {
	case err := <-errCh:
		t.Fatalf("Unexpected write error: %v", err)
	default:
		// 没有错误是好的
	}

	// 最终验证文件
	content, err = os.ReadFile(actualLogPath)
	if err != nil {
		t.Fatalf("Failed to read final log file: %v", err)
	}
	if !strings.Contains(string(content), "initial data") {
		t.Error("Log file missing initial data")
	}
	if len(content) <= len(testData) {
		t.Error("Log file does not contain any concurrent write data")
	}
}
