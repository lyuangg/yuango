// Package repository provides repository interfaces and implementations for data access.
package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

// GormRepository is a generic repository implementation using GORM.
type GormRepository[T Model] struct {
	db *gorm.DB
}

// NewGormRepository creates a new GORM repository.
func NewGormRepository[T Model](db *gorm.DB) *GormRepository[T] {
	return &GormRepository[T]{
		db: db,
	}
}

// Create creates a new record.
func (r *GormRepository[T]) Create(ctx context.Context, entity *T) error {
	if err := r.db.WithContext(ctx).Create(entity).Error; err != nil {
		return err
	}
	return nil
}

// Update updates an existing record.
func (r *GormRepository[T]) Update(ctx context.Context, entity *T) error {
	if err := r.db.WithContext(ctx).Save(entity).Error; err != nil {
		return err
	}
	return nil
}

// Delete deletes a record by ID.
func (r *GormRepository[T]) Delete(ctx context.Context, id uint) error {
	var entity T
	if err := r.db.WithContext(ctx).First(&entity, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}

	if err := r.db.WithContext(ctx).Delete(&entity).Error; err != nil {
		return err
	}
	return nil
}

// FindByID finds a record by ID.
func (r *GormRepository[T]) FindByID(ctx context.Context, id uint) (*T, error) {
	var entity T
	if err := r.db.WithContext(ctx).First(&entity, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &entity, nil
}

// FindByIDs finds records by IDs.
func (r *GormRepository[T]) FindByIDs(ctx context.Context, ids []uint) ([]*T, error) {
	if len(ids) == 0 {
		return []*T{}, nil
	}
	var entities []*T
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&entities).Error; err != nil {
		return nil, err
	}
	return entities, nil
}

// CreateBatch creates multiple records in batch.
func (r *GormRepository[T]) CreateBatch(ctx context.Context, entities []*T) error {
	if len(entities) == 0 {
		return nil
	}
	if err := r.db.WithContext(ctx).Create(entities).Error; err != nil {
		return err
	}
	return nil
}

// UpdateByID updates a record by ID with the given map of updates.
// This is useful when you only have the ID and partial data.
func (r *GormRepository[T]) UpdateByID(ctx context.Context, id uint, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	var entity T
	if err := r.db.WithContext(ctx).Model(&entity).Where("id = ?", id).Updates(updates).Error; err != nil {
		return err
	}
	return nil
}

// UpdateWhere updates records matching the given conditions with the given map of updates.
// If conditions is empty or nil, no update will be performed (safety measure).
func (r *GormRepository[T]) UpdateWhere(ctx context.Context, conditions map[string]interface{}, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	if len(conditions) == 0 {
		return nil
	}
	var entity T
	db := r.db.WithContext(ctx).Model(&entity).Where(conditions)
	if err := db.Updates(updates).Error; err != nil {
		return err
	}
	return nil
}

// DeleteWhere deletes records matching the given conditions.
// If conditions is empty or nil, no deletion will be performed (safety measure).
func (r *GormRepository[T]) DeleteWhere(ctx context.Context, conditions map[string]interface{}) error {
	if len(conditions) == 0 {
		return nil
	}
	var entity T
	db := r.db.WithContext(ctx).Where(conditions)
	if err := db.Delete(&entity).Error; err != nil {
		return err
	}
	return nil
}

// FindWhereMap finds records matching the given map conditions.
// This is a convenience method for simple equality queries.
// orderBy specifies the sort order, e.g., "created_at DESC" or "created_at DESC, id ASC".
// If orderBy is empty, no sorting is applied.
func (r *GormRepository[T]) FindWhereMap(ctx context.Context, conditions map[string]interface{}, orderBy string) ([]*T, error) {
	var entities []*T
	db := r.db.WithContext(ctx)
	if len(conditions) > 0 {
		db = db.Where(conditions)
	}
	if orderBy != "" {
		db = db.Order(orderBy)
	}
	if err := db.Find(&entities).Error; err != nil {
		return nil, err
	}
	return entities, nil
}

// FindPage finds records matching the given conditions with pagination.
// Returns the records, total count, and error.
// Page starts from 1. If page <= 0, it defaults to 1.
// If pageSize <= 0, it defaults to 10.
// orderBy specifies the sort order, e.g., "created_at DESC" or "created_at DESC, id ASC".
// If orderBy is empty, no sorting is applied.
func (r *GormRepository[T]) FindPage(ctx context.Context, conditions map[string]interface{}, page, pageSize int, orderBy string) ([]*T, int64, error) {
	var entity T
	db := r.db.WithContext(ctx).Model(&entity)

	// Apply conditions
	if len(conditions) > 0 {
		db = db.Where(conditions)
	}

	// Count total
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Validate and set defaults for pagination
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Apply sorting
	if orderBy != "" {
		db = db.Order(orderBy)
	}

	// Query with pagination
	var entities []*T
	if err := db.Offset(offset).Limit(pageSize).Find(&entities).Error; err != nil {
		return nil, 0, err
	}

	return entities, total, nil
}

// CountWhere counts records matching the given conditions.
// If conditions is empty or nil, counts all records.
func (r *GormRepository[T]) CountWhere(ctx context.Context, conditions map[string]interface{}) (int64, error) {
	var entity T
	db := r.db.WithContext(ctx).Model(&entity)

	// Apply conditions if provided
	if len(conditions) > 0 {
		db = db.Where(conditions)
	}

	// Count records
	var count int64
	if err := db.Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}
