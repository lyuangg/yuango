// Package redis provides Redis client connection management.
package redis

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// MockClient is an in-memory mock implementation of the Client interface for testing.
type MockClient struct {
	mu     sync.RWMutex
	data   map[string]string
	hashes map[string]map[string]string
	lists  map[string][]string
	sets   map[string]map[string]bool
	sorted map[string]map[string]float64 // sorted sets: key -> member -> score
	expiry map[string]time.Time
	closed bool
}

// NewMockClient creates a new mock Redis client.
func NewMockClient() *MockClient {
	return &MockClient{
		data:   make(map[string]string),
		hashes: make(map[string]map[string]string),
		lists:  make(map[string][]string),
		sets:   make(map[string]map[string]bool),
		sorted: make(map[string]map[string]float64),
		expiry: make(map[string]time.Time),
	}
}

// Get implements Client.Get
func (m *MockClient) Get(ctx context.Context, key string) *redis.StringCmd {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cmd := redis.NewStringCmd(ctx, "get", key)

	// Check expiry
	if exp, ok := m.expiry[key]; ok && time.Now().After(exp) {
		delete(m.data, key)
		delete(m.expiry, key)
		cmd.SetErr(redis.Nil)
		return cmd
	}

	if val, ok := m.data[key]; ok {
		cmd.SetVal(val)
	} else {
		cmd.SetErr(redis.Nil)
	}
	return cmd
}

// Set implements Client.Set
func (m *MockClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	m.mu.Lock()
	defer m.mu.Unlock()

	cmd := redis.NewStatusCmd(ctx, "set", key, value)
	m.data[key] = value.(string)

	if expiration > 0 {
		m.expiry[key] = time.Now().Add(expiration)
	} else {
		delete(m.expiry, key)
	}

	cmd.SetVal("OK")
	return cmd
}

// Del implements Client.Del
func (m *MockClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	m.mu.Lock()
	defer m.mu.Unlock()

	cmd := redis.NewIntCmd(ctx, "del", keys)
	count := int64(0)

	for _, key := range keys {
		if _, ok := m.data[key]; ok {
			delete(m.data, key)
			delete(m.expiry, key)
			count++
		}
	}

	cmd.SetVal(count)
	return cmd
}

// Exists implements Client.Exists
func (m *MockClient) Exists(ctx context.Context, keys ...string) *redis.IntCmd {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cmd := redis.NewIntCmd(ctx, "exists", keys)
	count := int64(0)

	for _, key := range keys {
		// Check expiry
		if exp, ok := m.expiry[key]; ok && time.Now().After(exp) {
			continue
		}
		if _, ok := m.data[key]; ok {
			count++
		}
	}

	cmd.SetVal(count)
	return cmd
}

// Expire implements Client.Expire
func (m *MockClient) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	m.mu.Lock()
	defer m.mu.Unlock()

	cmd := redis.NewBoolCmd(ctx, "expire", key, expiration)

	if _, ok := m.data[key]; ok {
		if expiration > 0 {
			m.expiry[key] = time.Now().Add(expiration)
		} else {
			delete(m.expiry, key)
		}
		cmd.SetVal(true)
	} else {
		cmd.SetVal(false)
	}

	return cmd
}

// TTL implements Client.TTL
func (m *MockClient) TTL(ctx context.Context, key string) *redis.DurationCmd {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cmd := redis.NewDurationCmd(ctx, time.Second)

	if exp, ok := m.expiry[key]; ok {
		ttl := time.Until(exp)
		if ttl > 0 {
			cmd.SetVal(ttl)
		} else {
			cmd.SetVal(-1 * time.Second) // Expired
		}
	} else if _, ok := m.data[key]; ok {
		cmd.SetVal(-1 * time.Second) // No expiry
	} else {
		cmd.SetVal(-2 * time.Second) // Key doesn't exist
	}

	return cmd
}

