package dto

import "time"

type BatchGetOntimeItem struct {
	ServerID uint
	Date     time.Time
}

type OntimeStats struct {
	Date  time.Time
	Stats float64
}

type BatchGetOntimeResponse struct {
	ServerID uint
	Result   []OntimeStats
}

type ServerWithOntime struct {
	Server      Server
	OntimeStats []OntimeStats
}
