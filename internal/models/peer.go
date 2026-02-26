package models

import (
	"time"

	"gorm.io/gorm"
)

type Peer struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"uniqueIndex;not null" json:"name"`
	PublicKey   string         `gorm:"uniqueIndex;not null" json:"public_key"`
	PrivateKey  string         `json:"private_key,omitempty"` // Chỉ lưu nếu app tự gen
	AllowedIPs    string         `json:"allowed_ips"`
	Endpoints     string         `json:"endpoint"`
	UseAsExitNode bool           `gorm:"default:true" json:"use_as_exit_node"`
	Enabled       bool           `gorm:"default:true" json:"enabled"`
	Icon          string         `json:"icon"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// Runtime stats (không lưu DB)
	Connected       bool      `gorm:"-" json:"connected"`
	RxBytes         int64     `gorm:"-" json:"rx_bytes"`
	TxBytes         int64     `gorm:"-" json:"tx_bytes"`
	LastHandshake   time.Time `gorm:"-" json:"last_handshake"`
}

type Setting struct {
	Key   string `gorm:"primaryKey"`
	Value string
}
