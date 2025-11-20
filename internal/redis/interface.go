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
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Exists(ctx context.Context, keys ...string) *redis.IntCmd
	Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
	TTL(ctx context.Context, key string) *redis.DurationCmd

	// Hash operations
	HGet(ctx context.Context, key, field string) *redis.StringCmd
	HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
	HGetAll(ctx context.Context, key string) *redis.MapStringStringCmd
	HDel(ctx context.Context, key string, fields ...string) *redis.IntCmd

	// List operations
	LPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
	RPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
	LPop(ctx context.Context, key string) *redis.StringCmd
	RPop(ctx context.Context, key string) *redis.StringCmd
	LLen(ctx context.Context, key string) *redis.IntCmd
	LRange(ctx context.Context, key string, start, stop int64) *redis.StringSliceCmd

	// Set operations
	SAdd(ctx context.Context, key string, members ...interface{}) *redis.IntCmd
	SMembers(ctx context.Context, key string) *redis.StringSliceCmd
	SIsMember(ctx context.Context, key string, member interface{}) *redis.BoolCmd
	SCard(ctx context.Context, key string) *redis.IntCmd

	// Sorted set operations
	ZAdd(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd
	ZRange(ctx context.Context, key string, start, stop int64) *redis.StringSliceCmd
	ZScore(ctx context.Context, key, member string) *redis.FloatCmd

	// General operations
	Ping(ctx context.Context) *redis.StatusCmd
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
