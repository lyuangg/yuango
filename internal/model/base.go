// Package model provides database models.
package model

import (
	"time"
)

// BaseModel provides common fields for all models.
type BaseModel struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GetID returns the ID of the model.
func (b BaseModel) GetID() uint {
	return b.ID
}
