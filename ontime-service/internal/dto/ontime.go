package dto

import "time"

type BatchGetOntimeItem struct {
	ServerID uint
	Date     time.Time
}

// DayResult is the internal unit passed between the calculator, the cache,
// and the batcher for a single [server, day] pair. HasData is carried
// alongside Uptime everywhere this value travels so "no data yet" can never
// get silently coerced into "0%" at any layer — including in Redis.
type DayResult struct {
	HasData bool
	Uptime  float64
}

// OntimeStats is the public, per-day shape returned to API callers.
type OntimeStats struct {
	Date    time.Time `json:"date"`
	Stats   float64   `json:"stats"`
	HasData bool      `json:"has_data"`
}

type BatchGetOntimeResponse struct {
	ServerID uint
	Result   []OntimeStats
}

type ServerOntime struct {
	ServerID    uint          `json:"server_id"`
	OntimeStats []OntimeStats `json:"ontime_stats"`
}
