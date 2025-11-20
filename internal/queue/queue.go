// Package queue provides a queue interface and implementations.
package queue

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrQueueEmpty indicates that the queue is empty.
	ErrQueueEmpty = errors.New("queue: queue is empty")
	// ErrInvalidDelay indicates that the delay duration is invalid.
	ErrInvalidDelay = errors.New("queue: invalid delay duration")
)

// Message represents a message in the queue.
type Message struct {
	ID      string    // Unique message ID
	Payload []byte    // Message payload
	Delay   time.Time // Scheduled execution time (for delayed messages)
}

// Queue defines the interface for queue operations.
type Queue interface {
	// Push adds a message to the queue immediately.
	Push(ctx context.Context, queueName string, payload []byte) error

	// PushDelayed adds a message to the queue with a delay.
	// The message will be available after the specified delay duration.
	PushDelayed(ctx context.Context, queueName string, payload []byte, delay time.Duration) error

	// Pop removes and returns a message from the queue.
	// It blocks until a message is available or the context is cancelled.
	// Returns ErrQueueEmpty if the queue is empty and context is not cancelled.
	Pop(ctx context.Context, queueName string) (*Message, error)

	// PopWithTimeout removes and returns a message from the queue with a timeout.
	// It blocks until a message is available, timeout expires, or context is cancelled.
	PopWithTimeout(ctx context.Context, queueName string, timeout time.Duration) (*Message, error)

	// Size returns the number of messages in the queue (including delayed messages).
	Size(ctx context.Context, queueName string) (int64, error)

	// Close closes the queue and releases resources.
	Close() error
}
