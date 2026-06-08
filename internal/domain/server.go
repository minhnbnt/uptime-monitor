package domain

import (
	"time"

	"gorm.io/gorm"
)

type Status string

const (
	StatusActive Status = "active"
	StatusPaused Status = "paused"
)

type Server struct {
	gorm.Model
	Name        string    `gorm:"type:varchar(255);not null"`
	Status      Status    `gorm:"type:varchar(10);not null;default:active"`
	Endpoint    *Endpoint `gorm:"foreignKey:ServerID;references:ID"`
	CreatedBy   *User     `gorm:"foreignKey:CreatedByID"`
	CreatedByID uint      `gorm:"not null;index"`
}

func (Server) TableName() string {
	return "servers"
}

type Endpoint struct {
	gorm.Model
	ServerID     uint          `gorm:"not null;index"`
	URL          string        `gorm:"type:text;not null"`
	Status       Status        `gorm:"type:varchar(10);not null;default:active"`
	Interval     time.Duration `gorm:"type:bigint;not null;default:30000000000"`
	Timeout      time.Duration `gorm:"type:bigint;not null;default:10000000000"`
	Method       string        `gorm:"type:varchar(10);not null;default:GET"`
	ExpectedCode int           `gorm:"type:int;not null;default:200"`
}

func (Endpoint) TableName() string {
	return "endpoints"
}
