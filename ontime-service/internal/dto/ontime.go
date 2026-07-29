package dto

import "time"

type BatchGetOntimeItem struct {
	ServerID uint
	Date     time.Time
}

type OntimeStats struct {
	Date  time.Time `json:"date"`
	Stats float64   `json:"stats"`
}

type BatchGetOntimeResponse struct {
	ServerID uint
	Result   []OntimeStats
}

type ServerOntime struct {
	ServerID    uint          `json:"server_id"`
	OntimeStats []OntimeStats `json:"ontime_stats"`
}