// HGet implements Client.HGet
func (m *MockClient) HGet(ctx context.Context, key, field string) *redis.StringCmd {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cmd := redis.NewStringCmd(ctx, "hget", key, field)

	if hash, ok := m.hashes[key]; ok {
		if val, ok := hash[field]; ok {
			cmd.SetVal(val)
		} else {
			cmd.SetErr(redis.Nil)
		}
	} else {
		cmd.SetErr(redis.Nil)
	}

	return cmd
}

// HSet implements Client.HSet
func (m *MockClient) HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	m.mu.Lock()
	defer m.mu.Unlock()

	cmd := redis.NewIntCmd(ctx, "hset", key, values)

	if m.hashes[key] == nil {
		m.hashes[key] = make(map[string]string)
	}

	count := int64(0)
	for i := 0; i < len(values); i += 2 {
		field := values[i].(string)
		value := values[i+1].(string)
		if _, exists := m.hashes[key][field]; !exists {
			count++
		}
		m.hashes[key][field] = value
	}

	cmd.SetVal(count)
	return cmd
}

// HGetAll implements Client.HGetAll
func (m *MockClient) HGetAll(ctx context.Context, key string) *redis.MapStringStringCmd {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cmd := redis.NewMapStringStringCmd(ctx, "hgetall", key)

	if hash, ok := m.hashes[key]; ok {
		cmd.SetVal(hash)
	} else {
		cmd.SetVal(make(map[string]string))
	}

	return cmd
}

// HDel implements Client.HDel
func (m *MockClient) HDel(ctx context.Context, key string, fields ...string) *redis.IntCmd {
	m.mu.Lock()
	defer m.mu.Unlock()

	cmd := redis.NewIntCmd(ctx, "hdel", key, fields)
	count := int64(0)

	if hash, ok := m.hashes[key]; ok {
		for _, field := range fields {
			if _, exists := hash[field]; exists {
				delete(hash, field)
				count++
			}
		}
		if len(hash) == 0 {
			delete(m.hashes, key)
		}
	}

	cmd.SetVal(count)
	return cmd
}

// LPush implements Client.LPush
func (m *MockClient) LPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	m.mu.Lock()
	defer m.mu.Unlock()

	cmd := redis.NewIntCmd(ctx, "lpush", key, values)

	if m.lists[key] == nil {
		m.lists[key] = make([]string, 0)
	}

	for i := len(values) - 1; i >= 0; i-- {
		var strVal string
		switch v := values[i].(type) {
		case string:
			strVal = v
		case []byte:
			strVal = string(v)
		default:
			strVal = fmt.Sprintf("%v", v)
		}
		m.lists[key] = append([]string{strVal}, m.lists[key]...)
	}

	cmd.SetVal(int64(len(m.lists[key])))
	return cmd
}

// RPush implements Client.RPush
func (m *MockClient) RPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	m.mu.Lock()
	defer m.mu.Unlock()

	cmd := redis.NewIntCmd(ctx, "rpush", key, values)

	if m.lists[key] == nil {
		m.lists[key] = make([]string, 0)
	}

	for _, val := range values {
		var strVal string
		switch v := val.(type) {
		case string:
			strVal = v
		case []byte:
			strVal = string(v)
		default:
			strVal = fmt.Sprintf("%v", v)
		}
		m.lists[key] = append(m.lists[key], strVal)
	}

	cmd.SetVal(int64(len(m.lists[key])))
	return cmd
}

// LPop implements Client.LPop
func (m *MockClient) LPop(ctx context.Context, key string) *redis.StringCmd {
	m.mu.Lock()
	defer m.mu.Unlock()

	cmd := redis.NewStringCmd(ctx, "lpop", key)

	if list, ok := m.lists[key]; ok && len(list) > 0 {
		val := list[0]
		m.lists[key] = list[1:]
		cmd.SetVal(val)
	} else {
		cmd.SetErr(redis.Nil)
	}

	return cmd
}

