// Package cache provides a high-level caching interface and implementations.
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lyuangg/yuango/internal/redis"
	redisv9 "github.com/redis/go-redis/v9"
)

// RedisCache is a Redis-based implementation of the Cache interface.
type RedisCache struct {
	client redis.Client
}

// NewRedisCache creates a new Redis-based cache.
func NewRedisCache(client redis.Client) *RedisCache {
	return &RedisCache{
		client: client,
	}
}

// Get retrieves a value from the cache by key.
func (r *RedisCache) Get(ctx context.Context, key string, value interface{}) error {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redisv9.Nil {
			return ErrNotFound
		}
		return fmt.Errorf("cache get failed: %w", err)
	}

	if err := json.Unmarshal([]byte(val), value); err != nil {
		return fmt.Errorf("cache unmarshal failed: %w", err)
	}

	return nil
}

// Set stores a value in the cache with the given key and expiration.
func (r *RedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache marshal failed: %w", err)
	}

	if err := r.client.Set(ctx, key, string(data), expiration).Err(); err != nil {
		return fmt.Errorf("cache set failed: %w", err)
	}

	return nil
}

// Delete removes a key from the cache.
func (r *RedisCache) Delete(ctx context.Context, key string) error {
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("cache delete failed: %w", err)
	}
	return nil
}

// DeleteMany removes multiple keys from the cache.
func (r *RedisCache) DeleteMany(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}

	count, err := r.client.Del(ctx, keys...).Result()
	if err != nil {
		return 0, fmt.Errorf("cache delete many failed: %w", err)
	}
	return count, nil
}

// Exists checks if a key exists in the cache.
func (r *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("cache exists failed: %w", err)
	}
	return count > 0, nil
}

// Expire sets the expiration time for a key.
func (r *RedisCache) Expire(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	ok, err := r.client.Expire(ctx, key, expiration).Result()
	if err != nil {
		return false, fmt.Errorf("cache expire failed: %w", err)
	}
	return ok, nil
}

// TTL returns the remaining time to live of a key.
func (r *RedisCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	ttl, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("cache ttl failed: %w", err)
	}
	return ttl, nil
}

// GetOrSet retrieves a value from the cache, or sets it if not found.
func (r *RedisCache) GetOrSet(ctx context.Context, key string, value interface{}, expiration time.Duration, setFunc func() (interface{}, error)) error {
	// Try to get from cache first
	err := r.Get(ctx, key, value)
	if err == nil {
		return nil // Found in cache
	}
	if err != ErrNotFound {
		return err // Some other error
	}

	// Key not found, call setFunc to generate value
	newValue, err := setFunc()
	if err != nil {
		return fmt.Errorf("cache setFunc failed: %w", err)
	}

	// Set the generated value in cache
	if err := r.Set(ctx, key, newValue, expiration); err != nil {
		return err
	}

	// Unmarshal the value into the provided pointer
	data, err := json.Marshal(newValue)
	if err != nil {
		return fmt.Errorf("cache marshal failed: %w", err)
	}
	if err := json.Unmarshal(data, value); err != nil {
		return fmt.Errorf("cache unmarshal failed: %w", err)
	}

	return nil
}

// Increment increments the value of a key by the given amount.
func (r *RedisCache) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	// Get current value
	var current int64
	err := r.Get(ctx, key, &current)
	if err != nil && err != ErrNotFound {
		return 0, fmt.Errorf("cache increment get failed: %w", err)
	}

	newValue := current + delta
	// Use Set to store the value (which will JSON marshal it)
	if err := r.Set(ctx, key, newValue, 0); err != nil {
		return 0, fmt.Errorf("cache increment set failed: %w", err)
	}

	return newValue, nil
}

// Decrement decrements the value of a key by the given amount.
func (r *RedisCache) Decrement(ctx context.Context, key string, delta int64) (int64, error) {
	return r.Increment(ctx, key, -delta)
}
