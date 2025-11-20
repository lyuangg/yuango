package redis

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMockClient tests the MockClient implementation.
func TestMockClient(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	t.Run("ping", func(t *testing.T) {
		err := client.Ping(ctx).Err()
		assert.NoError(t, err)
	})

	t.Run("set_and_get", func(t *testing.T) {
		key := "test:key"
		value := "test-value"

		err := client.Set(ctx, key, value, 0).Err()
		require.NoError(t, err)

		val, err := client.Get(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, value, val)
	})

	t.Run("set_with_expiry", func(t *testing.T) {
		key := "test:expiry"
		value := "expiry-value"

		err := client.Set(ctx, key, value, time.Second).Err()
		require.NoError(t, err)

		// Should exist immediately
		val, err := client.Get(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, value, val)

		// Check TTL
		ttl, err := client.TTL(ctx, key).Result()
		require.NoError(t, err)
		assert.Greater(t, ttl, time.Duration(0))
		assert.LessOrEqual(t, ttl, time.Second)
	})

	t.Run("get_nonexistent_key", func(t *testing.T) {
		_, err := client.Get(ctx, "nonexistent").Result()
		assert.Error(t, err)
	})

	t.Run("del", func(t *testing.T) {
		key := "test:del"
		value := "del-value"

		err := client.Set(ctx, key, value, 0).Err()
		require.NoError(t, err)

		deleted, err := client.Del(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, int64(1), deleted)

		// Should not exist after deletion
		_, err = client.Get(ctx, key).Result()
		assert.Error(t, err)
	})

	t.Run("exists", func(t *testing.T) {
		key1 := "test:exists:1"
		key2 := "test:exists:2"

		err := client.Set(ctx, key1, "value1", 0).Err()
		require.NoError(t, err)

		count, err := client.Exists(ctx, key1, key2).Result()
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})

	t.Run("hash_operations", func(t *testing.T) {
		key := "test:hash"
		field1 := "field1"
		value1 := "value1"
		field2 := "field2"
		value2 := "value2"

		// HSet
		count, err := client.HSet(ctx, key, field1, value1, field2, value2).Result()
		require.NoError(t, err)
		assert.Equal(t, int64(2), count)

		// HGet
		val, err := client.HGet(ctx, key, field1).Result()
		require.NoError(t, err)
		assert.Equal(t, value1, val)

		// HGetAll
		all, err := client.HGetAll(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, value1, all[field1])
		assert.Equal(t, value2, all[field2])

		// HDel
		deleted, err := client.HDel(ctx, key, field1).Result()
		require.NoError(t, err)
		assert.Equal(t, int64(1), deleted)

		// Verify deletion
		_, err = client.HGet(ctx, key, field1).Result()
		assert.Error(t, err)
	})

	t.Run("list_operations", func(t *testing.T) {
		key := "test:list"

		// LPush
		length, err := client.LPush(ctx, key, "value1", "value2").Result()
		require.NoError(t, err)
		assert.Equal(t, int64(2), length)

		// LLen
		length, err = client.LLen(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, int64(2), length)

		// LRange
		values, err := client.LRange(ctx, key, 0, -1).Result()
		require.NoError(t, err)
		assert.Equal(t, []string{"value1", "value2"}, values)

		// LPop
		val, err := client.LPop(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, "value1", val)

		// RPush
		length, err = client.RPush(ctx, key, "value3").Result()
		require.NoError(t, err)
		assert.Equal(t, int64(2), length)

		// RPop
		val, err = client.RPop(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, "value3", val)
	})

	t.Run("set_operations", func(t *testing.T) {
		key := "test:set"

		// SAdd
		count, err := client.SAdd(ctx, key, "member1", "member2", "member1").Result()
		require.NoError(t, err)
		assert.Equal(t, int64(2), count) // member1 added twice, but only counted once

		// SMembers
		members, err := client.SMembers(ctx, key).Result()
		require.NoError(t, err)
		assert.Len(t, members, 2)

		// SIsMember
		isMember, err := client.SIsMember(ctx, key, "member1").Result()
		require.NoError(t, err)
		assert.True(t, isMember)

		isMember, err = client.SIsMember(ctx, key, "nonexistent").Result()
		require.NoError(t, err)
		assert.False(t, isMember)

		// SCard
		card, err := client.SCard(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, int64(2), card)
	})

	t.Run("close", func(t *testing.T) {
		err := client.Close()
		assert.NoError(t, err)

		// After close, Ping should fail
		err = client.Ping(ctx).Err()
		assert.Error(t, err)
	})
}
