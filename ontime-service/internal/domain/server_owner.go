package domain

import (
	"time"

	"gorm.io/gorm"
)

type ServerOwner struct {
	ServerID  uint           `gorm:"primaryKey"`
	UserID    uint           `gorm:"not null"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
	CreatedAt time.Time      `gorm:"not null;autoCreateTime"`
	UpdatedAt time.Time      `gorm:"not null;autoUpdateTime"`
}

func (ServerOwner) TableName() string {
	return "server_owners"
}
