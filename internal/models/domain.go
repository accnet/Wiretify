package models

import (
	"time"

	"gorm.io/gorm"
)

type Domain struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	Name              string         `gorm:"not null;uniqueIndex" json:"name"`
	VerificationToken string         `json:"verification_token"`
	Status            string         `gorm:"default:'Pending'" json:"status"` // Pending, Active, Error
	LastVerifiedAt    *time.Time     `json:"last_verified_at"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}
