package dto

import (
	"time"

	serverdto "github.com/minhnbnt/uptime-monitor/internal/features/server/dto"
)

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
	Server      serverdto.Server
	OntimeStats []OntimeStats
}
