package domain

type ServerStatus string

const (
	StatusOn  ServerStatus = "online"
	StatusOff ServerStatus = "offline"
)
