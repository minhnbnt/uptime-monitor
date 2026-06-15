package handler

import (
	"context"
	"io"

	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
)

type ServerService interface {
	ListServers(ctx context.Context, createdByID uint, page, perPage int) ([]dto.Server, error)
	CreateServer(ctx context.Context, req dto.CreateServerRequest, createdByID uint) (*dto.Server, error)
	GetServer(ctx context.Context, id uint) (*dto.Server, error)
	UpdateServer(ctx context.Context, id uint, req dto.UpdateServerRequest) (*dto.Server, error)
	DeleteServer(ctx context.Context, id uint) error
	SearchServers(ctx context.Context, q string, createdByID uint, page, perPage int) ([]dto.Server, int64, error)
}

type OntimeService interface {
	ListServersWithOntime(ctx context.Context, createdByID uint, page, perPage int) ([]dto.ServerWithOntime, int64, error)
	GetServerWithOntime(ctx context.Context, serverID uint) (*dto.ServerWithOntime, error)
}

type EndpointService interface {
	SetCheckMethod(ctx context.Context, serverID uint, req dto.SetCheckMethodRequest) error
	TestEndpoint(ctx context.Context, req dto.TestEndpointRequest) (*dto.TestEndpointResponse, error)
}

type ImportService interface {
	ImportServers(ctx context.Context, userID uint, file io.Reader) (*dto.ImportResult, error)
	GenerateTemplate(w io.Writer) error
}

type AuthService interface {
	Register(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error)
	Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error)
	Refresh(ctx context.Context, req dto.RefreshRequest) (*dto.AuthResponse, error)
	Logout(ctx context.Context, refreshToken string) error
}
