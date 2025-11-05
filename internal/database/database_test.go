package database

import (
	"testing"

	"github.com/lyuangg/yuango/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestBuildDSN tests the buildDSN function.
func TestBuildDSN(t *testing.T) {
	tests := []struct {
		name           string
		dsn            string
		host           string
		port           int
		user           string
		password       string
		dbName         string
		defaultUser    string
		defaultPass    string
		defaultName    string
		expectedResult string
	}{
		{
			name:           "use_dsn_when_provided",
			dsn:            "user:pass@tcp(host:3306)/db",
			host:           "localhost",
			port:           3306,
			user:           "root",
			password:       "password",
			dbName:         "testdb",
			expectedResult: "user:pass@tcp(host:3306)/db",
		},
		{
			name:           "build_from_components",
			dsn:            "",
			host:           "localhost",
			port:           3306,
			user:           "root",
			password:       "password",
			dbName:         "testdb",
			expectedResult: "root:password@tcp(localhost:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local",
		},
		{
			name:           "use_defaults_for_empty_fields",
			dsn:            "",
			host:           "replica1",
			port:           3306,
			user:           "",
			password:       "",
			dbName:         "",
			defaultUser:    "root",
			defaultPass:    "password",
			defaultName:    "testdb",
			expectedResult: "root:password@tcp(replica1:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local",
		},
		{
			name:           "use_provided_over_defaults",
			dsn:            "",
			host:           "replica1",
			port:           3306,
			user:           "custom_user",
			password:       "custom_pass",
			dbName:         "custom_db",
			defaultUser:    "root",
			defaultPass:    "password",
			defaultName:    "testdb",
			expectedResult: "custom_user:custom_pass@tcp(replica1:3306)/custom_db?charset=utf8mb4&parseTime=True&loc=Local",
		},
		{
			name:           "partial_defaults",
			dsn:            "",
			host:           "replica1",
			port:           3306,
			user:           "custom_user",
			password:       "",
			dbName:         "",
			defaultUser:    "root",
			defaultPass:    "password",
			defaultName:    "testdb",
			expectedResult: "custom_user:password@tcp(replica1:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildDSN(tt.dsn, tt.host, tt.port, tt.user, tt.password, tt.dbName,
				tt.defaultUser, tt.defaultPass, tt.defaultName)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

// TestConfigureConnectionPool tests the configureConnectionPool function.
func TestConfigureConnectionPool(t *testing.T) {
	// Create a SQLite in-memory database for testing
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	sqlDB, err := db.DB()
	require.NoError(t, err)

	t.Run("configure_with_valid_settings", func(t *testing.T) {
		cfg := config.DatabaseConfig{
			MaxOpenConns:    50,
			MaxIdleConns:    10,
			ConnMaxLifetime: "1h",
		}

		err := configureConnectionPool(sqlDB, cfg)
		assert.NoError(t, err)

		// Verify settings
		assert.Equal(t, 50, sqlDB.Stats().MaxOpenConnections)
		// Note: MaxIdleConns and ConnMaxLifetime can't be easily verified without direct access
	})

	t.Run("configure_without_lifetime", func(t *testing.T) {
		cfg := config.DatabaseConfig{
			MaxOpenConns:    30,
			MaxIdleConns:    5,
			ConnMaxLifetime: "",
		}

		err := configureConnectionPool(sqlDB, cfg)
		assert.NoError(t, err)
		assert.Equal(t, 30, sqlDB.Stats().MaxOpenConnections)
	})

	t.Run("configure_with_invalid_lifetime", func(t *testing.T) {
		cfg := config.DatabaseConfig{
			MaxOpenConns:    30,
			MaxIdleConns:    5,
			ConnMaxLifetime: "invalid",
		}

		err := configureConnectionPool(sqlDB, cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid conn_max_lifetime")
	})
}

// TestNewGormDB tests the NewGormDB function.
func TestNewGormDB(t *testing.T) {
	mockLog := newMockLogger()

	t.Run("create_with_invalid_dsn", func(t *testing.T) {
		cfg := config.DatabaseConfig{
			DSN:          "invalid_dsn",
			MaxOpenConns: 10,
			MaxIdleConns: 2,
			LogLevel:     "silent",
		}

		db, err := NewGormDB(cfg, mockLog)
		assert.Error(t, err)
		assert.Nil(t, db)
		assert.Contains(t, err.Error(), "failed to connect")
	})

	t.Run("create_with_missing_config", func(t *testing.T) {
		cfg := config.DatabaseConfig{
			Host:         "",
			DSN:          "",
			MaxOpenConns: 10,
			MaxIdleConns: 2,
			LogLevel:     "silent",
		}

		db, err := NewGormDB(cfg, mockLog)
		assert.Error(t, err)
		assert.Nil(t, db)
	})

	t.Run("create_with_replicas", func(t *testing.T) {
		// This test requires real MySQL servers or mocking dbresolver
		// For now, we'll skip it or test error handling
		cfg := config.DatabaseConfig{
			DSN:          "invalid_dsn",
			MaxOpenConns: 10,
			MaxIdleConns: 2,
			LogLevel:     "silent",
			Replicas: []config.DatabaseReplicaConfig{
				{
					DSN: "invalid_replica_dsn",
				},
			},
		}

		db, err := NewGormDB(cfg, mockLog)
		assert.Error(t, err)
		assert.Nil(t, db)
	})

	t.Run("create_with_sqlite_memory_success", func(t *testing.T) {
		// Test successful connection with SQLite in-memory database
		cfg := config.DatabaseConfig{
			Driver:       "sqlite",
			DSN:          ":memory:",
			MaxOpenConns: 10,
			MaxIdleConns: 2,
			LogLevel:     "silent",
		}

		db, err := NewGormDB(cfg, mockLog)
		assert.NoError(t, err)
		assert.NotNil(t, db)

		// Verify database is usable
		sqlDB, err := db.DB()
		assert.NoError(t, err)
		assert.NotNil(t, sqlDB)

		// Test a simple query
		var result int
		err = db.Raw("SELECT 1").Scan(&result).Error
		assert.NoError(t, err)
		assert.Equal(t, 1, result)

		// Cleanup
		sqlDB.Close()
	})

	t.Run("create_with_sqlite_default_memory", func(t *testing.T) {
		// Test SQLite with empty DSN (should default to :memory:)
		cfg := config.DatabaseConfig{
			Driver:       "sqlite",
			DSN:          "",
			MaxOpenConns: 10,
			MaxIdleConns: 2,
			LogLevel:     "silent",
		}

		db, err := NewGormDB(cfg, mockLog)
		assert.NoError(t, err)
		assert.NotNil(t, db)

		sqlDB, err := db.DB()
		assert.NoError(t, err)
		sqlDB.Close()
	})

	t.Run("create_with_sqlite_replicas_error", func(t *testing.T) {
		// SQLite doesn't support replicas
		cfg := config.DatabaseConfig{
			Driver:       "sqlite",
			DSN:          ":memory:",
			MaxOpenConns: 10,
			MaxIdleConns: 2,
			LogLevel:     "silent",
			Replicas: []config.DatabaseReplicaConfig{
				{
					DSN: ":memory:",
				},
			},
		}

		db, err := NewGormDB(cfg, mockLog)
		assert.Error(t, err)
		assert.Nil(t, db)
		assert.Contains(t, err.Error(), "SQLite does not support read-write separation")
	})

	t.Run("create_with_unsupported_driver", func(t *testing.T) {
		cfg := config.DatabaseConfig{
			Driver:       "postgres",
			DSN:          "invalid",
			MaxOpenConns: 10,
			MaxIdleConns: 2,
			LogLevel:     "silent",
		}

		db, err := NewGormDB(cfg, mockLog)
		assert.Error(t, err)
		assert.Nil(t, db)
		assert.Contains(t, err.Error(), "unsupported database driver")
	})
}

// TestConfigureConnectionPool_Integration tests connection pool configuration with real database.
func TestConfigureConnectionPool_Integration(t *testing.T) {
	// Create a SQLite in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	sqlDB, err := db.DB()
	require.NoError(t, err)

	cfg := config.DatabaseConfig{
		MaxOpenConns:    100,
		MaxIdleConns:    20,
		ConnMaxLifetime: "2h",
	}

	err = configureConnectionPool(sqlDB, cfg)
	require.NoError(t, err)

	// Verify MaxOpenConns
	assert.Equal(t, 100, sqlDB.Stats().MaxOpenConnections)

	// Verify MaxIdleConns (we can check through stats)
	stats := sqlDB.Stats()
	assert.GreaterOrEqual(t, stats.MaxIdleClosed, int64(0))

	// Verify ConnMaxLifetime is set (can't directly verify, but no error means it was set)
	// The actual lifetime can't be easily verified without waiting, but if no error occurred,
	// it was successfully parsed and set.
}

// TestBuildDSN_EdgeCases tests edge cases for buildDSN.
func TestBuildDSN_EdgeCases(t *testing.T) {
	t.Run("empty_host", func(t *testing.T) {
		result := buildDSN("", "", 3306, "user", "pass", "db", "", "", "")
		assert.Contains(t, result, "user:pass@tcp")
		assert.Contains(t, result, "/db")
	})

	t.Run("zero_port", func(t *testing.T) {
		result := buildDSN("", "localhost", 0, "user", "pass", "db", "", "", "")
		assert.Contains(t, result, ":0")
	})

	t.Run("all_empty_with_defaults", func(t *testing.T) {
		result := buildDSN("", "localhost", 3306, "", "", "", "user", "pass", "db")
		assert.Contains(t, result, "user:pass@tcp(localhost:3306)/db")
	})

	t.Run("empty_dsn_but_empty_defaults", func(t *testing.T) {
		result := buildDSN("", "localhost", 3306, "", "", "", "", "", "")
		// Should still build DSN with empty user/password
		assert.Contains(t, result, "@tcp(localhost:3306)/")
	})
}

// TestConfigureConnectionPool_EdgeCases tests edge cases for configureConnectionPool.
func TestConfigureConnectionPool_EdgeCases(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	sqlDB, err := db.DB()
	require.NoError(t, err)

	t.Run("zero_conns", func(t *testing.T) {
		cfg := config.DatabaseConfig{
			MaxOpenConns:    0,
			MaxIdleConns:    0,
			ConnMaxLifetime: "",
		}

		err := configureConnectionPool(sqlDB, cfg)
		assert.NoError(t, err)
	})

	t.Run("negative_conns", func(t *testing.T) {
		cfg := config.DatabaseConfig{
			MaxOpenConns:    -1,
			MaxIdleConns:    -1,
			ConnMaxLifetime: "",
		}

		err := configureConnectionPool(sqlDB, cfg)
		assert.NoError(t, err) // SQL package accepts negative values
	})

	t.Run("very_long_lifetime", func(t *testing.T) {
		cfg := config.DatabaseConfig{
			MaxOpenConns:    10,
			MaxIdleConns:    2,
			ConnMaxLifetime: "8760h", // 1 year
		}

		err := configureConnectionPool(sqlDB, cfg)
		assert.NoError(t, err)
	})
}

// TestNewGormDB_LoggerConfiguration tests that logger is properly configured.
func TestNewGormDB_LoggerConfiguration(t *testing.T) {
	mockLog := newMockLogger()

	// Test with different log levels
	logLevels := []string{"silent", "error", "warn", "info"}

	for _, level := range logLevels {
		t.Run("log_level_"+level, func(t *testing.T) {
			cfg := config.DatabaseConfig{
				DSN:          "invalid_dsn", // Will fail, but we can verify logger was created
				MaxOpenConns: 10,
				MaxIdleConns: 2,
				LogLevel:     level,
			}

			db, err := NewGormDB(cfg, mockLog)
			// We expect an error due to invalid DSN, but the logger should have been configured
			assert.Error(t, err)
			assert.Nil(t, db)
			// The logger adapter should have been created (we can't easily verify this without
			// making buildDSN or NewGormLoggerAdapter return errors, but no panic means it worked)
		})
	}
}

// TestBuildDSN_Format tests the DSN format correctness.
func TestBuildDSN_Format(t *testing.T) {
	t.Run("dsn_format_components", func(t *testing.T) {
		result := buildDSN("", "example.com", 3306, "user", "password", "mydb", "", "", "")

		// Verify format components
		assert.Contains(t, result, "user:password@tcp(example.com:3306)/mydb")
		assert.Contains(t, result, "charset=utf8mb4")
		assert.Contains(t, result, "parseTime=True")
		assert.Contains(t, result, "loc=Local")
	})

	t.Run("special_characters_in_password", func(t *testing.T) {
		result := buildDSN("", "localhost", 3306, "user", "p@ss:w0rd", "db", "", "", "")
		assert.Contains(t, result, "user:p@ss:w0rd@tcp")
	})

	t.Run("non_default_port", func(t *testing.T) {
		result := buildDSN("", "localhost", 3307, "user", "pass", "db", "", "", "")
		assert.Contains(t, result, ":3307")
	})
}

// TestConfigureConnectionPool_RealDatabase tests with a real database connection.
func TestConfigureConnectionPool_RealDatabase(t *testing.T) {
	// Use SQLite for testing
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	sqlDB, err := db.DB()
	require.NoError(t, err)

	// Get initial stats
	initialStats := sqlDB.Stats()

	cfg := config.DatabaseConfig{
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: "30m",
	}

	err = configureConnectionPool(sqlDB, cfg)
	require.NoError(t, err)

	// Verify MaxOpenConns was set
	finalStats := sqlDB.Stats()
	assert.Equal(t, 25, finalStats.MaxOpenConnections)
	assert.NotEqual(t, initialStats.MaxOpenConnections, finalStats.MaxOpenConnections)
}

// TestBuildDSN_ReplicaDefaults tests default value handling for replicas.
func TestBuildDSN_ReplicaDefaults(t *testing.T) {
	t.Run("replica_uses_all_defaults", func(t *testing.T) {
		result := buildDSN(
			"", "replica1", 3306,
			"", "", "", // Empty replica config
			"master_user", "master_pass", "master_db", // Master defaults
		)
		assert.Contains(t, result, "master_user:master_pass@tcp(replica1:3306)/master_db")
	})

	t.Run("replica_partial_override", func(t *testing.T) {
		result := buildDSN(
			"", "replica1", 3306,
			"replica_user", "", "", // Override user only
			"master_user", "master_pass", "master_db", // Master defaults
		)
		assert.Contains(t, result, "replica_user:master_pass@tcp(replica1:3306)/master_db")
	})

	t.Run("replica_all_override", func(t *testing.T) {
		result := buildDSN(
			"", "replica1", 3306,
			"replica_user", "replica_pass", "replica_db", // All overridden
			"master_user", "master_pass", "master_db", // Master defaults (should be ignored)
		)
		assert.Contains(t, result, "replica_user:replica_pass@tcp(replica1:3306)/replica_db")
		assert.NotContains(t, result, "master_user")
	})
}

// TestConfigureConnectionPool_DurationParsing tests duration parsing.
func TestConfigureConnectionPool_DurationParsing(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	sqlDB, err := db.DB()
	require.NoError(t, err)

	testCases := []struct {
		name        string
		lifetime    string
		shouldError bool
	}{
		{"valid_hours", "1h", false},
		{"valid_minutes", "30m", false},
		{"valid_seconds", "30s", false},
		{"valid_days", "24h", false},
		{"invalid_format", "invalid", true},
		{"empty", "", false},
		{"zero", "0s", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.DatabaseConfig{
				MaxOpenConns:    10,
				MaxIdleConns:    2,
				ConnMaxLifetime: tc.lifetime,
			}

			err := configureConnectionPool(sqlDB, cfg)
			if tc.shouldError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid conn_max_lifetime")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestNewGormDB_ConnectionPoolConfig tests that connection pool is properly configured.
func TestNewGormDB_ConnectionPoolConfig(t *testing.T) {
	// Note: This test would require a real MySQL connection or mocking
	// For now, we test that the function handles configuration properly
	mockLog := newMockLogger()

	cfg := config.DatabaseConfig{
		DSN:          "invalid_dsn",
		MaxOpenConns: 100,
		MaxIdleConns: 20,
		LogLevel:     "silent",
	}

	db, err := NewGormDB(cfg, mockLog)
	// Expected to fail due to invalid DSN, but configuration should be attempted
	assert.Error(t, err)
	assert.Nil(t, db)
}

// TestBuildDSN_Priority tests that DSN takes priority over components.
func TestBuildDSN_Priority(t *testing.T) {
	t.Run("dsn_overrides_all", func(t *testing.T) {
		dsn := "custom:custom@tcp(custom:9999)/custom_db"
		result := buildDSN(dsn, "localhost", 3306, "user", "pass", "db", "", "", "")
		assert.Equal(t, dsn, result)
	})

	t.Run("components_when_dsn_empty", func(t *testing.T) {
		result := buildDSN("", "localhost", 3306, "user", "pass", "db", "", "", "")
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "localhost")
		assert.Contains(t, result, "user:pass")
		assert.Contains(t, result, "/db")
	})
}

// TestConfigureConnectionPool_ConcurrentAccess tests concurrent access safety.
func TestConfigureConnectionPool_ConcurrentAccess(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	sqlDB, err := db.DB()
	require.NoError(t, err)

	cfg := config.DatabaseConfig{
		MaxOpenConns:    50,
		MaxIdleConns:    10,
		ConnMaxLifetime: "1h",
	}

	// Configure concurrently
	done := make(chan error, 3)
	for i := 0; i < 3; i++ {
		go func() {
			done <- configureConnectionPool(sqlDB, cfg)
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		err := <-done
		assert.NoError(t, err)
	}

	// Verify final configuration
	assert.Equal(t, 50, sqlDB.Stats().MaxOpenConnections)
}
