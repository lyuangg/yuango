// Package queue provides a queue interface and implementations.
package queue

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lyuangg/yuango/internal/redis"
	redisv9 "github.com/redis/go-redis/v9"
)

// RedisQueue is a Redis-based implementation of the Queue interface.
// It uses Redis List for immediate messages and Sorted Set for delayed messages.
type RedisQueue struct {
	client redis.Client
	prefix string // Key prefix for queue names
}

type storedMessage struct {
	ID      string `json:"id"`
	Payload []byte `json:"payload"`
}

// NewRedisQueue creates a new Redis-based queue.
func NewRedisQueue(client redis.Client) *RedisQueue {
	return &RedisQueue{
		client: client,
		prefix: "queue",
	}
}

// NewRedisQueueWithPrefix creates a new Redis-based queue with a custom prefix.
func NewRedisQueueWithPrefix(client redis.Client, prefix string) *RedisQueue {
	return &RedisQueue{
		client: client,
		prefix: prefix,
	}
}

// getQueueKey returns the Redis key for the queue.
func (r *RedisQueue) getQueueKey(queueName string) string {
	return fmt.Sprintf("%s:%s", r.prefix, queueName)
}

// getDelayedKey returns the Redis key for the delayed queue (sorted set).
func (r *RedisQueue) getDelayedKey(queueName string) string {
	return fmt.Sprintf("%s:%s:delayed", r.prefix, queueName)
}

// Push adds a message to the queue immediately.
func (r *RedisQueue) Push(ctx context.Context, queueName string, payload []byte) error {
	key := r.getQueueKey(queueName)
	msg := storedMessage{
		ID:      generateMessageID(),
		Payload: payload,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("queue marshal message failed: %w", err)
	}
	if err := r.client.RPush(ctx, key, string(data)).Err(); err != nil {
		return fmt.Errorf("queue push failed: %w", err)
	}
	return nil
}

// PushDelayed adds a message to the queue with a delay.
func (r *RedisQueue) PushDelayed(ctx context.Context, queueName string, payload []byte, delay time.Duration) error {
	if delay < 0 {
		return ErrInvalidDelay
	}

	// Calculate the execution time (use nanosecond precision to avoid same-second collision)
	executeAt := time.Now().Add(delay)
	score := float64(executeAt.UnixNano())

	stored := storedMessage{
		ID:      generateMessageID(),
		Payload: payload,
	}

	// Create message with metadata
	msg := delayedMessage{
		Message:   stored,
		ExecuteAt: executeAt,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("queue marshal delayed message failed: %w", err)
	}

	// Add to sorted set with score = execution timestamp
	key := r.getDelayedKey(queueName)
	member := redisv9.Z{
		Score:  score,
		Member: string(data),
	}
	if err := r.client.ZAdd(ctx, key, member).Err(); err != nil {
		return fmt.Errorf("queue push delayed failed: %w", err)
	}

	return nil
}

// Pop removes and returns a message from the queue.
func (r *RedisQueue) Pop(ctx context.Context, queueName string) (*Message, error) {
	return r.PopWithTimeout(ctx, queueName, 0)
}

// PopWithTimeout removes and returns a message from the queue with a timeout.
func (r *RedisQueue) PopWithTimeout(ctx context.Context, queueName string, timeout time.Duration) (*Message, error) {
	key := r.getQueueKey(queueName)
	blocking := timeout > 0
	deadline := time.Time{}
	if blocking {
		deadline = time.Now().Add(timeout)
	}

	const pollInterval = 100 * time.Millisecond

	for {
		if err := r.moveDelayedMessages(ctx, queueName); err != nil {
			return nil, fmt.Errorf("queue move delayed messages failed: %w", err)
		}

		result, err := r.client.LPop(ctx, key).Result()
		if err == nil {
			var stored storedMessage
			if err := json.Unmarshal([]byte(result), &stored); err != nil {
				return nil, fmt.Errorf("queue unmarshal message failed: %w", err)
			}
			return &Message{
				ID:      stored.ID,
				Payload: stored.Payload,
			}, nil
		}
		if err != redisv9.Nil {
			return nil, fmt.Errorf("queue pop failed: %w", err)
		}

		if !blocking {
			return nil, ErrQueueEmpty
		}

		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil, ErrQueueEmpty
		}

		wait := pollInterval
		if remaining < wait {
			wait = remaining
		}

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

// Size returns the number of messages in the queue.
func (r *RedisQueue) Size(ctx context.Context, queueName string) (int64, error) {
	queueKey := r.getQueueKey(queueName)
	delayedKey := r.getDelayedKey(queueName)

	// Get size of immediate queue
	queueSize, err := r.client.LLen(ctx, queueKey).Result()
	if err != nil {
		return 0, fmt.Errorf("queue get size failed: %w", err)
	}

	// Get size of delayed queue (count messages ready or not)
	delayedSize, err := r.client.ZCard(ctx, delayedKey).Result()
	if err != nil && err != redisv9.Nil {
		return 0, fmt.Errorf("queue get delayed size failed: %w", err)
	}

	return queueSize + delayedSize, nil
}

// Close closes the queue.
func (r *RedisQueue) Close() error {
	return r.client.Close()
}

// moveDelayedMessages moves ready delayed messages from sorted set to the main queue.
func (r *RedisQueue) moveDelayedMessages(ctx context.Context, queueName string) error {
	delayedKey := r.getDelayedKey(queueName)
	queueKey := r.getQueueKey(queueName)
	now := time.Now().UnixNano()

	// Get all messages with score <= now (ready to execute)
	messages, err := r.client.ZRange(ctx, delayedKey, 0, -1).Result()
	if err != nil && err != redisv9.Nil {
		return fmt.Errorf("queue get delayed messages failed: %w", err)
	}

	var readyMsgData []string // Store original message data for deletion

	for _, msgData := range messages {
		var msg delayedMessage
		if err := json.Unmarshal([]byte(msgData), &msg); err != nil {
			// Skip invalid messages
			continue
		}

		if msg.ExecuteAt.UnixNano() <= now {
			// Move to main queue
			data, err := json.Marshal(msg.Message)
			if err != nil {
				return fmt.Errorf("queue marshal stored message failed: %w", err)
			}
			if err := r.client.RPush(ctx, queueKey, string(data)).Err(); err != nil {
				return fmt.Errorf("queue move delayed message failed: %w", err)
			}
			// Mark for deletion
			readyMsgData = append(readyMsgData, msgData)
		}
	}

	if len(readyMsgData) == 0 {
		return nil
	}

	// Remove moved messages from delayed queue
	for _, msgData := range readyMsgData {
		if err := r.client.ZRem(ctx, delayedKey, msgData).Err(); err != nil && err != redisv9.Nil {
			return fmt.Errorf("queue remove delayed message failed: %w", err)
		}
	}

	return nil
}

// delayedMessage represents a delayed message with metadata.
type delayedMessage struct {
	Message   storedMessage `json:"message"`
	ExecuteAt time.Time     `json:"execute_at"`
}

// generateMessageID generates a unique message ID.
func generateMessageID() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("msg_%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("msg_%d_%s", time.Now().UnixNano(), hex.EncodeToString(buf[:]))
}
