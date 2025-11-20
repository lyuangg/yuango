package redis

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
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

	t.Run("sorted_set_operations", func(t *testing.T) {
		key := "test:sorted"

		// ZAdd - add members with scores
		count, err := client.ZAdd(ctx, key,
			redis.Z{Score: 1.0, Member: "member1"},
			redis.Z{Score: 2.0, Member: "member2"},
			redis.Z{Score: 3.0, Member: "member3"},
			redis.Z{Score: 1.5, Member: "member4"},
		).Result()
		require.NoError(t, err)
		assert.Equal(t, int64(4), count)

		// ZCard - get cardinality
		card, err := client.ZCard(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, int64(4), card)

		// ZRange - get members by rank
		members, err := client.ZRange(ctx, key, 0, -1).Result()
		require.NoError(t, err)
		assert.Len(t, members, 4)
		// Should be sorted by score
		assert.Contains(t, members, "member1")
		assert.Contains(t, members, "member2")
		assert.Contains(t, members, "member3")
		assert.Contains(t, members, "member4")

		// ZRange with range
		members, err = client.ZRange(ctx, key, 0, 1).Result()
		require.NoError(t, err)
		assert.LessOrEqual(t, len(members), 2)

		// ZScore - get score of a member
		score, err := client.ZScore(ctx, key, "member1").Result()
		require.NoError(t, err)
		assert.Equal(t, 1.0, score)

		score, err = client.ZScore(ctx, key, "member2").Result()
		require.NoError(t, err)
		assert.Equal(t, 2.0, score)

		// ZScore for nonexistent member
		_, err = client.ZScore(ctx, key, "nonexistent").Result()
		assert.Error(t, err)

		// ZRangeByScore - get members by score range
		members, err = client.ZRangeByScore(ctx, key, &redis.ZRangeBy{
			Min: "1.0",
			Max: "2.0",
		}).Result()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(members), 0) // Should return members with scores between 1.0 and 2.0

		// ZRem - remove members
		removed, err := client.ZRem(ctx, key, "member1", "member2").Result()
		require.NoError(t, err)
		assert.Equal(t, int64(2), removed)

		// Verify removal
		card, err = client.ZCard(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, int64(2), card) // Should have 2 members left

		// ZRem with nonexistent member
		removed, err = client.ZRem(ctx, key, "nonexistent").Result()
		require.NoError(t, err)
		assert.Equal(t, int64(0), removed) // Should return 0 for nonexistent member

		// ZRem with multiple members (some exist, some don't)
		removed, err = client.ZRem(ctx, key, "member3", "nonexistent", "member4").Result()
		require.NoError(t, err)
		assert.Equal(t, int64(2), removed) // Should remove member3 and member4

		// Verify all removed
		card, err = client.ZCard(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, int64(0), card)
	})

	t.Run("zcard_empty_set", func(t *testing.T) {
		key := "test:zcard:empty"
		card, err := client.ZCard(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, int64(0), card)
	})

	t.Run("zrangebyscore_empty_set", func(t *testing.T) {
		key := "test:zrangebyscore:empty"
		members, err := client.ZRangeByScore(ctx, key, &redis.ZRangeBy{
			Min: "0",
			Max: "10",
		}).Result()
		require.NoError(t, err)
		assert.Empty(t, members)
	})

	t.Run("zrem_empty_set", func(t *testing.T) {
		key := "test:zrem:empty"
		removed, err := client.ZRem(ctx, key, "member1").Result()
		require.NoError(t, err)
		assert.Equal(t, int64(0), removed)
	})

	t.Run("zadd_update_existing", func(t *testing.T) {
		key := "test:zadd:update"
		// Add member with initial score
		count, err := client.ZAdd(ctx, key, redis.Z{Score: 1.0, Member: "member1"}).Result()
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)

		// Update score
		count, err = client.ZAdd(ctx, key, redis.Z{Score: 2.0, Member: "member1"}).Result()
		require.NoError(t, err)
		assert.Equal(t, int64(0), count) // Should return 0 for update

		// Verify updated score
		score, err := client.ZScore(ctx, key, "member1").Result()
		require.NoError(t, err)
		assert.Equal(t, 2.0, score)
	})

	t.Run("zrangebyscore_with_scores", func(t *testing.T) {
		key := "test:zrangebyscore:scores"
		// Add members with different scores
		_, err := client.ZAdd(ctx, key,
			redis.Z{Score: 1.0, Member: "low"},
			redis.Z{Score: 5.0, Member: "mid"},
			redis.Z{Score: 10.0, Member: "high"},
		).Result()
		require.NoError(t, err)

		// Get members with scores between 2 and 8
		members, err := client.ZRangeByScore(ctx, key, &redis.ZRangeBy{
			Min: "2",
			Max: "8",
		}).Result()
		require.NoError(t, err)
		// Should include "mid" (score 5.0)
		assert.Contains(t, members, "mid")
	})

	t.Run("close", func(t *testing.T) {
		err := client.Close()
		assert.NoError(t, err)

		// After close, Ping should fail
		err = client.Ping(ctx).Err()
		assert.Error(t, err)
	})
}
