package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/domain"
)

type Server struct {
	ID            uint
	Name          string
	Namespace     string
	Kind          string
	ObjectID      string
	ContainerName string
	Interval      time.Duration
	Timeout       time.Duration
	CreatedByID   uuid.UUID
	MonitorStatus domain.ServerStatus
	HttpConfig    *HttpConfig
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func ServerFromDomain(s domain.Server) Server {
	return Server{
		ID:            s.ID,
		Name:          s.Name,
		Namespace:     s.Namespace,
		Kind:          s.Kind,
		ObjectID:      s.ObjectID,
		ContainerName: s.ContainerName,
		Interval:      s.Interval,
		Timeout:       s.Timeout,
		CreatedByID:   s.CreatedByID,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}
}

type CreateServerRequest struct {
	Name          string
	Namespace     string
	Kind          string
	ObjectID      string
	ContainerName string
	Interval      time.Duration
	Timeout       time.Duration
	HttpConfig    *HttpConfig
}

type UpdateServerRequest struct {
	Name          *string
	Namespace     *string
	Kind          *string
	ObjectID      *string
	ContainerName *string
	Interval      *time.Duration
	Timeout       *time.Duration
	HttpConfig    *HttpConfig
}

type TestEndpointRequest struct {
	Namespace     string
	ObjectID      string
	Kind          string
	ContainerName string
	Timeout       time.Duration
	HttpConfig    *HttpConfig
}

type TestEndpointResponse struct {
	Running bool
	Error   *string
}

type Container struct {
	Name  string
	Image string
}

type CreateK8sObjectRequest struct {
	Name          string
	Namespace     string
	ObjectID      string
	Containers    []Container
	ContainerName string
	Interval      time.Duration
	Timeout       time.Duration
	HttpConfig    *HttpConfig
}

type DeleteK8sObjectRequest struct {
	Namespace string
	ObjectID  string
}
