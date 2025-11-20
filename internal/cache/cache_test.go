package cache

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMemoryCache tests the MemoryCache implementation.
func TestMemoryCache(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	t.Run("set_and_get", func(t *testing.T) {
		key := "test:key"
		value := "test-value"

		err := cache.Set(ctx, key, value, 0)
		require.NoError(t, err)

		var result string
		err = cache.Get(ctx, key, &result)
		require.NoError(t, err)
		assert.Equal(t, value, result)
	})

	t.Run("get_nonexistent_key", func(t *testing.T) {
		var result string
		err := cache.Get(ctx, "nonexistent", &result)
		assert.Error(t, err)
		assert.Equal(t, ErrNotFound, err)
	})

	t.Run("set_with_expiration", func(t *testing.T) {
		key := "test:expiry"
		value := "expiry-value"

		err := cache.Set(ctx, key, value, 100*time.Millisecond)
		require.NoError(t, err)

		// Should exist immediately
		var result string
		err = cache.Get(ctx, key, &result)
		require.NoError(t, err)
		assert.Equal(t, value, result)

		// Wait for expiration
		time.Sleep(150 * time.Millisecond)

		// Should not exist after expiration
		err = cache.Get(ctx, key, &result)
		assert.Error(t, err)
		assert.Equal(t, ErrNotFound, err)
	})

	t.Run("delete", func(t *testing.T) {
		key := "test:delete"
		value := "delete-value"

		err := cache.Set(ctx, key, value, 0)
		require.NoError(t, err)

		err = cache.Delete(ctx, key)
		require.NoError(t, err)

		var result string
		err = cache.Get(ctx, key, &result)
		assert.Error(t, err)
		assert.Equal(t, ErrNotFound, err)
	})

	t.Run("delete_many", func(t *testing.T) {
		key1 := "test:delete:1"
		key2 := "test:delete:2"
		key3 := "test:delete:3"

		cache.Set(ctx, key1, "value1", 0)
		cache.Set(ctx, key2, "value2", 0)
		// key3 not set

		count, err := cache.DeleteMany(ctx, key1, key2, key3)
		require.NoError(t, err)
		assert.Equal(t, int64(2), count) // Only key1 and key2 were deleted
	})

	t.Run("exists", func(t *testing.T) {
		key := "test:exists"
		value := "exists-value"

		exists, err := cache.Exists(ctx, key)
		require.NoError(t, err)
		assert.False(t, exists)

		cache.Set(ctx, key, value, 0)

		exists, err = cache.Exists(ctx, key)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("expire", func(t *testing.T) {
		key := "test:expire"
		value := "expire-value"

		cache.Set(ctx, key, value, 0)

		ok, err := cache.Expire(ctx, key, 100*time.Millisecond)
		require.NoError(t, err)
		assert.True(t, ok)

		// Should expire
		time.Sleep(150 * time.Millisecond)
		exists, err := cache.Exists(ctx, key)
		require.NoError(t, err)
		assert.False(t, exists)

		// Test with nonexistent key
		ok, err = cache.Expire(ctx, "nonexistent", time.Second)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("ttl", func(t *testing.T) {
		key1 := "test:ttl:1"
		key2 := "test:ttl:2"
		key3 := "test:ttl:3"

		// Key with expiration
		cache.Set(ctx, key1, "value1", time.Second)
		ttl, err := cache.TTL(ctx, key1)
		require.NoError(t, err)
		assert.Greater(t, ttl, time.Duration(0))
		assert.LessOrEqual(t, ttl, time.Second)

		// Key without expiration
		cache.Set(ctx, key2, "value2", 0)
		ttl, err = cache.TTL(ctx, key2)
		require.NoError(t, err)
		assert.Equal(t, -1*time.Second, ttl)

		// Nonexistent key
		ttl, err = cache.TTL(ctx, key3)
		require.NoError(t, err)
		assert.Equal(t, -2*time.Second, ttl)
	})

	t.Run("get_or_set", func(t *testing.T) {
		key := "test:getorset"
		callCount := 0

		setFunc := func() (interface{}, error) {
			callCount++
			return "generated-value", nil
		}

		// First call should generate value
		var result string
		err := cache.GetOrSet(ctx, key, &result, time.Minute, setFunc)
		require.NoError(t, err)
		assert.Equal(t, "generated-value", result)
		assert.Equal(t, 1, callCount)

		// Second call should use cached value
		var result2 string
		err = cache.GetOrSet(ctx, key, &result2, time.Minute, setFunc)
		require.NoError(t, err)
		assert.Equal(t, "generated-value", result2)
		assert.Equal(t, 1, callCount) // Should not call setFunc again
	})

	t.Run("increment", func(t *testing.T) {
		key := "test:increment"

		// Increment nonexistent key (should start at 0)
		val, err := cache.Increment(ctx, key, 5)
		require.NoError(t, err)
		assert.Equal(t, int64(5), val)

		// Increment again
		val, err = cache.Increment(ctx, key, 3)
		require.NoError(t, err)
		assert.Equal(t, int64(8), val)

		// Verify value
		var result int64
		err = cache.Get(ctx, key, &result)
		require.NoError(t, err)
		assert.Equal(t, int64(8), result)
	})

	t.Run("decrement", func(t *testing.T) {
		key := "test:decrement"

		// Set initial value
		cache.Set(ctx, key, int64(10), 0)

		// Decrement
		val, err := cache.Decrement(ctx, key, 3)
		require.NoError(t, err)
		assert.Equal(t, int64(7), val)

		// Verify value
		var result int64
		err = cache.Get(ctx, key, &result)
		require.NoError(t, err)
		assert.Equal(t, int64(7), result)
	})

	t.Run("complex_types", func(t *testing.T) {
		key := "test:complex"
		type User struct {
			ID    int    `json:"id"`
			Name  string `json:"name"`
			Email string `json:"email"`
		}

		user := User{
			ID:    1,
			Name:  "John",
			Email: "john@example.com",
		}

		err := cache.Set(ctx, key, user, 0)
		require.NoError(t, err)

		var result User
		err = cache.Get(ctx, key, &result)
		require.NoError(t, err)
		assert.Equal(t, user, result)
	})
}

// TestNamespaceCache tests the NamespaceCache wrapper.
func TestNamespaceCache(t *testing.T) {
	baseCache := NewMemoryCache()
	cache := NewNamespaceCache(baseCache, "app")
	ctx := context.Background()

	t.Run("namespace_prefix", func(t *testing.T) {
		key := "user:123"
		value := "user-data"

		err := cache.Set(ctx, key, value, 0)
		require.NoError(t, err)

		// Should be stored with namespace prefix
		var result string
		err = baseCache.Get(ctx, "app:user:123", &result)
		require.NoError(t, err)
		assert.Equal(t, value, result)

		// Should retrieve with namespace
		var result2 string
		err = cache.Get(ctx, key, &result2)
		require.NoError(t, err)
		assert.Equal(t, value, result2)
	})

	t.Run("delete_many_with_namespace", func(t *testing.T) {
		key1 := "key1"
		key2 := "key2"

		cache.Set(ctx, key1, "value1", 0)
		cache.Set(ctx, key2, "value2", 0)

		count, err := cache.DeleteMany(ctx, key1, key2)
		require.NoError(t, err)
		assert.Equal(t, int64(2), count)

		// Verify deletion
		exists, err := cache.Exists(ctx, key1)
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

// TestMemoryCache_Concurrent tests concurrent access to MemoryCache.
// This test verifies thread safety by performing concurrent read/write operations.
func TestMemoryCache_Concurrent(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	t.Run("concurrent_set_and_get", func(t *testing.T) {
		const numGoroutines = 100
		const keysPerGoroutine = 10

		var wg sync.WaitGroup
		errorCount := int64(0)

		// Concurrent writes
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()
				for j := 0; j < keysPerGoroutine; j++ {
					key := fmt.Sprintf("key:%d:%d", goroutineID, j)
					value := fmt.Sprintf("value:%d:%d", goroutineID, j)
					if err := cache.Set(ctx, key, value, 0); err != nil {
						atomic.AddInt64(&errorCount, 1)
					}
				}
			}(i)
		}

		wg.Wait()
		assert.Equal(t, int64(0), errorCount, "No errors should occur during concurrent writes")

		// Concurrent reads
		readErrorCount := int64(0)
		readSuccessCount := int64(0)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()
				for j := 0; j < keysPerGoroutine; j++ {
					key := fmt.Sprintf("key:%d:%d", goroutineID, j)
					expectedValue := fmt.Sprintf("value:%d:%d", goroutineID, j)
					var result string
					if err := cache.Get(ctx, key, &result); err != nil {
						atomic.AddInt64(&readErrorCount, 1)
					} else {
						if result == expectedValue {
							atomic.AddInt64(&readSuccessCount, 1)
						}
					}
				}
			}(i)
		}

		wg.Wait()
		assert.Equal(t, int64(0), readErrorCount, "No errors should occur during concurrent reads")
		assert.Equal(t, int64(numGoroutines*keysPerGoroutine), readSuccessCount, "All reads should succeed with correct values")
	})

	t.Run("concurrent_increment", func(t *testing.T) {
		const numGoroutines = 50
		const incrementsPerGoroutine = 20
		key := "concurrent:counter"

		var wg sync.WaitGroup
		errorCount := int64(0)

		// Concurrent increments
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < incrementsPerGoroutine; j++ {
					if _, err := cache.Increment(ctx, key, 1); err != nil {
						atomic.AddInt64(&errorCount, 1)
					}
				}
			}()
		}

		wg.Wait()
		assert.Equal(t, int64(0), errorCount, "No errors should occur during concurrent increments")

		// Verify final value
		var result int64
		err := cache.Get(ctx, key, &result)
		require.NoError(t, err)
		expectedValue := int64(numGoroutines * incrementsPerGoroutine)
		assert.Equal(t, expectedValue, result, "Counter should equal total increments")
	})

	t.Run("concurrent_get_or_set", func(t *testing.T) {
		const numGoroutines = 50
		key := "concurrent:getorset"
		callCount := int64(0)

		setFunc := func() (interface{}, error) {
			atomic.AddInt64(&callCount, 1)
			time.Sleep(10 * time.Millisecond) // Simulate expensive operation
			return "generated-value", nil
		}

		var wg sync.WaitGroup
		errorCount := int64(0)
		successCount := int64(0)

		// Concurrent GetOrSet calls
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				var result string
				if err := cache.GetOrSet(ctx, key, &result, time.Minute, setFunc); err != nil {
					atomic.AddInt64(&errorCount, 1)
				} else {
					if result == "generated-value" {
						atomic.AddInt64(&successCount, 1)
					}
				}
			}()
		}

		wg.Wait()
		assert.Equal(t, int64(0), errorCount, "No errors should occur")
		assert.Equal(t, int64(numGoroutines), successCount, "All calls should succeed")
		// Note: setFunc may be called multiple times in concurrent scenarios
		// This is acceptable as long as all goroutines get the correct value
		assert.Greater(t, callCount, int64(0), "setFunc should be called at least once")
	})

	t.Run("concurrent_delete_and_get", func(t *testing.T) {
		const numKeys = 100
		const numGoroutines = 50

		// Setup: create keys
		for i := 0; i < numKeys; i++ {
			key := fmt.Sprintf("delete:key:%d", i)
			cache.Set(ctx, key, fmt.Sprintf("value:%d", i), 0)
		}

		var wg sync.WaitGroup
		deleteCount := int64(0)
		getCount := int64(0)

		// Concurrent delete and get operations
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()
				key := fmt.Sprintf("delete:key:%d", goroutineID%numKeys)
				if goroutineID%2 == 0 {
					// Delete operation
					if err := cache.Delete(ctx, key); err == nil {
						atomic.AddInt64(&deleteCount, 1)
					}
				} else {
					// Get operation
					var result string
					if err := cache.Get(ctx, key, &result); err == nil {
						atomic.AddInt64(&getCount, 1)
					}
				}
			}(i)
		}

		wg.Wait()
		// Both operations should complete without panics or data corruption
		assert.Greater(t, deleteCount+getCount, int64(0), "Some operations should succeed")
	})

	t.Run("concurrent_set_with_expiration", func(t *testing.T) {
		const numGoroutines = 50
		key := "concurrent:expiry"

		var wg sync.WaitGroup
		errorCount := int64(0)

		// Concurrent sets with different expiration times
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				value := fmt.Sprintf("value:%d", id)
				expiration := time.Duration(id+1) * time.Second
				if err := cache.Set(ctx, key, value, expiration); err != nil {
					atomic.AddInt64(&errorCount, 1)
				}
			}(i)
		}

		wg.Wait()
		assert.Equal(t, int64(0), errorCount, "No errors should occur during concurrent sets")

		// Verify key exists and has a value
		var result string
		err := cache.Get(ctx, key, &result)
		assert.NoError(t, err, "Key should exist after concurrent sets")
		assert.NotEmpty(t, result, "Value should not be empty")
	})

	t.Run("concurrent_exists_and_ttl", func(t *testing.T) {
		const numKeys = 50
		const numGoroutines = 100

		// Setup: create keys with expiration
		for i := 0; i < numKeys; i++ {
			key := fmt.Sprintf("ttl:key:%d", i)
			cache.Set(ctx, key, fmt.Sprintf("value:%d", i), time.Second)
		}

		var wg sync.WaitGroup
		existsCount := int64(0)
		ttlCount := int64(0)

		// Concurrent Exists and TTL operations
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				key := fmt.Sprintf("ttl:key:%d", id%numKeys)
				if id%2 == 0 {
					// Exists operation
					if exists, err := cache.Exists(ctx, key); err == nil && exists {
						atomic.AddInt64(&existsCount, 1)
					}
				} else {
					// TTL operation
					if ttl, err := cache.TTL(ctx, key); err == nil && ttl > 0 {
						atomic.AddInt64(&ttlCount, 1)
					}
				}
			}(i)
		}

		wg.Wait()
		// Operations should complete without errors
		assert.Greater(t, existsCount+ttlCount, int64(0), "Some operations should succeed")
	})

	t.Run("concurrent_mixed_operations", func(t *testing.T) {
		const numGoroutines = 100
		const operationsPerGoroutine = 20

		var wg sync.WaitGroup
		errorCount := int64(0)

		// Mixed concurrent operations: Set, Get, Delete, Increment, Exists
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()
				baseKey := fmt.Sprintf("mixed:%d", goroutineID)

				for j := 0; j < operationsPerGoroutine; j++ {
					key := fmt.Sprintf("%s:%d", baseKey, j)
					operation := j % 5

					switch operation {
					case 0: // Set
						if err := cache.Set(ctx, key, fmt.Sprintf("value:%d", j), 0); err != nil {
							atomic.AddInt64(&errorCount, 1)
						}
					case 1: // Get
						var result string
						if err := cache.Get(ctx, key, &result); err != nil && err != ErrNotFound {
							atomic.AddInt64(&errorCount, 1)
						}
					case 2: // Delete
						if err := cache.Delete(ctx, key); err != nil {
							atomic.AddInt64(&errorCount, 1)
						}
					case 3: // Increment
						if _, err := cache.Increment(ctx, key+"counter", 1); err != nil {
							atomic.AddInt64(&errorCount, 1)
						}
					case 4: // Exists
						if _, err := cache.Exists(ctx, key); err != nil {
							atomic.AddInt64(&errorCount, 1)
						}
					}
				}
			}(i)
		}

		wg.Wait()
		assert.Equal(t, int64(0), errorCount, "No errors should occur during mixed concurrent operations")
	})
}
