package service

import (
	"context"
	"time"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
)

type mockNotificationConfigRepository struct {
	getByUserIDFn func(ctx context.Context, userID uint) (*domain.NotificationConfig, error)
	upsertFn      func(ctx context.Context, cfg *domain.NotificationConfig) error
}

func (m *mockNotificationConfigRepository) GetByUserID(ctx context.Context, userID uint) (*domain.NotificationConfig, error) {
	return m.getByUserIDFn(ctx, userID)
}

func (m *mockNotificationConfigRepository) Upsert(ctx context.Context, cfg *domain.NotificationConfig) error {
	return m.upsertFn(ctx, cfg)
}

type mockDigestStarter struct {
	startDigestFn    func(ctx context.Context, userID uint) error
	upsertScheduleFn func(ctx context.Context, userID uint, fromDate, toDate time.Time, digestTime string) error
	deleteScheduleFn func(ctx context.Context, userID uint) error
}

func (m *mockDigestStarter) StartDigest(ctx context.Context, userID uint) error {
	return m.startDigestFn(ctx, userID)
}

func (m *mockDigestStarter) UpsertSchedule(ctx context.Context, userID uint, fromDate, toDate time.Time, digestTime string) error {
	return m.upsertScheduleFn(ctx, userID, fromDate, toDate, digestTime)
}

func (m *mockDigestStarter) DeleteSchedule(ctx context.Context, userID uint) error {
	return m.deleteScheduleFn(ctx, userID)
}