// RPop implements Client.RPop
func (m *MockClient) RPop(ctx context.Context, key string) *redis.StringCmd {
	m.mu.Lock()
	defer m.mu.Unlock()

	cmd := redis.NewStringCmd(ctx, "rpop", key)

	if list, ok := m.lists[key]; ok && len(list) > 0 {
		val := list[len(list)-1]
		m.lists[key] = list[:len(list)-1]
		cmd.SetVal(val)
	} else {
		cmd.SetErr(redis.Nil)
	}

	return cmd
}

// LLen implements Client.LLen
func (m *MockClient) LLen(ctx context.Context, key string) *redis.IntCmd {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cmd := redis.NewIntCmd(ctx, "llen", key)

	if list, ok := m.lists[key]; ok {
		cmd.SetVal(int64(len(list)))
	} else {
		cmd.SetVal(0)
	}

	return cmd
}

// LRange implements Client.LRange
func (m *MockClient) LRange(ctx context.Context, key string, start, stop int64) *redis.StringSliceCmd {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cmd := redis.NewStringSliceCmd(ctx, "lrange", key, start, stop)

	if list, ok := m.lists[key]; ok {
		length := int64(len(list))
		if start < 0 {
			start = length + start
		}
		if stop < 0 {
			stop = length + stop
		}
		if start < 0 {
			start = 0
		}
		if stop >= length {
			stop = length - 1
		}
		if start <= stop {
			cmd.SetVal(list[start : stop+1])
		} else {
			cmd.SetVal([]string{})
		}
	} else {
		cmd.SetVal([]string{})
	}

	return cmd
}

// SAdd implements Client.SAdd
func (m *MockClient) SAdd(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	m.mu.Lock()
	defer m.mu.Unlock()

	cmd := redis.NewIntCmd(ctx, "sadd", key, members)

	if m.sets[key] == nil {
		m.sets[key] = make(map[string]bool)
	}

	count := int64(0)
	for _, member := range members {
		memberStr := member.(string)
		if !m.sets[key][memberStr] {
			m.sets[key][memberStr] = true
			count++
		}
	}

	cmd.SetVal(count)
	return cmd
}

// SMembers implements Client.SMembers
func (m *MockClient) SMembers(ctx context.Context, key string) *redis.StringSliceCmd {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cmd := redis.NewStringSliceCmd(ctx, "smembers", key)

	if set, ok := m.sets[key]; ok {
		members := make([]string, 0, len(set))
		for member := range set {
			members = append(members, member)
		}
		cmd.SetVal(members)
	} else {
		cmd.SetVal([]string{})
	}

	return cmd
}

// SIsMember implements Client.SIsMember
func (m *MockClient) SIsMember(ctx context.Context, key string, member interface{}) *redis.BoolCmd {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cmd := redis.NewBoolCmd(ctx, "sismember", key, member)

	if set, ok := m.sets[key]; ok {
		cmd.SetVal(set[member.(string)])
	} else {
		cmd.SetVal(false)
	}

	return cmd
}

// SCard implements Client.SCard
func (m *MockClient) SCard(ctx context.Context, key string) *redis.IntCmd {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cmd := redis.NewIntCmd(ctx, "scard", key)

	if set, ok := m.sets[key]; ok {
		cmd.SetVal(int64(len(set)))
	} else {
		cmd.SetVal(0)
	}

	return cmd
}

