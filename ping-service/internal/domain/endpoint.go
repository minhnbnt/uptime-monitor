package domain

import "time"

// Endpoint shares its primary key with its owner server (endpoint.id == server.id).
type Endpoint struct {
	ID            uint
	URL           string
	Interval      time.Duration
	Timeout       time.Duration
	Method        string
	ExpectedCode  int
	BodyCheckExpr *string
}
