package ontimeclient

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/samber/do/v2"
	"github.com/samber/lo"

	eventv1 "github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/event/v1"
	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/domain"
)

type Client struct {
	client     eventv1.OntimeServiceClient
	wrapper    *config.GRPCOntimeClientWrapper
	logger     *slog.Logger
	maxRecords int
}

func NewClient(
	wrapper *config.GRPCOntimeClientWrapper,
	logger *slog.Logger, maxRecords int,
) *Client {

	client := eventv1.NewOntimeServiceClient(wrapper.GetConn())

	return &Client{
		client:     client,
		logger:     logger,
		wrapper:    wrapper,
		maxRecords: maxRecords,
	}
}

func RegisterClient(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*Client, error) {

		cfg := do.MustInvoke[*config.Config](i)
		wrapper, err := config.NewGRPCOntimeClientWrapper(cfg.GRPC.EventAddr)
		if err != nil {
			return nil, fmt.Errorf("dial ontime grpc: %w", err)
		}

		return NewClient(
			wrapper,
			do.MustInvoke[*slog.Logger](i),
			cfg.Digest.MaxRecords,
		), nil
	})
}

func (a *Client) Shutdown() error {
	return a.wrapper.Shutdown()
}

func (a *Client) GetServersOntimeForDates(
	ctx context.Context, userID uint,
	servers []domain.Server, dates []time.Time,
) (map[uint][]domain.OntimeStats, error) {

	a.logger.Debug(
		"ontimeclient.GetServersOntimeForDates: sending gRPC request",
		slog.Uint64("user_id", uint64(userID)),
		slog.Int("servers", len(servers)),
		slog.Int("dates", len(dates)),
	)

	serverIDs := lo.Map(servers, func(sv domain.Server, _ int) uint64 {
		return uint64(sv.ID)
	})

	request := eventv1.GetServersOntimeRequest{
		UserId: uint64(userID),
	}
	if len(serverIDs) > 0 {
		request.ServerIds = serverIDs
	}
	if len(dates) > 0 {
		request.FromDate = dates[0].UnixMilli()
		request.ToDate = dates[len(dates)-1].UnixMilli()
	}

	resp, err := a.client.GetServersOntime(ctx, &request)
	if err != nil {
		a.logger.Error(
			"ontimeclient.GetServersOntimeForDates: rpc failed",
			slog.Uint64("user_id", uint64(userID)),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("get servers ontime: %w", err)
	}

	result := make(map[uint][]domain.OntimeStats, len(resp.Servers))
	for _, sv := range resp.Servers {

		stats := make([]domain.OntimeStats, 0, len(sv.OntimeStats))

		for _, st := range sv.OntimeStats {
			parsed, perr := time.Parse("2006-01-02", st.Date)
			if perr != nil {
				continue
			}
			stats = append(stats, domain.OntimeStats{
				Date:  parsed,
				Stats: st.Stats,
			})
		}

		result[uint(sv.ServerId)] = stats
	}

	return result, nil
}