// ZAdd implements Client.ZAdd
func (m *MockClient) ZAdd(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd {
	m.mu.Lock()
	defer m.mu.Unlock()

	cmd := redis.NewIntCmd(ctx, "zadd", key, members)
	count := int64(0)

	if m.sorted[key] == nil {
		m.sorted[key] = make(map[string]float64)
	}

	for _, member := range members {
		memberStr := ""
		switch v := member.Member.(type) {
		case string:
			memberStr = v
		case []byte:
			memberStr = string(v)
		default:
			memberStr = fmt.Sprintf("%v", v)
		}
		if _, exists := m.sorted[key][memberStr]; !exists {
			count++
		}
		m.sorted[key][memberStr] = member.Score
	}

	cmd.SetVal(count)
	return cmd
}

// ZRange implements Client.ZRange
func (m *MockClient) ZRange(ctx context.Context, key string, start, stop int64) *redis.StringSliceCmd {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cmd := redis.NewStringSliceCmd(ctx, "zrange", key, start, stop)

	if sorted, ok := m.sorted[key]; ok {
		// Convert map to sorted slice
		type memberScore struct {
			member string
			score  float64
		}
		var items []memberScore
		for member, score := range sorted {
			items = append(items, memberScore{member: member, score: score})
		}
		// Sort by score (simple implementation)
		for i := 0; i < len(items)-1; i++ {
			for j := i + 1; j < len(items); j++ {
				if items[i].score > items[j].score {
					items[i], items[j] = items[j], items[i]
				}
			}
		}
		// Apply range
		var results []string
		length := int64(len(items))
		if start < 0 {
			start = length + start
		}
		if stop < 0 {
			stop = length + stop
		}
		if start < 0 {
			start = 0
		}
		if stop >= length {
			stop = length - 1
		}
		for i := start; i <= stop && i < length; i++ {
			results = append(results, items[i].member)
		}
		cmd.SetVal(results)
	} else {
		cmd.SetVal([]string{})
	}

	return cmd
}

// ZRangeByScore implements Client.ZRangeByScore
func (m *MockClient) ZRangeByScore(ctx context.Context, key string, opt *redis.ZRangeBy) *redis.StringSliceCmd {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cmd := redis.NewStringSliceCmd(ctx, "zrangebyscore", key, opt)

	if sorted, ok := m.sorted[key]; ok {
		var results []string
		// Simple implementation: return all members (real implementation would parse Min/Max)
		for member := range sorted {
			results = append(results, member)
		}
		cmd.SetVal(results)
	} else {
		cmd.SetVal([]string{})
	}

	return cmd
}

// ZRem implements Client.ZRem
func (m *MockClient) ZRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	m.mu.Lock()
	defer m.mu.Unlock()

	cmd := redis.NewIntCmd(ctx, "zrem", key, members)
	count := int64(0)

	if sorted, ok := m.sorted[key]; ok {
		for _, member := range members {
			memberStr := ""
			switch v := member.(type) {
			case string:
				memberStr = v
			case []byte:
				memberStr = string(v)
			default:
				memberStr = fmt.Sprintf("%v", v)
			}
			if _, exists := sorted[memberStr]; exists {
				delete(sorted, memberStr)
				count++
			}
		}
		if len(sorted) == 0 {
			delete(m.sorted, key)
		}
	}

	cmd.SetVal(count)
	return cmd
}

// ZCard implements Client.ZCard
func (m *MockClient) ZCard(ctx context.Context, key string) *redis.IntCmd {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cmd := redis.NewIntCmd(ctx, "zcard", key)

	if sorted, ok := m.sorted[key]; ok {
		cmd.SetVal(int64(len(sorted)))
	} else {
		cmd.SetVal(0)
	}

	return cmd
}

// ZScore implements Client.ZScore
func (m *MockClient) ZScore(ctx context.Context, key, member string) *redis.FloatCmd {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cmd := redis.NewFloatCmd(ctx, "zscore", key, member)

	if sorted, ok := m.sorted[key]; ok {
		if score, exists := sorted[member]; exists {
			cmd.SetVal(score)
		} else {
			cmd.SetErr(redis.Nil)
		}
	} else {
		cmd.SetErr(redis.Nil)
	}

	return cmd
}

// Ping implements Client.Ping
func (m *MockClient) Ping(ctx context.Context) *redis.StatusCmd {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cmd := redis.NewStatusCmd(ctx, "ping")
	if m.closed {
		cmd.SetErr(redis.ErrClosed)
	} else {
		cmd.SetVal("PONG")
	}
	return cmd
}

// Close implements Client.Close
func (m *MockClient) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.closed = true
	return nil
}
