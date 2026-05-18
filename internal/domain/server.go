package domain

import (
	"github.com/google/uuid"
	"time"
)

type Status string

const (
	StatusActive Status = "active"
	StatusPaused Status = "paused"
)

type Server struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name      string    `gorm:"type:varchar(255);not null"`
	URL       string    `gorm:"type:text;not null"`
	Status    Status    `gorm:"type:varchar(10);not null;default:active"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (Server) TableName() string {
	return "servers"
}
