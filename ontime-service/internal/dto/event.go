package dto

import "time"

type ServerStatus string

const (
	ServerStatusOn  ServerStatus = "ON"
	ServerStatusOff ServerStatus = "OFF"
)

func (s ServerStatus) String() string { return string(s) }

type RecordEventRequest struct {
	EndpointID uint
	Status     ServerStatus
	Time       time.Time
}

type EndpointStatus struct {
	EndpointID uint
	Status     ServerStatus
}

type StatusCount struct {
	Online  int64
	Offline int64
}
