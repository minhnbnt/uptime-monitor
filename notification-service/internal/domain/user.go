package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID    uuid.UUID
	Email string
}

type ScheduleConfig struct {
	FromDate   time.Time
	ToDate     time.Time
	DigestTime string
}

type ScheduleInfo struct {
	Exists     bool
	FromDate   time.Time
	ToDate     time.Time
	DigestTime string
}
