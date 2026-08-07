package service

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"github.com/samber/lo/it"

	serverv1 "github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/server/v1"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/infrastructure/repository"
)

const batchChunkSize = 100

type ServerBatchService struct {
	serverRepo *repository.ServerRepository
	logger     *slog.Logger
}

func RegisterServerBatchService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ServerBatchService, error) {
		return &ServerBatchService{
			serverRepo: do.MustInvoke[*repository.ServerRepository](i),
			logger:     do.MustInvoke[*slog.Logger](i),
		}, nil
	})
}

func (s *ServerBatchService) BatchCreateServers(
	ctx context.Context,
	inputs []*serverv1.ServerWithEndpointInput,
) ([]*serverv1.BatchCreateServerResult, error) {

	results := []*serverv1.BatchCreateServerResult{}

	for chunk := range it.Chunk(slices.Values(inputs), batchChunkSize) {

		servers, configs, err := buildDomainServers(chunk)
		if err != nil {

			s.logger.Error("batch create servers failed", slog.Any("error", err))

			newResults := it.Map(
				slices.Values(chunk),
				func(input *serverv1.ServerWithEndpointInput) *serverv1.BatchCreateServerResult {
					return &serverv1.BatchCreateServerResult{
						Row:       input.Row,
						Name:      input.Name,
						Namespace: input.Namespace,
						Error:     err.Error(),
					}
				},
			)

			results = slices.AppendSeq(results, newResults)
			continue
		}

		err = s.serverRepo.BatchCreateServers(ctx, servers, configs)
		if err != nil {

			s.logger.Error("batch create servers failed", slog.Any("error", err))

			newResults := it.Map(
				slices.Values(chunk),
				func(input *serverv1.ServerWithEndpointInput) *serverv1.BatchCreateServerResult {
					return &serverv1.BatchCreateServerResult{
						Row:       input.Row,
						Name:      input.Name,
						Namespace: input.Namespace,
						Error:     err.Error(),
					}
				},
			)

			results = slices.AppendSeq(results, newResults)
			continue
		}

		for i, sv := range servers {
			results = append(results, &serverv1.BatchCreateServerResult{
				Row:       chunk[i].Row,
				Name:      sv.Name,
				Namespace: sv.Namespace,
				ServerId:  uint64(sv.ID),
			})
		}
	}

	return results, nil
}

func buildDomainServers(inputs []*serverv1.ServerWithEndpointInput) ([]domain.Server, []*domain.ServerHttpConfig, error) {

	servers := make([]domain.Server, 0, len(inputs))
	configs := make([]*domain.ServerHttpConfig, 0, len(inputs))
	for _, in := range inputs {

		userID, err := uuid.Parse(in.UserId)
		if err != nil {
			return nil, nil, fmt.Errorf("row %d: invalid user id %q: %w", in.Row, in.UserId, err)
		}

		servers = append(servers, domain.Server{
			Name:          in.Name,
			Namespace:     in.Namespace,
			Kind:          in.Kind,
			ObjectID:      in.ObjectId,
			ContainerName: in.ContainerName,
			Interval:      time.Duration(in.IntervalMs) * time.Millisecond,
			Timeout:       time.Duration(in.TimeoutMs) * time.Millisecond,
			CreatedByID:   userID,
		})

		configs = append(configs, httpConfigFromInput(in.HttpConfig))
	}

	return servers, configs, nil
}

func httpConfigFromInput(in *serverv1.HttpConfigInput) *domain.ServerHttpConfig {

	if in == nil {
		return nil
	}

	return &domain.ServerHttpConfig{
		Port:          int(in.Port),
		EndpointPath:  in.EndpointPath,
		ExpectedCode:  int(in.ExpectedCode),
		BodyCheckExpr: in.BodyCheckExpr,
		Method:        defaultHttpMethod(in.Method),
	}
}
