package domain

import "time"

type User struct {
	ID       uint
	Email    string
	Username string
	Name     string
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
