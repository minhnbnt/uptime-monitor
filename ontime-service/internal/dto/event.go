package dto

type ServerStatus string

const (
	ServerStatusOn  ServerStatus = "ON"
	ServerStatusOff ServerStatus = "OFF"
)

func (s ServerStatus) String() string { return string(s) }

type RecordEventRequest struct {
	ServerID uint
	Status   ServerStatus
}

type EndpointStatus struct {
	ServerID uint
	Status   ServerStatus
}

type StatusCount struct {
	Online  int64
	Offline int64
}
