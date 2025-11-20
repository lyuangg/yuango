// Package redis provides Redis client connection management.
package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/lyuangg/yuango/internal/config"
	"github.com/redis/go-redis/v9"
)

// NewRedisClient creates a new Redis client from configuration.
// Returns a Client interface for better testability.
func NewRedisClient(cfg config.RedisConfig) (Client, error) {
	// Parse timeouts
	dialTimeout, err := time.ParseDuration(cfg.DialTimeout)
	if err != nil {
		return nil, fmt.Errorf("invalid dial_timeout: %w", err)
	}

	readTimeout, err := time.ParseDuration(cfg.ReadTimeout)
	if err != nil {
		return nil, fmt.Errorf("invalid read_timeout: %w", err)
	}

	writeTimeout, err := time.ParseDuration(cfg.WriteTimeout)
	if err != nil {
		return nil, fmt.Errorf("invalid write_timeout: %w", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  dialTimeout,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return NewClientAdapter(rdb), nil
}
