// Package cache provides a high-level caching interface and implementations.
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// MemoryCache is an in-memory implementation of the Cache interface.
// It can be used as a default cache implementation when Redis is not available,
// or for testing purposes.
type MemoryCache struct {
	mu     sync.RWMutex
	data   map[string]cacheItem
	closed bool
}

type cacheItem struct {
	value      string
	expiration time.Time
}

// NewMemoryCache creates a new in-memory cache.
// This is the default cache implementation when Redis is not configured.
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		data: make(map[string]cacheItem),
	}
}

// Get retrieves a value from the cache by key.
func (m *MemoryCache) Get(ctx context.Context, key string, value interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return fmt.Errorf("cache is closed")
	}

	item, ok := m.data[key]
	if !ok {
		return ErrNotFound
	}

	// Check expiration
	if !item.expiration.IsZero() && time.Now().After(item.expiration) {
		delete(m.data, key)
		return ErrNotFound
	}

	if err := json.Unmarshal([]byte(item.value), value); err != nil {
		return fmt.Errorf("cache unmarshal failed: %w", err)
	}

	return nil
}

// Set stores a value in the cache with the given key and expiration.
func (m *MemoryCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return fmt.Errorf("cache is closed")
	}

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache marshal failed: %w", err)
	}

	item := cacheItem{
		value: string(data),
	}
	if expiration > 0 {
		item.expiration = time.Now().Add(expiration)
	}

	m.data[key] = item
	return nil
}

// Delete removes a key from the cache.
func (m *MemoryCache) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return fmt.Errorf("cache is closed")
	}

	delete(m.data, key)
	return nil
}

// DeleteMany removes multiple keys from the cache.
func (m *MemoryCache) DeleteMany(ctx context.Context, keys ...string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return 0, fmt.Errorf("cache is closed")
	}

	count := int64(0)
	for _, key := range keys {
		if _, ok := m.data[key]; ok {
			delete(m.data, key)
			count++
		}
	}
	return count, nil
}

// Exists checks if a key exists in the cache.
func (m *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return false, fmt.Errorf("cache is closed")
	}

	item, ok := m.data[key]
	if !ok {
		return false, nil
	}

	// Check expiration
	if !item.expiration.IsZero() && time.Now().After(item.expiration) {
		delete(m.data, key)
		return false, nil
	}

	return true, nil
}

// Expire sets the expiration time for a key.
func (m *MemoryCache) Expire(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return false, fmt.Errorf("cache is closed")
	}

	item, ok := m.data[key]
	if !ok {
		return false, nil
	}

	if expiration > 0 {
		item.expiration = time.Now().Add(expiration)
	} else {
		item.expiration = time.Time{}
	}
	m.data[key] = item

	return true, nil
}

// TTL returns the remaining time to live of a key.
func (m *MemoryCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return 0, fmt.Errorf("cache is closed")
	}

	item, ok := m.data[key]
	if !ok {
		return -2 * time.Second, nil // Key doesn't exist
	}

	if item.expiration.IsZero() {
		return -1 * time.Second, nil // No expiration
	}

	ttl := time.Until(item.expiration)
	if ttl <= 0 {
		return -2 * time.Second, nil // Expired
	}

	return ttl, nil
}

// GetOrSet retrieves a value from the cache, or sets it if not found.
func (m *MemoryCache) GetOrSet(ctx context.Context, key string, value interface{}, expiration time.Duration, setFunc func() (interface{}, error)) error {
	// Try to get from cache first
	err := m.Get(ctx, key, value)
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
	if err := m.Set(ctx, key, newValue, expiration); err != nil {
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
func (m *MemoryCache) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return 0, fmt.Errorf("cache is closed")
	}

	var current int64
	item, ok := m.data[key]
	if ok {
		if err := json.Unmarshal([]byte(item.value), &current); err != nil {
			// If unmarshal fails, treat as 0
			current = 0
		}
	}

	newValue := current + delta
	data, err := json.Marshal(newValue)
	if err != nil {
		return 0, fmt.Errorf("cache marshal failed: %w", err)
	}

	m.data[key] = cacheItem{
		value:      string(data),
		expiration: item.expiration, // Preserve expiration
	}

	return newValue, nil
}

// Decrement decrements the value of a key by the given amount.
func (m *MemoryCache) Decrement(ctx context.Context, key string, delta int64) (int64, error) {
	return m.Increment(ctx, key, -delta)
}
