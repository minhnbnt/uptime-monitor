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
	// Timezone is the IANA name digest_time is interpreted in.
	// Empty means UTC.
	Timezone string
}

type ScheduleInfo struct {
	Exists     bool
	FromDate   time.Time
	ToDate     time.Time
	DigestTime string
	// Timezone is the IANA name digest_time is interpreted in.
	// Always resolved (defaults to UTC) so GET round-trips.
	Timezone string
}
