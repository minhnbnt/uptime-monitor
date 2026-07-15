package domain

import "time"

type Server struct {
	ID   uint
	Name string
}

type OntimeStats struct {
	Date  time.Time
	Stats float64
}
