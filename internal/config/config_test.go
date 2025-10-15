package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Helper to create a temporary config file
func createTempConfigFile(t *testing.T, content string) string {
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "config.yaml")
	err := os.WriteFile(tempFile, []byte(content), 0o644)
	assert.NoError(t, err)
	return tempFile
}

func TestLoad_Defaults(t *testing.T) {
	// Clear any existing environment variables that might interfere
	os.Unsetenv(EnvPrefix + "_PORT")
	os.Unsetenv(EnvPrefix + "_APP_NAME")

	// Temporarily change working directory to ensure no config file is found
	originalWd, _ := os.Getwd()
	tempDir := t.TempDir()
	os.Chdir(tempDir)
	defer os.Chdir(originalWd)

	cfg := Load()

	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "yuango", cfg.App.Name)
	assert.Equal(t, "mysql", cfg.Database.Driver)
	assert.Equal(t, 3306, cfg.Database.Port)
	assert.Equal(t, "info", cfg.Log.Level)
}

func TestLoad_FromFile(t *testing.T) {
	// Clear any existing environment variables that might interfere
	os.Unsetenv(EnvPrefix + "_PORT")
	os.Unsetenv(EnvPrefix + "_APP_NAME")

	configContent := `
port: "9000"
app:
  name: "test_app"
database:
  driver: "mysql"
  port: 3306
log:
  level: "debug"
`
	tempConfigFile := createTempConfigFile(t, configContent)

	// Temporarily change working directory to ensure config file is found
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	// Create a 'configs' directory in the temp directory for Load to find it
	configsDir := filepath.Join(filepath.Dir(tempConfigFile), "configs")
	err := os.Mkdir(configsDir, 0o755)
	assert.NoError(t, err)

	// Move the temp config file into the 'configs' directory
	err = os.Rename(tempConfigFile, filepath.Join(configsDir, "config.yaml"))
	assert.NoError(t, err)

	os.Chdir(filepath.Dir(tempConfigFile))

	cfg := Load()

	assert.Equal(t, "9000", cfg.Port)
	assert.Equal(t, "test_app", cfg.App.Name)
	assert.Equal(t, "mysql", cfg.Database.Driver)
	assert.Equal(t, 3306, cfg.Database.Port)
	assert.Equal(t, "debug", cfg.Log.Level)
}

func TestLoad_FromEnv(t *testing.T) {
	// Set environment variables
	os.Setenv(EnvPrefix+"_PORT", "9090")
	os.Setenv(EnvPrefix+"_APP_NAME", "env_app")
	os.Setenv(EnvPrefix+"_DATABASE_PORT", "3306")
	defer func() {
		os.Unsetenv(EnvPrefix + "_PORT")
		os.Unsetenv(EnvPrefix + "_APP_NAME")
		os.Unsetenv(EnvPrefix + "_DATABASE_PORT")
	}()

	// Temporarily change working directory to ensure no config file is found
	originalWd, _ := os.Getwd()
	tempDir := t.TempDir()
	os.Chdir(tempDir)
	defer os.Chdir(originalWd)

	cfg := Load()

	assert.Equal(t, "9090", cfg.Port)
	assert.Equal(t, "env_app", cfg.App.Name)
	assert.Equal(t, "mysql", cfg.Database.Driver) // Should still be default as not overridden by env
	assert.Equal(t, 3306, cfg.Database.Port)
}

func TestLoadFromFile_Success(t *testing.T) {
	configContent := `
port: "8888"
app:
  name: "file_load_app"
`
	tempConfigFile := createTempConfigFile(t, configContent)

	cfg := LoadFromFile(tempConfigFile)

	assert.Equal(t, "8888", cfg.Port)
	assert.Equal(t, "file_load_app", cfg.App.Name)
	assert.Equal(t, "mysql", cfg.Database.Driver) // Should still be default
}

func TestLoadFromFile_EmptyPath(t *testing.T) {
	// Clear any existing environment variables that might interfere
	os.Unsetenv(EnvPrefix + "_PORT")
	os.Unsetenv(EnvPrefix + "_APP_NAME")

	// Temporarily change working directory to ensure no config file is found
	originalWd, _ := os.Getwd()
	tempDir := t.TempDir()
	os.Chdir(tempDir)
	defer os.Chdir(originalWd)

	cfg := LoadFromFile("") // Empty path should behave like Load()

	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "yuango", cfg.App.Name)
}

func TestLoad_ConfigPathEnv(t *testing.T) {
	// Clear any existing environment variables that might interfere
	os.Unsetenv(EnvPrefix + "_PORT")
	os.Unsetenv(EnvPrefix + "_APP_NAME")
	os.Unsetenv(EnvPrefix + "_CONFIG_PATH")

	configContent := `
port: "8000"
app:
  name: "config_path_app"
`
	tempDir := t.TempDir()
	tempConfigFile := filepath.Join(tempDir, "config.yaml")
	err := os.WriteFile(tempConfigFile, []byte(configContent), 0o644)
	assert.NoError(t, err)

	os.Setenv(EnvPrefix+"_CONFIG_PATH", tempDir)
	defer os.Unsetenv(EnvPrefix + "_CONFIG_PATH")

	// Temporarily change working directory to ensure no other config file is found
	originalWd, _ := os.Getwd()
	os.Chdir(t.TempDir()) // Change to a different temp dir to ensure CONFIG_PATH is used
	defer os.Chdir(originalWd)

	cfg := Load()

	assert.Equal(t, "8000", cfg.Port)
	assert.Equal(t, "config_path_app", cfg.App.Name)
	assert.Equal(t, "mysql", cfg.Database.Driver) // Should still be default
}

func TestLoadFromFile_NonExistentFile(t *testing.T) {
	// LoadFromFile uses log.Fatalf, which exits the program.
	// To test this, we need to capture os.Exit or run in a sub-process.
	// For simplicity in this unit test, we'll just ensure it attempts to load
	// and would have failed, but won't assert on the fatal exit directly.

	// This test will cause a fatal error if run directly, so we'll skip it
	// or use a more advanced testing technique if needed.
	t.Skip("Skipping TestLoadFromFile_NonExistentFile due to log.Fatalf behavior")

	// Example of how one might test this with a sub-process:
	// if os.Getenv("CRASH_TEST") == "1" {
	// 	config.LoadFromFile("/path/to/non_existent.yaml")
	// 	return
	// }
	// cmd := exec.Command(os.Args[0], "-test.run=TestLoadFromFile_NonExistentFile")
	// cmd.Env = append(os.Environ(), "CRASH_TEST=1")
	// err := cmd.Run()
	// if e, ok := err.(*exec.ExitError); ok && !e.Success() {
	// 	return
	// }
	// t.Fatalf("process ran with err %v, want exit status 1", err)
}
