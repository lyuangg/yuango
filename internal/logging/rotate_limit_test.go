package logging

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRotateWriter_MaxFiles(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "test")

	// 手动创建5个模拟的日志文件，日期从5天前到1天前
	today := time.Now()
	for i := 5; i >= 1; i-- {
		date := today.AddDate(0, 0, -i).Format("2006-01-02")
		filename := filepath.Join(tempDir, "test-"+date+".log")
		f, err := os.Create(filename)
		if err != nil {
			t.Fatalf("Failed to create test log file: %v", err)
		}
		f.WriteString("test log content")
		f.Close()

		// 确保文件修改时间按顺序排列
		modTime := today.AddDate(0, 0, -i)
		err = os.Chtimes(filename, modTime, modTime)
		if err != nil {
			t.Fatalf("Failed to set file time: %v", err)
		}
	}

	// 创建一个保留3个文件的RotateWriter
	maxFiles := 3
	drw, err := NewRotateWriter(logPath, "daily", maxFiles)
	if err != nil {
		t.Fatalf("Failed to create RotateWriter: %v", err)
	}
	defer drw.Close()

	// 列出目录中的所有日志文件
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	// 计算.log文件的数量
	var logFiles []string
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".log" {
			logFiles = append(logFiles, entry.Name())
		}
	}

	// 验证保留的文件数量是否符合预期
	if len(logFiles) != maxFiles {
		t.Errorf("Expected %d log files, but found %d", maxFiles, len(logFiles))
	}

	// 验证是否保留了最新的文件
	// a. a new log file for the current day
	date := today.Format("2006-01-02")
	filename := filepath.Join(tempDir, "test-"+date+".log")
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Errorf("Expected file %s to exist", filename)
	}
	// b. the other (maxFiles - 1) log files
	for i := 1; i < maxFiles; i++ {
		date := today.AddDate(0, 0, -i).Format("2006-01-02")
		filename := filepath.Join(tempDir, "test-"+date+".log")
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			t.Errorf("Expected file %s to exist", filename)
		}
	}

	// 验证最旧的文件是否被删除
	for i := maxFiles; i <= 5; i++ {
		date := today.AddDate(0, 0, -i).Format("2006-01-02")
		filename := filepath.Join(tempDir, "test-"+date+".log")
		if _, err := os.Stat(filename); !os.IsNotExist(err) {
			t.Errorf("Expected file %s to be deleted", filename)
		}
	}
}

func TestRotateWriter_MaxFiles_Hourly(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "test")

	// 手动创建5个模拟的日志文件，时间从5小时前到1小时前
	now := time.Now()
	for i := 5; i >= 1; i-- {
		hour := now.Add(-time.Duration(i) * time.Hour).Format("2006-01-02-15")
		filename := filepath.Join(tempDir, "test-"+hour+".log")
		f, err := os.Create(filename)
		if err != nil {
			t.Fatalf("Failed to create test log file: %v", err)
		}
		f.WriteString("test log content")
		f.Close()

		// 确保文件修改时间按顺序排列
		modTime := now.Add(-time.Duration(i) * time.Hour)
		err = os.Chtimes(filename, modTime, modTime)
		if err != nil {
			t.Fatalf("Failed to set file time: %v", err)
		}
	}

	// 创建一个保留3个文件的RotateWriter
	maxFiles := 3
	drw, err := NewRotateWriter(logPath, "hourly", maxFiles)
	if err != nil {
		t.Fatalf("Failed to create RotateWriter: %v", err)
	}
	defer drw.Close()

	// 列出目录中的所有日志文件
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	// 计算.log文件的数量
	var logFiles []string
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".log" {
			logFiles = append(logFiles, entry.Name())
		}
	}

	// 验证保留的文件数量是否符合预期
	if len(logFiles) != maxFiles {
		t.Errorf("Expected %d log files, but found %d", maxFiles, len(logFiles))
	}

	// 验证是否保留了最新的文件
	// a. a new log file for the current hour
	hour := now.Format("2006-01-02-15")
	filename := filepath.Join(tempDir, "test-"+hour+".log")
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Errorf("Expected file %s to exist", filename)
	}
	// b. the other (maxFiles - 1) log files
	for i := 1; i < maxFiles; i++ {
		hour := now.Add(-time.Duration(i) * time.Hour).Format("2006-01-02-15")
		filename := filepath.Join(tempDir, "test-"+hour+".log")
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			t.Errorf("Expected file %s to exist", filename)
		}
	}

	// 验证最旧的文件是否被删除
	for i := maxFiles; i <= 5; i++ {
		hour := now.Add(-time.Duration(i) * time.Hour).Format("2006-01-02-15")
		filename := filepath.Join(tempDir, "test-"+hour+".log")
		if _, err := os.Stat(filename); !os.IsNotExist(err) {
			t.Errorf("Expected file %s to be deleted", filename)
		}
	}
}
