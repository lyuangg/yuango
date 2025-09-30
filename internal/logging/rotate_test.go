package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestDailyRotateWriter_Close_Comprehensive 全面测试关闭功能
func TestDailyRotateWriter_Close_Comprehensive(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("close sequence", func(t *testing.T) {
		logPath := filepath.Join(tmpDir, "close_sequence.log")
		writer, err := NewDailyRotateWriter(logPath)
		if err != nil {
			t.Fatal(err)
		}

		// 写入一些数据
		_, err = writer.Write([]byte("test data\n"))
		if err != nil {
			t.Fatal(err)
		}

		// 第一次关闭
		if err := writer.Close(); err != nil {
			t.Errorf("First close failed: %v", err)
		}

		// 第二次关闭 - 在文件已经关闭的情况下会返回错误，这是预期的行为
		err = writer.Close()
		if err == nil {
			t.Error("Expected error on second close")
		}
	})

	t.Run("concurrent close", func(t *testing.T) {
		logPath := filepath.Join(tmpDir, "concurrent_close.log")
		writer, err := NewDailyRotateWriter(logPath)
		if err != nil {
			t.Fatal(err)
		}

		// 并发关闭
		var wg sync.WaitGroup
		closeCount := 10
		wg.Add(closeCount)

		for range closeCount {
			go func() {
				defer wg.Done()
				writer.Close()
			}()
		}

		wg.Wait()
	})

	t.Run("close during write", func(t *testing.T) {
		logPath := filepath.Join(tmpDir, "close_write.log")
		writer, err := NewDailyRotateWriter(logPath)
		if err != nil {
			t.Fatal(err)
		}

		// 启动写入 goroutine
		done := make(chan struct{})
		go func() {
			defer close(done)
			for i := 0; i < 1000; i++ {
				writer.Write([]byte("test data\n"))
				time.Sleep(time.Microsecond)
			}
		}()

		// 等待一小段时间后关闭
		time.Sleep(time.Millisecond)
		writer.Close()

		<-done // 等待写入完成
	})

	t.Run("close during rotation", func(t *testing.T) {
		logPath := filepath.Join(tmpDir, "close_rotate.log")
		writer, err := NewDailyRotateWriter(logPath)
		if err != nil {
			t.Fatal(err)
		}

		// 写入一些数据
		_, err = writer.Write([]byte("initial data\n"))
		if err != nil {
			t.Fatal(err)
		}

		// 启动 rotation
		rotateDone := make(chan struct{})
		go func() {
			defer close(rotateDone)
			for i := 0; i < 5; i++ {
				writer.rotateIfNeeded()
				time.Sleep(time.Millisecond)
			}
		}()

		// 等待一小段时间后关闭
		time.Sleep(time.Millisecond)
		writer.Close()

		<-rotateDone // 等待轮转完成
	})
}

