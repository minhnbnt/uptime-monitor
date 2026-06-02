package domain

import (
	"time"

	"github.com/google/uuid"
)

type ServerStatus string

const (
	StatusOn  ServerStatus = "ON"
	StatusOff ServerStatus = "OFF"
)

type ServerEvent struct {
	ID         uuid.UUID    `gorm:"type:uuid;primaryKey"`
	EndpointID uint         `gorm:"not null;index:idx_endpoint_time,priority:1"`
	Status     ServerStatus `gorm:"type:varchar(20);not null"`
	Time       time.Time    `gorm:"not null;index:idx_endpoint_time,priority:2"`
}

func (ServerEvent) TableName() string {
	return "server_events"
}
