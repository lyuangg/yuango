package cache

import (
	"context"
	"testing"
	"time"

	"github.com/lyuangg/yuango/internal/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRedisCache tests the RedisCache implementation.
func TestRedisCache(t *testing.T) {
	mockClient := redis.NewMockClient()
	cache := NewRedisCache(mockClient)
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

	t.Run("delete_many_empty", func(t *testing.T) {
		count, err := cache.DeleteMany(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
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

	t.Run("get_or_set_with_error", func(t *testing.T) {
		key := "test:getorset:error"
		setFunc := func() (interface{}, error) {
			return nil, assert.AnError
		}

		var result string
		err := cache.GetOrSet(ctx, key, &result, time.Minute, setFunc)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "setFunc failed")
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

	t.Run("increment_negative", func(t *testing.T) {
		key := "test:increment:neg"

		val, err := cache.Increment(ctx, key, -5)
		require.NoError(t, err)
		assert.Equal(t, int64(-5), val)
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

	t.Run("set_invalid_json", func(t *testing.T) {
		key := "test:invalid"
		// Create a value that cannot be marshaled to JSON
		invalidValue := make(chan int) // channels cannot be marshaled

		err := cache.Set(ctx, key, invalidValue, 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "marshal failed")
	})

	t.Run("get_invalid_json", func(t *testing.T) {
		key := "test:invalid:get"
		// Set a value that is not valid JSON for the target type
		cache.Set(ctx, key, "not-a-number", 0)

		var result int64
		err := cache.Get(ctx, key, &result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unmarshal failed")
	})
}
