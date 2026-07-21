package service

import (
	"context"
	"time"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/consumer"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/repository"
)

type OwnershipService struct {
	consumer *consumer.OwnershipConsumer
	repo     *repository.ServerOwnerRepository
}

func RegisterOwnershipService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*OwnershipService, error) {
		return &OwnershipService{
			consumer: do.MustInvoke[*consumer.OwnershipConsumer](i),
			repo:     do.MustInvoke[*repository.ServerOwnerRepository](i),
		}, nil
	})
}

func (s *OwnershipService) OnCreate(ctx context.Context, serverID, userID uint) error {
	return s.repo.Upsert(ctx, serverID, userID, nil)
}

func (s *OwnershipService) OnUpdate(ctx context.Context, serverID, userID uint, deletedAt *time.Time) error {
	return s.repo.Upsert(ctx, serverID, userID, deletedAt)
}

func (s *OwnershipService) OnDelete(ctx context.Context, serverID uint) error {
	return s.repo.Delete(ctx, serverID)
}

func (s *OwnershipService) Run(ctx context.Context) {
	s.consumer.Run(ctx, s)
}
