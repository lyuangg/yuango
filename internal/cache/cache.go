// Package cache provides a high-level caching interface and implementations.
package cache

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrNotFound is returned when a key is not found in the cache.
	ErrNotFound = errors.New("cache: key not found")

	// ErrInvalidValue is returned when the value type is invalid.
	ErrInvalidValue = errors.New("cache: invalid value type")
)

// Cache defines the interface for cache operations.
// This interface abstracts away the underlying storage implementation (Redis, memory, etc.).
type Cache interface {
	// Get retrieves a value from the cache by key.
	// Returns ErrNotFound if the key does not exist.
	// The value will be unmarshaled into the provided pointer.
	Get(ctx context.Context, key string, value interface{}) error

	// Set stores a value in the cache with the given key and expiration.
	// If expiration is 0, the key will not expire.
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error

	// Delete removes a key from the cache.
	// Returns no error if the key does not exist.
	Delete(ctx context.Context, key string) error

	// DeleteMany removes multiple keys from the cache.
	// Returns the number of keys deleted.
	DeleteMany(ctx context.Context, keys ...string) (int64, error)

	// Exists checks if a key exists in the cache.
	Exists(ctx context.Context, key string) (bool, error)

	// Expire sets the expiration time for a key.
	// Returns false if the key does not exist.
	Expire(ctx context.Context, key string, expiration time.Duration) (bool, error)

	// TTL returns the remaining time to live of a key.
	// Returns -1 if the key exists but has no expiration.
	// Returns -2 if the key does not exist.
	TTL(ctx context.Context, key string) (time.Duration, error)

	// GetOrSet retrieves a value from the cache, or sets it if not found.
	// The setFunc is called to generate the value if the key doesn't exist.
	// The generated value is cached with the given expiration.
	GetOrSet(ctx context.Context, key string, value interface{}, expiration time.Duration, setFunc func() (interface{}, error)) error

	// Increment increments the value of a key by the given amount.
	// If the key does not exist, it is initialized to 0 before incrementing.
	Increment(ctx context.Context, key string, delta int64) (int64, error)

	// Decrement decrements the value of a key by the given amount.
	// If the key does not exist, it is initialized to 0 before decrementing.
	Decrement(ctx context.Context, key string, delta int64) (int64, error)
}

// NamespaceCache wraps a Cache with a namespace prefix.
// All keys are automatically prefixed with the namespace.
type NamespaceCache struct {
	Cache
	namespace string
}

// NewNamespaceCache creates a new NamespaceCache with the given namespace.
func NewNamespaceCache(cache Cache, namespace string) *NamespaceCache {
	return &NamespaceCache{
		Cache:     cache,
		namespace: namespace,
	}
}

// buildKey adds the namespace prefix to the key.
func (n *NamespaceCache) buildKey(key string) string {
	if n.namespace == "" {
		return key
	}
	return n.namespace + ":" + key
}

// Get retrieves a value from the cache with namespace prefix.
func (n *NamespaceCache) Get(ctx context.Context, key string, value interface{}) error {
	return n.Cache.Get(ctx, n.buildKey(key), value)
}

// Set stores a value in the cache with namespace prefix.
func (n *NamespaceCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return n.Cache.Set(ctx, n.buildKey(key), value, expiration)
}

// Delete removes a key from the cache with namespace prefix.
func (n *NamespaceCache) Delete(ctx context.Context, key string) error {
	return n.Cache.Delete(ctx, n.buildKey(key))
}

// DeleteMany removes multiple keys from the cache with namespace prefix.
func (n *NamespaceCache) DeleteMany(ctx context.Context, keys ...string) (int64, error) {
	prefixedKeys := make([]string, len(keys))
	for i, key := range keys {
		prefixedKeys[i] = n.buildKey(key)
	}
	return n.Cache.DeleteMany(ctx, prefixedKeys...)
}

// Exists checks if a key exists in the cache with namespace prefix.
func (n *NamespaceCache) Exists(ctx context.Context, key string) (bool, error) {
	return n.Cache.Exists(ctx, n.buildKey(key))
}

// Expire sets the expiration time for a key with namespace prefix.
func (n *NamespaceCache) Expire(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	return n.Cache.Expire(ctx, n.buildKey(key), expiration)
}

// TTL returns the remaining time to live of a key with namespace prefix.
func (n *NamespaceCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	return n.Cache.TTL(ctx, n.buildKey(key))
}

// GetOrSet retrieves or sets a value with namespace prefix.
func (n *NamespaceCache) GetOrSet(ctx context.Context, key string, value interface{}, expiration time.Duration, setFunc func() (interface{}, error)) error {
	return n.Cache.GetOrSet(ctx, n.buildKey(key), value, expiration, setFunc)
}

// Increment increments a key with namespace prefix.
func (n *NamespaceCache) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	return n.Cache.Increment(ctx, n.buildKey(key), delta)
}

// Decrement decrements a key with namespace prefix.
func (n *NamespaceCache) Decrement(ctx context.Context, key string, delta int64) (int64, error) {
	return n.Cache.Decrement(ctx, n.buildKey(key), delta)
}
