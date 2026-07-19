package handler

import (
	"context"

	"github.com/samber/do/v2"
	"github.com/samber/lo"

	eventv1 "github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/event/v1"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/dto"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/service"
)

type OntimeGRPCServer struct {
	eventv1.UnsafeOntimeServiceServer
	ontimeService *service.OntimeService
}

func RegisterOntimeGRPCServer(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*OntimeGRPCServer, error) {
		return &OntimeGRPCServer{
			ontimeService: do.MustInvoke[*service.OntimeService](i),
		}, nil
	})
}

func (s *OntimeGRPCServer) GetServersOntime(
	ctx context.Context, req *eventv1.GetServersOntimeRequest,
) (*eventv1.GetServersOntimeResponse, error) {

	ontimeMap, err := s.ontimeService.GetServersOntime(ctx, uint(req.UserId))
	if err != nil {
		return nil, err
	}

	servers := lo.Map(lo.Keys(ontimeMap), func(id uint, _ int) *eventv1.ServerOntimeStat {
		return &eventv1.ServerOntimeStat{
			ServerId: uint64(id),
			OntimeStats: lo.Map(ontimeMap[id], func(stat dto.OntimeStats, _ int) *eventv1.OntimeDayStat {
				return &eventv1.OntimeDayStat{
					Date:  stat.Date.Format("2006-01-02"),
					Stats: stat.Stats,
				}
			}),
		}
	})

	return &eventv1.GetServersOntimeResponse{Servers: servers}, nil
}

var _ eventv1.OntimeServiceServer = (*OntimeGRPCServer)(nil)
