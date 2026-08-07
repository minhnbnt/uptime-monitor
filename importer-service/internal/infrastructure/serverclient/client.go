package serverclient

import (
	"context"

	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"github.com/samber/lo"

	serverv1 "github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/server/v1"
	"github.com/minhnbnt/uptime-monitor-microservices/importer-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/importer-service/internal/dto"
)

type ServerClient struct {
	client serverv1.ServerServiceClient
}

func RegisterServerClient(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ServerClient, error) {

		wrapper := do.MustInvoke[*config.GRPCClientWrapper](i)
		client := serverv1.NewServerServiceClient(wrapper.GetClient())

		return &ServerClient{client: client}, nil
	})
}

func (c *ServerClient) BatchCreateServers(
	ctx context.Context, userID uuid.UUID, rows []dto.ImportRow,
) ([]dto.ImportSuccess, []dto.ImportError, error) {

	request := serverv1.BatchCreateServersRequest{
		Servers: toServerInputs(rows, userID),
	}

	resp, err := c.client.BatchCreateServers(ctx, &request)
	if err != nil {
		return nil, nil, err
	}

	successes, batchErrors := toImportResults(resp.Results)
	return successes, batchErrors, nil
}

func (c *ServerClient) SearchServers(
	ctx context.Context,
	userID uuid.UUID, params dto.SearchServersParams,
) ([]dto.Server, error) {

	request := serverv1.SearchServersRequest{
		UserId:    userID.String(),
		Q:         params.Q,
		From:      int32(params.From),
		To:        int32(params.To),
		SortBy:    params.SortBy,
		SortOrder: params.SortOrder,
	}

	resp, err := c.client.SearchServers(ctx, &request)
	if err != nil {
		return nil, err
	}

	return lo.Map(resp.Servers, toServerDto), nil
}
