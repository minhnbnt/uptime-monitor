package domain

import (
	"time"

	"gorm.io/gorm"
)

type Server struct {
	gorm.Model
	Name        string    `gorm:"type:varchar(255);not null"`
	Endpoint    *Endpoint `gorm:"foreignKey:ID;references:ID"`
	CreatedByID uint      `gorm:"not null;default:0"`
}

func (Server) TableName() string {
	return "servers"
}

// Endpoint shares its primary key with its owner server (endpoint.id == server.id),
// a strict 1-1. ponytail: collapse the pair into one identifier; split tables if a
// server ever needs multiple endpoints.
type Endpoint struct {
	ID            uint          `gorm:"primaryKey"`
	URL           string        `gorm:"type:text;not null"`
	Interval      time.Duration `gorm:"type:bigint;not null;default:30000000000"`
	Timeout       time.Duration `gorm:"type:bigint;not null;default:10000000000"`
	Method        string        `gorm:"type:varchar(10);not null;default:GET"`
	ExpectedCode  int           `gorm:"type:int;not null;default:200"`
	BodyCheckExpr *string       `gorm:"type:text"`
}

func (Endpoint) TableName() string {
	return "endpoints"
}
