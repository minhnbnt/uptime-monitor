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

// asStatus converts a raw DB status string into a known ServerStatus.
// Anything other than exactly "ON"/"OFF" — including the empty string that
// comes back from a LEFT JOIN with no matching row — is reported unknown.
// This is the single place that decides what counts as "real" status.
func ToServerStatus(raw string) (ServerStatus, bool) {

	switch ServerStatus(raw) {
	case StatusOn, StatusOff:
		return ServerStatus(raw), true

	default:
		return "", false
	}
}

type ServerEvent struct {
	ID       uuid.UUID    `gorm:"type:uuid;primaryKey"`
	ServerID uint         `gorm:"not null;index:idx_server_time,priority:1"`
	Status   ServerStatus `gorm:"type:varchar(20);not null"`
	Time     time.Time    `gorm:"not null;index:idx_server_time,priority:2"`
}

func (ServerEvent) TableName() string {
	return "server_events"
}
