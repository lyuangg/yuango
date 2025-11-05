// Package repository provides repository interfaces and implementations for data access.
package repository

import (
	"github.com/lyuangg/yuango/internal/model"
	"gorm.io/gorm"
)

// UserRepository is a repository for User model.
// This is an example showing how to use the generic repository.
type UserRepository interface {
	Repository[model.User]
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(db *gorm.DB) UserRepository {
	return NewGormRepository[model.User](db)
}
