package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/dto"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/errors"
)

var errTestError = errors.New("test error")

var testUserID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
var testUserID2 = uuid.MustParse("22222222-2222-2222-2222-222222222222")

type mockDigestStarter struct {
	mock.Mock
}

func (m *mockDigestStarter) StartDigest(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *mockDigestStarter) UpsertSchedule(ctx context.Context, userID uuid.UUID, cfg domain.ScheduleConfig) error {
	args := m.Called(ctx, userID, cfg)
	return args.Error(0)
}

func (m *mockDigestStarter) DeleteSchedule(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *mockDigestStarter) DescribeSchedule(ctx context.Context, userID uuid.UUID) (*domain.ScheduleInfo, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(*domain.ScheduleInfo), args.Error(1)
}

func newTestService(mockStarter *mockDigestStarter) *NotificationService {
	return &NotificationService{
		digestStarter: mockStarter,
		logger:        slog.Default(),
	}
}

func TestGetNotificationConfig_WithSchedule(t *testing.T) {
	mockS := &mockDigestStarter{}
	svc := newTestService(mockS)

	fromDate := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	mockS.On("DescribeSchedule", mock.Anything, testUserID).Return(&domain.ScheduleInfo{
		Exists:     true,
		FromDate:   fromDate,
		ToDate:     toDate,
		DigestTime: "09:00",
	}, nil)

	resp, err := svc.GetNotificationConfig(t.Context(), testUserID)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, "2026-06-01", resp.FromDate)
	require.Equal(t, "2026-07-01", resp.ToDate)
	require.Equal(t, "09:00", resp.DigestTime)
}

func TestGetNotificationConfig_NoSchedule(t *testing.T) {
	mockS := &mockDigestStarter{}
	svc := newTestService(mockS)

	mockS.On("DescribeSchedule", mock.Anything, testUserID).Return(&domain.ScheduleInfo{}, nil)

	_, err := svc.GetNotificationConfig(t.Context(), testUserID)
	require.ErrorIs(t, err, apperrors.ErrNotFound)
}

func TestGetNotificationConfig_DescribeError(t *testing.T) {
	mockS := &mockDigestStarter{}
	svc := newTestService(mockS)

	mockS.On("DescribeSchedule", mock.Anything, testUserID).Return(&domain.ScheduleInfo{}, errTestError)

	_, err := svc.GetNotificationConfig(t.Context(), testUserID)
	require.Error(t, err)
}

func TestUpdateNotificationConfig_Active(t *testing.T) {
	mockS := &mockDigestStarter{}
	svc := newTestService(mockS)

	req := &dto.NotificationConfigRequest{
		FromDate:   "2026-06-01",
		ToDate:     "2026-07-01",
		DigestTime: "09:00",
	}

	mockS.On("UpsertSchedule", mock.Anything, testUserID, domain.ScheduleConfig{
		FromDate:   time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		ToDate:     time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		DigestTime: "09:00",
	}).Return(nil)

	err := svc.UpdateNotificationConfig(t.Context(), testUserID, req)
	require.NoError(t, err)
	mockS.AssertExpectations(t)
}

func TestUpdateNotificationConfig_Inactive_DeletesSchedule(t *testing.T) {
	mockS := &mockDigestStarter{}
	svc := newTestService(mockS)

	req := &dto.NotificationConfigRequest{
		FromDate:   "",
		ToDate:     "",
		DigestTime: "08:00",
	}

	mockS.On("DeleteSchedule", mock.Anything, testUserID).Return(nil)

	err := svc.UpdateNotificationConfig(t.Context(), testUserID, req)
	require.NoError(t, err)
	mockS.AssertExpectations(t)
}

func TestSendReport(t *testing.T) {
	mockS := &mockDigestStarter{}
	svc := newTestService(mockS)

	mockS.On("DescribeSchedule", mock.Anything, testUserID).Return(&domain.ScheduleInfo{Exists: true}, nil)
	mockS.On("StartDigest", mock.Anything, testUserID).Return(nil)

	err := svc.SendReport(t.Context(), testUserID)
	require.NoError(t, err)
	mockS.AssertExpectations(t)
}

func TestSendReport_NoConfig(t *testing.T) {
	mockS := &mockDigestStarter{}
	svc := newTestService(mockS)

	mockS.On("DescribeSchedule", mock.Anything, testUserID2).Return(&domain.ScheduleInfo{}, nil)

	err := svc.SendReport(t.Context(), testUserID2)
	require.ErrorIs(t, err, apperrors.ErrNotFound)
	mockS.AssertExpectations(t)
}
