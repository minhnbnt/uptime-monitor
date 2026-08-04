package dto

import "time"

// BatchGetOntimeItem is the cache/request key for a single [server, day] pair.
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

// DayStats couples a day with its computed result, reusing DayResult so the
// no-data distinction lives in exactly one place. These DTOs are internal
// transport shapes (mapped to API/proto at the handler edge), so no JSON tags
// are needed here.
type DayStats struct {
	Date   time.Time
	Result DayResult
}

// ServerOntime holds the per-day stats of a single server. It is the single
// container used both by the batcher's internal aggregation and by the
// service layer (previously duplicated as BatchGetOntimeResponse).
type ServerOntime struct {
	ServerID uint
	DayStats []DayStats
}
