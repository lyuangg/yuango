// Package redis provides Redis client connection management.
package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client defines the interface for Redis operations.
// This interface allows for easy mocking in tests.
type Client interface {
	// String operations
	// Get fetches the value stored at key.
	Get(ctx context.Context, key string) *redis.StringCmd
	// Set writes value at key with an optional expiration.
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	// Del deletes one or more keys and returns the count of removed entries.
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	// Exists checks whether the provided keys exist.
	Exists(ctx context.Context, keys ...string) *redis.IntCmd
	// Expire sets the expiration time for a key.
	Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
	// TTL returns the remaining time to live for a key.
	TTL(ctx context.Context, key string) *redis.DurationCmd

	// Hash operations
	// HGet retrieves the value of a hash field.
	HGet(ctx context.Context, key, field string) *redis.StringCmd
	// HSet sets one or more field-value pairs on a hash.
	HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
	// HGetAll returns all field-value pairs for a hash.
	HGetAll(ctx context.Context, key string) *redis.MapStringStringCmd
	// HDel deletes one or more fields from a hash.
	HDel(ctx context.Context, key string, fields ...string) *redis.IntCmd

	// List operations
	// LPush prepends one or more values to a list.
	LPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
	// RPush appends one or more values to a list.
	RPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
	// LPop removes and returns the first element of a list.
	LPop(ctx context.Context, key string) *redis.StringCmd
	// RPop removes and returns the last element of a list.
	RPop(ctx context.Context, key string) *redis.StringCmd
	// LLen returns the length of a list.
	LLen(ctx context.Context, key string) *redis.IntCmd
	// LRange retrieves a range of elements from a list.
	LRange(ctx context.Context, key string, start, stop int64) *redis.StringSliceCmd

	// Set operations
	// SAdd adds one or more members to a set.
	SAdd(ctx context.Context, key string, members ...interface{}) *redis.IntCmd
	// SMembers returns all members of a set.
	SMembers(ctx context.Context, key string) *redis.StringSliceCmd
	// SIsMember checks if a value is a member of a set.
	SIsMember(ctx context.Context, key string, member interface{}) *redis.BoolCmd
	// SCard returns the cardinality (size) of a set.
	SCard(ctx context.Context, key string) *redis.IntCmd

	// Sorted set operations
	// ZAdd adds one or more scored members to a sorted set.
	ZAdd(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd
	// ZRange returns members within the specified rank range.
	ZRange(ctx context.Context, key string, start, stop int64) *redis.StringSliceCmd
	// ZRangeByScore returns members whose scores fall in the provided range.
	ZRangeByScore(ctx context.Context, key string, opt *redis.ZRangeBy) *redis.StringSliceCmd
	// ZRem removes one or more members from a sorted set.
	ZRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd
	// ZCard returns the number of members in a sorted set.
	ZCard(ctx context.Context, key string) *redis.IntCmd
	// ZScore returns the score of a member in a sorted set.
	ZScore(ctx context.Context, key, member string) *redis.FloatCmd

	// General operations
	// Ping tests connectivity to Redis.
	Ping(ctx context.Context) *redis.StatusCmd
	// Close closes the underlying Redis client.
	Close() error
}

// clientAdapter adapts *redis.Client to implement the Client interface.
type clientAdapter struct {
	*redis.Client
}

// NewClientAdapter wraps a *redis.Client to implement the Client interface.
func NewClientAdapter(client *redis.Client) Client {
	return &clientAdapter{Client: client}
}
