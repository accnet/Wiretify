package models

import (
	"time"

	"gorm.io/gorm"
)

type PortForward struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	PublicPort int            `gorm:"not null" json:"public_port"`
	TargetNode string         `gorm:"not null" json:"target_node"`
	TargetPort int            `gorm:"not null" json:"target_port"`
	Protocol   string         `gorm:"not null" json:"protocol"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}
