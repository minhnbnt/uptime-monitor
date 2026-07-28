package domain

import (
	"time"

	"gorm.io/gorm"
)

type ServerStatus string

const (
	StatusOn  ServerStatus = "ON"
	StatusOff ServerStatus = "OFF"
)

type Server struct {
	gorm.Model
	Name          string        `gorm:"type:varchar(255);not null"`
	Namespace     string        `gorm:"type:varchar(255);not null"`
	Kind          string        `gorm:"type:varchar(50);not null"`
	ObjectID      string        `gorm:"type:varchar(255);not null"`
	ContainerName string        `gorm:"type:varchar(255);not null;default:''"`
	Interval      time.Duration `gorm:"type:bigint;not null;default:30000000000"`
	Timeout       time.Duration `gorm:"type:bigint;not null;default:10000000000"`
	CreatedByID   uint          `gorm:"not null;default:0;index"`
}

func (Server) TableName() string {
	return "servers"
}
