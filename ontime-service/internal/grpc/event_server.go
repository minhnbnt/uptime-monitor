package grpc

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	eventv1 "github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/event/v1"
	"github.com/samber/do/v2"
	"github.com/samber/lo"
	"google.golang.org/grpc"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/recorder"
)

type EventServer struct {
	eventv1.UnimplementedEventServiceServer
	recorder *recorder.DedupRecorder
	db       *gorm.DB
}

func RegisterEventServer(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EventServer, error) {
		dbWrapper := do.MustInvoke[*config.GORMWrapper](i)
		return &EventServer{
			recorder: do.MustInvoke[*recorder.DedupRecorder](i),
			db:       dbWrapper.GetDB(),
		}, nil
	})

	gRPCServer := do.MustInvoke[*grpc.Server](i)
	eventv1.RegisterEventServiceServer(gRPCServer, do.MustInvoke[*EventServer](i))
}

func (s *EventServer) RecordEvent(ctx context.Context, req *eventv1.RecordEventRequest) (*eventv1.RecordEventResponse, error) {

	if err := s.recorder.RecordEvent(ctx, uint(req.EndpointId), domain.ServerStatus(req.Status)); err != nil {
		return nil, err
	}

	return &eventv1.RecordEventResponse{}, nil
}

func (s *EventServer) GetCurrentStatuses(ctx context.Context, req *eventv1.GetCurrentStatusesRequest) (*eventv1.GetCurrentStatusesResponse, error) {

	if len(req.EndpointIds) == 0 {
		return &eventv1.GetCurrentStatusesResponse{}, nil
	}

	ids := make([]uint, len(req.EndpointIds))
	for i, id := range req.EndpointIds {
		ids[i] = uint(id)
	}

	type statusRow struct {
		EndpointID uint
		Status     string
	}

	var rows []statusRow
	result := s.db.WithContext(ctx).
		Select("DISTINCT ON (endpoint_id) endpoint_id, status").
		Table("server_events").
		Where("endpoint_id IN ?", ids).
		Order("endpoint_id, time DESC").
		Scan(&rows)

	if err := result.Error; err != nil {
		return nil, fmt.Errorf("query current statuses: %w", err)
	}

	statuses := make([]*eventv1.EndpointStatus, len(rows))
	for i, r := range rows {
		statuses[i] = &eventv1.EndpointStatus{
			EndpointId: uint64(r.EndpointID),
			Status:     r.Status,
		}
	}

	return &eventv1.GetCurrentStatusesResponse{Statuses: statuses}, nil
}

func (s *EventServer) CountByStatus(ctx context.Context, req *eventv1.CountByStatusRequest) (*eventv1.CountByStatusResponse, error) {

	if len(req.EndpointIds) == 0 {
		return &eventv1.CountByStatusResponse{}, nil
	}

	ids := lo.Map(req.EndpointIds, func(id uint64, _ int) uint {
		return uint(id)
	})

	type countRow struct {
		Status string
		Count  int64
	}

	latestStatusTable := s.db.WithContext(ctx).
		Select("DISTINCT ON (endpoint_id) endpoint_id, status").
		Table("server_events").
		Where("endpoint_id IN ?", ids).
		Order("endpoint_id, time DESC")

	rows := []countRow{}

	result := s.db.WithContext(ctx).
		Select("status, COUNT(*) AS count").
		Table("(?) AS latest", latestStatusTable).
		Group("status").
		Scan(&rows)

	if err := result.Error; err != nil {
		return nil, fmt.Errorf("count by status: %w", err)
	}

	online, offline := int64(0), int64(0)
	for _, row := range rows {
		switch row.Status {
		case "ON":
			online = row.Count
		case "OFF":
			offline = row.Count
		}
	}

	return &eventv1.CountByStatusResponse{Online: online, Offline: offline}, nil
}
