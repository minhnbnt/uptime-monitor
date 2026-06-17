package domain

import (
	"time"

	"gorm.io/gorm"
)

type NotificationConfig struct {
	gorm.Model
	UserID     uint      `gorm:"uniqueIndex;not null"`
	Active     bool      `gorm:"not null;default:true"`
	FromDate   time.Time `gorm:"not null;type:date"`
	ToDate     time.Time `gorm:"not null;type:date"`
	DigestTime string    `gorm:"type:varchar(5);not null;default:08:00"`
}

func (NotificationConfig) TableName() string {
	return "notification_configs"
}
