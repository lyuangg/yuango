// Package database provides database connection management.
package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/lyuangg/yuango/internal/config"
	"github.com/lyuangg/yuango/internal/logging"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

// NewGormDB creates a new GORM database connection.
// It supports MySQL and SQLite drivers.
// It uses the provided application logger for GORM logging.
// If replicas are configured for MySQL, it enables read-write separation using dbresolver plugin.
func NewGormDB(cfg config.DatabaseConfig, appLogger logging.Logger) (*gorm.DB, error) {
	// Normalize driver name
	driver := strings.ToLower(cfg.Driver)
	if driver == "" {
		driver = "mysql" // Default to MySQL
	}

	// Configure GORM logger using application logger
	gormLogger := NewGormLoggerAdapter(appLogger, cfg.LogLevel)

	// SQLite doesn't support read-write separation
	if (driver == "sqlite" || driver == "sqlite3") && len(cfg.Replicas) > 0 {
		return nil, fmt.Errorf("SQLite does not support read-write separation")
	}

	var masterDialector gorm.Dialector
	switch driver {
	case "sqlite", "sqlite3":
		// For SQLite, use DSN directly or use default ":memory:" for testing
		dsn := cfg.DSN
		if dsn == "" {
			dsn = ":memory:"
		}
		masterDialector = sqlite.Open(dsn)

	case "mysql":
		// Build master DSN (no defaults needed for master)
		masterDialector = mysql.Open(buildDSN(cfg.DSN, cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, "", "", ""))

	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}

	// Open database connection
	db, err := gorm.Open(masterDialector, &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}
	if err := configureConnectionPool(sqlDB, cfg); err != nil {
		return nil, fmt.Errorf("failed to configure connection pool: %w", err)
	}

	// Enable read-write separation if replicas are configured (MySQL only)
	if driver == "mysql" && len(cfg.Replicas) > 0 {
		if err := configureReplicas(db, masterDialector, cfg); err != nil {
			return nil, err
		}
	}

	return db, nil
}

// configureReplicas configures read-write separation using dbresolver plugin.
// It builds replica dialectors from configuration and registers the dbresolver plugin.
// The plugin automatically routes:
// - Write operations (INSERT, UPDATE, DELETE) to Sources (master)
// - Read operations (SELECT) to Replicas (slaves) with load balancing
func configureReplicas(db *gorm.DB, masterDialector gorm.Dialector, cfg config.DatabaseConfig) error {
	replicaDialectors := make([]gorm.Dialector, 0, len(cfg.Replicas))
	for _, replicaCfg := range cfg.Replicas {
		replicaDSN := buildDSN(
			replicaCfg.DSN, replicaCfg.Host, replicaCfg.Port,
			replicaCfg.User, replicaCfg.Password, replicaCfg.Name,
			cfg.User, cfg.Password, cfg.Name, // defaults for replicas
		)
		replicaDialectors = append(replicaDialectors, mysql.Open(replicaDSN))
	}

	// Register dbresolver plugin for read-write separation
	err := db.Use(dbresolver.Register(dbresolver.Config{
		Sources:  []gorm.Dialector{masterDialector}, // Master for writes
		Replicas: replicaDialectors,                 // Replicas for reads
		Policy:   dbresolver.RandomPolicy{},         // Random load balancing
	}))
	if err != nil {
		return fmt.Errorf("failed to register dbresolver plugin: %w", err)
	}

	// Note: dbresolver plugin manages replica connection pools internally
	// Connection pool settings are inherited from the master configuration
	return nil
}

// DatabasesResult holds the result of initializing multiple databases.
type DatabasesResult struct {
	Databases map[string]*gorm.DB
	Default   *gorm.DB
}

// NewDatabases initializes all databases (default and named) from config.
// It handles:
// - Initializing the default database (backward compatibility)
// - Initializing named databases
// - Selecting the default database if not explicitly configured
// Returns an error if no databases are configured.
func NewDatabases(cfg config.Config, appLogger logging.Logger) (*DatabasesResult, error) {
	databases := make(map[string]*gorm.DB)
	var defaultDB *gorm.DB

	// Initialize default database (backward compatibility)
	if cfg.Database.Host != "" || cfg.Database.DSN != "" {
		db, err := NewGormDB(cfg.Database, appLogger)
		if err != nil {
			return nil, fmt.Errorf("init default database failed: %w", err)
		}
		defaultDB = db
		databases["default"] = db
	}

	// Initialize named databases
	if cfg.Databases != nil {
		for name, dbCfg := range cfg.Databases {
			// Skip if already initialized as default
			if name == "default" && defaultDB != nil {
				continue
			}
			db, err := NewGormDB(dbCfg, appLogger)
			if err != nil {
				return nil, fmt.Errorf("init database '%s' failed: %w", name, err)
			}
			databases[name] = db
		}
	}

	// Validate at least one database is configured
	if len(databases) == 0 {
		return nil, fmt.Errorf("no database configured")
	}

	// Set default DB if not set (use first database)
	if defaultDB == nil {
		for _, db := range databases {
			defaultDB = db
			break
		}
	}

	return &DatabasesResult{
		Databases: databases,
		Default:   defaultDB,
	}, nil
}

// buildDSN builds a DSN string from database configuration.
// If dsn is provided, it's used directly. Otherwise, builds DSN from components.
// Default values (defaultUser, defaultPassword, defaultName) are used for empty fields (useful for replicas).
func buildDSN(dsn, host string, port int, user, password, name string, defaultUser, defaultPassword, defaultName string) string {
	// Use DSN if provided
	if dsn != "" {
		return dsn
	}

	// Use defaults for empty fields (for replicas)
	if user == "" && defaultUser != "" {
		user = defaultUser
	}
	if password == "" && defaultPassword != "" {
		password = defaultPassword
	}
	if name == "" && defaultName != "" {
		name = defaultName
	}

	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		user, password, host, port, name)
}

// configureConnectionPool configures connection pool settings for a sql.DB.
func configureConnectionPool(sqlDB *sql.DB, cfg config.DatabaseConfig) error {
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)

	if cfg.ConnMaxLifetime != "" {
		maxLifetime, err := time.ParseDuration(cfg.ConnMaxLifetime)
		if err != nil {
			return fmt.Errorf("invalid conn_max_lifetime: %w", err)
		}
		sqlDB.SetConnMaxLifetime(maxLifetime)
	}
	return nil
}
