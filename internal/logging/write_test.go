package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestDailyRotateWriter_Write_Comprehensive 全面测试写入功能
func TestDailyRotateWriter_Write_Comprehensive(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("write to uninitialized writer", func(t *testing.T) {
		// 创建一个未初始化文件的 writer
		drw := &DailyRotateWriter{
			basePath: filepath.Join(tmpDir, "uninit.log"),
			stopCh:   make(chan struct{}),
		}

		_, err := drw.Write([]byte("test"))
		if err == nil {
			t.Error("Expected error when writing to uninitialized file")
		}
	})

	t.Run("write after close", func(t *testing.T) {
		// 创建并关闭 writer
		logPath := filepath.Join(tmpDir, "close.log")
		writer, err := NewDailyRotateWriter(logPath)
		if err != nil {
			t.Fatal(err)
		}

		writer.Close()
		_, err = writer.Write([]byte("test"))
		if err == nil {
			t.Error("Expected error when writing to closed file")
		}
	})

	t.Run("concurrent writes", func(t *testing.T) {
		logPath := filepath.Join(tmpDir, "concurrent.log")
		writer, err := NewDailyRotateWriter(logPath)
		if err != nil {
			t.Fatal(err)
		}
		defer writer.Close()

		// 并发写入
		var wg sync.WaitGroup
		writeCount := 100
		wg.Add(writeCount)

		for i := 0; i < writeCount; i++ {
			go func(i int) {
				defer wg.Done()
				data := []byte("test data\n")
				n, err := writer.Write(data)
				if err != nil {
					t.Errorf("Write failed: %v", err)
				}
				if n != len(data) {
					t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
				}
			}(i)
		}

		wg.Wait()

		// 验证文件内容
		today := time.Now().Format("2006-01-02")
		ext := filepath.Ext(logPath)
		basename := logPath
		if ext != "" {
			basename = logPath[:len(logPath)-len(ext)]
		}
		filename := fmt.Sprintf("%s-%s%s", basename, today, ext)
		if ext == "" {
			filename += ".log"
		}
		content, err := os.ReadFile(filename)
		if err != nil {
			t.Fatal(err)
		}
		if len(content) == 0 {
			t.Error("Expected non-empty file content")
		}
	})

	t.Run("write during rotation", func(t *testing.T) {
		logPath := filepath.Join(tmpDir, "rotate.log")
		writer, err := NewDailyRotateWriter(logPath)
		if err != nil {
			t.Fatal(err)
		}
		defer writer.Close()

		// 启动 goroutine 执行写入
		doneChan := make(chan struct{})
		go func() {
			for i := 0; i < 1000; i++ {
				writer.Write([]byte("test data\n"))
				time.Sleep(time.Microsecond) // 给rotation一个机会
			}
			close(doneChan)
		}()

		// 同时执行多次轮转
		for i := 0; i < 5; i++ {
			writer.rotateIfNeeded()
			time.Sleep(time.Millisecond)
		}

		<-doneChan // 等待写入完成
	})

	t.Run("write large data", func(t *testing.T) {
		logPath := filepath.Join(tmpDir, "large.log")
		writer, err := NewDailyRotateWriter(logPath)
		if err != nil {
			t.Fatal(err)
		}
		defer writer.Close()

		// 创建大数据块
		data := make([]byte, 1024*1024) // 1MB
		for i := range data {
			data[i] = byte(i % 256)
		}

		// 多次写入大数据块
		for i := 0; i < 5; i++ {
			n, err := writer.Write(data)
			if err != nil {
				t.Errorf("Write failed on iteration %d: %v", i, err)
			}
			if n != len(data) {
				t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
			}
		}
	})

	t.Run("write with file permissions", func(t *testing.T) {
		logPath := filepath.Join(tmpDir, "perms.log")
		writer, err := NewDailyRotateWriter(logPath)
		if err != nil {
			t.Fatal(err)
		}

		// 写入一些数据
		testData := []byte("test data\n")
		n, err := writer.Write(testData)
		if err != nil {
			t.Fatal(err)
		}
		if n != len(testData) {
			t.Errorf("Expected to write %d bytes, wrote %d", len(testData), n)
		}

		writer.Close()

		// 验证文件权限
		today := time.Now().Format("2006-01-02")
		ext := filepath.Ext(logPath)
		basename := logPath
		if ext != "" {
			basename = logPath[:len(logPath)-len(ext)]
		}
		filename := fmt.Sprintf("%s-%s%s", basename, today, ext)
		if ext == "" {
			filename += ".log"
		}
		info, err := os.Stat(filename)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o644 {
			t.Errorf("Expected file permissions 0644, got %v", info.Mode().Perm())
		}
	})

	t.Run("write zero bytes", func(t *testing.T) {
		logPath := filepath.Join(tmpDir, "zero.log")
		writer, err := NewDailyRotateWriter(logPath)
		if err != nil {
			t.Fatal(err)
		}
		defer writer.Close()

		// 写入空数据
		n, err := writer.Write([]byte{})
		if err != nil {
			t.Error("Expected no error when writing zero bytes")
		}
		if n != 0 {
			t.Errorf("Expected to write 0 bytes, wrote %d", n)
		}
	})
}
