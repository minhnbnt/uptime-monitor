package dto

import "time"

type BatchGetOntimeItem struct {
	ServerID uint      `validate:"required"`
	Date     time.Time `validate:"required"`
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