// TestAutoRotate_Advanced 全面测试自动轮转功能
func TestAutoRotate_Advanced(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("auto rotation cycle", func(t *testing.T) {
		logPath := filepath.Join(tmpDir, "auto_rotate_cycle.log")
		writer, err := NewDailyRotateWriter(logPath)
		if err != nil {
			t.Fatal(err)
		}
		defer writer.Close()

		// 写入初始数据并验证
		initialData := "initial data\n"
		_, err = writer.Write([]byte(initialData))
		if err != nil {
			t.Fatal(err)
		}

		// 验证初始文件创建
		today := time.Now().Format("2006-01-02")
		expectedPath := strings.Replace(logPath, ".log", "-"+today+".log", 1)
		if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
			t.Fatalf("Expected log file %s was not created", expectedPath)
		}

		// 循环写入并验证
		cycles := 3
		writeCount := int32(1) // 初始数据计数
		for i := 0; i < cycles; i++ {
			// 写入一批数据
			for j := 0; j < 10; j++ {
				msg := fmt.Sprintf("cycle-%d-data-%d\n", i, j)
				if _, err := writer.Write([]byte(msg)); err != nil {
					t.Errorf("Write during cycle %d failed: %v", i, err)
				}
				atomic.AddInt32(&writeCount, 1)
			}

			// 等待并验证文件内容
			time.Sleep(time.Second)
			content, err := os.ReadFile(expectedPath)
			if err != nil {
				t.Fatalf("Failed to read log file: %v", err)
			}

			// 验证消息数量
			lines := strings.Split(string(content), "\n")
			expectedLines := int(atomic.LoadInt32(&writeCount))
			if len(lines)-1 != expectedLines { // -1 因为最后一个换行符会产生空行
				t.Errorf("Cycle %d: Expected %d lines, got %d", i, expectedLines, len(lines)-1)
			}
		}
	})

	t.Run("stop during rotation", func(t *testing.T) {
		logPath := filepath.Join(tmpDir, "stop_rotate.log")
		writer, err := NewDailyRotateWriter(logPath)
		if err != nil {
			t.Fatal(err)
		}

		// 写入数据并验证
		testData := []string{
			"test data 1\n",
			"test data 2\n",
			"test data 3\n",
		}

		for _, data := range testData {
			if _, err := writer.Write([]byte(data)); err != nil {
				t.Fatal(err)
			}
		}

		// 验证文件创建和内容
		today := time.Now().Format("2006-01-02")
		expectedPath := strings.Replace(logPath, ".log", "-"+today+".log", 1)

		// 关闭前验证文件内容
		content, err := os.ReadFile(expectedPath)
		if err != nil {
			t.Fatal(err)
		}

		for _, data := range testData {
			if !strings.Contains(string(content), data) {
				t.Errorf("Missing expected data in log: %s", data)
			}
		}

		// 快速关闭并验证
		if err := writer.Close(); err != nil {
			t.Error("Close failed:", err)
		}

		// 验证关闭后不能写入
		_, err = writer.Write([]byte("post-close data\n"))
		if err == nil {
			t.Error("Expected error when writing to closed writer")
		}
	})

	t.Run("rotation with concurrent writes", func(t *testing.T) {
		logPath := filepath.Join(tmpDir, "rotate_concurrent.log")
		writer, err := NewDailyRotateWriter(logPath)
		if err != nil {
			t.Fatal(err)
		}
		defer writer.Close()

		var wg sync.WaitGroup
		writers := 5
		messagesPerWriter := 100
		writeCount := int32(0)
		errChan := make(chan error, writers*messagesPerWriter)

		// 启动并发写入
		for i := 0; i < writers; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < messagesPerWriter; j++ {
					msg := fmt.Sprintf("writer-%d-msg-%d\n", id, j)
					if _, err := writer.Write([]byte(msg)); err != nil {
						errChan <- fmt.Errorf("writer %d failed: %v", id, err)
						return
					}
					atomic.AddInt32(&writeCount, 1)
					time.Sleep(time.Millisecond)
				}
			}(i)
		}

		// 等待完成
		wg.Wait()
		close(errChan)

		// 检查错误
		for err := range errChan {
			t.Error(err)
		}

		// 验证写入计数
		actualCount := atomic.LoadInt32(&writeCount)
		expectedCount := int32(writers * messagesPerWriter)
		if actualCount != expectedCount {
			t.Errorf("Expected %d writes, got %d", expectedCount, actualCount)
		}

		// 验证文件内容
		today := time.Now().Format("2006-01-02")
		expectedPath := strings.Replace(logPath, ".log", "-"+today+".log", 1)
		content, err := os.ReadFile(expectedPath)
		if err != nil {
			t.Fatal(err)
		}

		lines := strings.Split(string(content), "\n")
		if len(lines)-1 != int(expectedCount) {
			t.Errorf("Expected %d lines in log file, got %d", expectedCount, len(lines)-1)
		}
	})

	t.Run("rotation error handling", func(t *testing.T) {
		// 创建一个子目录用于权限测试
		errorDir := filepath.Join(tmpDir, "error_test")
		if err := os.MkdirAll(errorDir, 0o755); err != nil {
			t.Fatal(err)
		}

		logPath := filepath.Join(errorDir, "rotate_error.log")

		// 修改目录权限为只读
		if err := os.Chmod(errorDir, 0o000); err != nil {
			t.Fatal("Failed to change directory permissions:", err)
		}
		defer os.Chmod(errorDir, 0o755) // 确保清理

		// 在只读目录中创建writer应该失败
		_, err := NewDailyRotateWriter(logPath)
		if err == nil {
			t.Fatal("Expected error when creating writer in read-only directory")
		}

		// 修改回正常权限
		os.Chmod(errorDir, 0o755)

		// 现在应该可以成功创建
		writer, err := NewDailyRotateWriter(logPath)
		if err != nil {
			t.Fatal(err)
		}
		defer writer.Close()

		// 写入一些数据
		if _, err := writer.Write([]byte("test data\n")); err != nil {
			t.Fatal(err)
		}

		// 将当前文件的权限设置为只读
		today := time.Now().Format("2006-01-02")
		currentLogFile := strings.TrimSuffix(logPath, ".log") + "-" + today + ".log"
		if err := os.Chmod(currentLogFile, 0o400); err != nil {
			t.Fatal("Failed to change file permissions:", err)
		}
		defer os.Chmod(currentLogFile, 0o644)

		// 修改日期以强制轮转
		newDate := time.Now().Add(24 * time.Hour).Format("2006-01-02")
		writer.lastDate = newDate // 强制下一次 rotateIfNeeded 进行轮转

		// 尝试轮转，应该会失败因为文件是只读的
		if err := writer.rotateIfNeeded(); err == nil {
			t.Error("Expected error during rotation in read-only mode")
		}
	})
}
