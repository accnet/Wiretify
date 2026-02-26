package models

import (
	"time"

	"gorm.io/gorm"
)

type Endpoint struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	PeerID    uint           `gorm:"not null" json:"peer_id"`
	Peer      Peer           `json:"peer"`
	DomainID  uint           `gorm:"not null" json:"domain_id"`
	Domain    Domain         `json:"domain"`
	Subdomain string         `gorm:"not null" json:"subdomain"` // e.g. "tuupc"
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
