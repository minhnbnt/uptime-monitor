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
	ID       uuid.UUID    `json:"id"`
	ServerID uint         `json:"server_id"`
	Status   ServerStatus `json:"status"`
	Time     time.Time    `json:"time"`
}
