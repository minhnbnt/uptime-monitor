package service

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"github.com/samber/do/v2"
	"github.com/samber/lo"
	"github.com/samber/lo/it"

	serverv1 "github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/server/v1"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/infrastructure/repository"
)

const batchChunkSize = 100

type ServerBatchService struct {
	serverRepo   *repository.ServerRepository
	endpointRepo *repository.EndpointRepository
	logger       *slog.Logger
}

func RegisterServerBatchService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ServerBatchService, error) {
		return &ServerBatchService{
			serverRepo:   do.MustInvoke[*repository.ServerRepository](i),
			endpointRepo: do.MustInvoke[*repository.EndpointRepository](i),
			logger:       do.MustInvoke[*slog.Logger](i),
		}, nil
	})
}

func (s *ServerBatchService) BatchCreateServers(
	ctx context.Context,
	inputs []*serverv1.ServerWithEndpointInput,
) ([]*serverv1.BatchCreateServerResult, error) {

	var results []*serverv1.BatchCreateServerResult

	for chunk := range it.Chunk(slices.Values(inputs), batchChunkSize) {
		servers := buildDomainServers(chunk)
		if err := s.serverRepo.BatchCreateServers(ctx, servers); err != nil {
			s.logger.Error("batch create servers failed", slog.Any("error", err))
			for _, input := range chunk {
				results = append(results, &serverv1.BatchCreateServerResult{
					Row:   input.Row,
					Name:  input.Name,
					Url:   input.Url,
					Error: err.Error(),
				})
			}
			continue
		}

		for i, sv := range servers {
			results = append(results, &serverv1.BatchCreateServerResult{
				Row:      chunk[i].Row,
				Name:     sv.Name,
				Url:      chunk[i].Url,
				ServerId: uint64(sv.ID),
			})
		}

		endpoints := buildDomainEndpoints(chunk, servers)
		if len(endpoints) == 0 {
			continue
		}

		if err := s.endpointRepo.BatchCreateEndpoints(ctx, endpoints); err != nil {
			s.logger.Error("batch create endpoints failed", slog.Any("error", err))
			for i := range servers {
				results[len(results)-len(servers)+i].Error = "endpoint creation failed"
			}
		}
	}

	return results, nil
}

func buildDomainServers(inputs []*serverv1.ServerWithEndpointInput) []domain.Server {
	return lo.Map(inputs, func(in *serverv1.ServerWithEndpointInput, _ int) domain.Server {
		return domain.Server{
			Name:        in.Name,
			CreatedByID: uint(in.UserId),
		}
	})
}

func buildDomainEndpoints(inputs []*serverv1.ServerWithEndpointInput, servers []domain.Server) []domain.Endpoint {
	endpoints := make([]domain.Endpoint, 0, len(servers))
	for i, sv := range servers {
		if inputs[i].Url == "" {
			continue
		}
		endpoints = append(endpoints, domain.Endpoint{
			ServerID:     sv.ID,
			URL:          inputs[i].Url,
			Interval:     time.Duration(inputs[i].IntervalMs) * time.Millisecond,
			Timeout:      time.Duration(inputs[i].TimeoutMs) * time.Millisecond,
			Method:       inputs[i].Method,
			ExpectedCode: int(inputs[i].ExpectedCode),
		})
	}
	return endpoints
}
