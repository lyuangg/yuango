// Package repository provides repository interfaces and implementations for data access.
package repository

import (
	"context"
)

// Model represents a database model with an ID field.
type Model interface {
	GetID() uint
}

// Repository defines the common database operations.
type Repository[T Model] interface {
	// Create creates a new record.
	Create(ctx context.Context, entity *T) error

	// Update updates an existing record.
	Update(ctx context.Context, entity *T) error

	// Delete deletes a record by ID.
	Delete(ctx context.Context, id uint) error

	// FindByID finds a record by ID.
	FindByID(ctx context.Context, id uint) (*T, error)

	// FindByIDs finds records by IDs.
	FindByIDs(ctx context.Context, ids []uint) ([]*T, error)

	// CreateBatch creates multiple records in batch.
	CreateBatch(ctx context.Context, entities []*T) error

	// UpdateByID updates a record by ID with the given map of updates.
	// This is useful when you only have the ID and partial data.
	UpdateByID(ctx context.Context, id uint, updates map[string]interface{}) error

	// UpdateWhere updates records matching the given conditions with the given map of updates.
	// If conditions is empty or nil, no update will be performed (safety measure).
	UpdateWhere(ctx context.Context, conditions map[string]interface{}, updates map[string]interface{}) error

	// DeleteWhere deletes records matching the given conditions.
	// If conditions is empty or nil, no deletion will be performed (safety measure).
	DeleteWhere(ctx context.Context, conditions map[string]interface{}) error

	// FindWhereMap finds records matching the given map conditions.
	// This is a convenience method for simple equality queries.
	// orderBy specifies the sort order, e.g., "created_at DESC" or "created_at DESC, id ASC".
	// If orderBy is empty, no sorting is applied.
	FindWhereMap(ctx context.Context, conditions map[string]interface{}, orderBy string) ([]*T, error)

	// FindPage finds records matching the given conditions with pagination.
	// Returns the records, total count, and error.
	// Page starts from 1. If page <= 0, it defaults to 1.
	// If pageSize <= 0, it defaults to 10.
	// orderBy specifies the sort order, e.g., "created_at DESC" or "created_at DESC, id ASC".
	// If orderBy is empty, no sorting is applied.
	FindPage(ctx context.Context, conditions map[string]interface{}, page, pageSize int, orderBy string) ([]*T, int64, error)

	// CountWhere counts records matching the given conditions.
	// If conditions is empty or nil, counts all records.
	CountWhere(ctx context.Context, conditions map[string]interface{}) (int64, error)
}
