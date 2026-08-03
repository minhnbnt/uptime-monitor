package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ServerStatus string

const (
	StatusOn  ServerStatus = "ON"
	StatusOff ServerStatus = "OFF"
)

type Server struct {
	gorm.Model
	Name          string        `gorm:"type:varchar(255);not null;unique"`
	Namespace     string        `gorm:"type:varchar(255);not null"`
	Kind          string        `gorm:"type:varchar(50);not null"`
	ObjectID      string        `gorm:"type:varchar(255);not null"`
	ContainerName string        `gorm:"type:varchar(255);not null;default:''"`
	Interval      time.Duration `gorm:"type:bigint;not null;default:30000000000"`
	Timeout       time.Duration `gorm:"type:bigint;not null;default:10000000000"`
	CreatedByID   uuid.UUID     `gorm:"type:uuid;not null;index"`
	// Managed indicates the backing k8s object was created by this system
	// (via /k8s-objects) and may therefore be deleted through the API.
	Managed bool `gorm:"type:boolean;not null;default:false"`
}

func (Server) TableName() string {
	return "servers"
}
