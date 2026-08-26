package service

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/samber/do/v2"
	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	pinginfra "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/grpcclient"
	pingrepo "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/repository"
)

const (
	PushInterval = 30 * time.Second

	// PushStaleInterval is the freshness lease per push event: an agent that
	// stays silent for 3 push windows in a row is considered stale.
	PushStaleInterval = 90 * time.Second
)

type PushEventItem struct {
	ID     uint64
	Status string
}

type PushEventError struct {
	ID    uint64
	Error string
}

type PushEventResult struct {
	NextTime time.Time
	Accepted []uint64
	Errors   []PushEventError
}

type RateLimitedError struct {
	NextTime time.Time
}

func (e *RateLimitedError) Error() string {
	return "push rate limited until " + e.NextTime.String()
}

type PushServerResolver interface {
	ResolveServers(ctx context.Context, userID uint, ids []uint64) ([]uint64, error)
}

type PushRateGate interface {
	Allow(ctx context.Context, sessionID string, interval time.Duration) (time.Time, bool, error)
	Release(ctx context.Context, sessionID string) error
}

type PushEventRecorder interface {
	Record(ctx context.Context, event *domain.ServerEvent, freshness time.Duration) error
}

type PushEventService struct {
	resolver PushServerResolver
	gate     PushRateGate
	recorder PushEventRecorder
	logger   *slog.Logger
}

func NewPushEventService(
	resolver PushServerResolver,
	gate PushRateGate,
	recorder PushEventRecorder,
	logger *slog.Logger,
) *PushEventService {
	return &PushEventService{
		resolver: resolver,
		gate:     gate,
		recorder: recorder,
		logger:   logger,
	}
}

func RegisterPushEventService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*PushEventService, error) {
		return NewPushEventService(
			do.MustInvoke[*grpcclient.ServerClient](i),
			do.MustInvoke[*pingrepo.PushRateLimiter](i),
			do.MustInvoke[*pinginfra.RecordStatusWorker](i),
			do.MustInvoke[*slog.Logger](i),
		), nil
	})
}

var validStatuses = []string{
	string(domain.StatusOn),
	string(domain.StatusOff),
}

// filterValid splits items into those with a usable status and per-item errors.
func filterValid(items []PushEventItem) ([]PushEventItem, []PushEventError) {

	var pending []PushEventItem
	var errs []PushEventError

	for _, item := range items {

		if slices.Contains(validStatuses, item.Status) {
			pending = append(pending, item)
			continue
		}

		errs = append(errs, PushEventError{ID: item.ID, Error: "invalid status"})
	}

	return pending, errs
}

func idsOf(items []PushEventItem) []uint64 {
	return lo.Map(items, func(item PushEventItem, _ int) uint64 { return item.ID })
}

func (s *PushEventService) resolveOwned(ctx context.Context, userID uint, ids []uint64) (map[uint64]struct{}, error) {

	resolved, err := s.resolver.ResolveServers(ctx, userID, ids)
	if err != nil {
		return nil, fmt.Errorf("resolve servers: %w", err)
	}

	owned := make(map[uint64]struct{}, len(resolved))
	for _, id := range resolved {
		owned[id] = struct{}{}
	}

	return owned, nil
}

// filterOwned keeps items whose id the user owns; the rest become errors.
func filterOwned(pending []PushEventItem, owned map[uint64]struct{}) ([]PushEventItem, []PushEventError) {

	var valid []PushEventItem
	var errs []PushEventError

	for _, item := range pending {

		if _, ok := owned[item.ID]; ok {
			valid = append(valid, item)
			continue
		}

		errs = append(errs, PushEventError{ID: item.ID, Error: "not found"})
	}

	return valid, errs
}

func (s *PushEventService) Handle(ctx context.Context, userID uint, sessionID string, items []PushEventItem) (*PushEventResult, error) {

	pending, errs := filterValid(items)

	if len(pending) > 0 {

		owned, err := s.resolveOwned(ctx, userID, idsOf(pending))
		if err != nil {
			return nil, err
		}

		var ownedErrs []PushEventError
		pending, ownedErrs = filterOwned(pending, owned)
		errs = append(errs, ownedErrs...)
	}

	result := &PushEventResult{Errors: errs}
	if len(pending) == 0 {
		return result, nil
	}

	next, allowed, err := s.gate.Allow(ctx, sessionID, PushInterval)
	if err != nil {
		return nil, fmt.Errorf("push gate: %w", err)
	}
	if !allowed {
		return nil, &RateLimitedError{NextTime: next}
	}

	result.NextTime = next

	for _, item := range pending {

		event := domain.ServerEvent{
			EndpointID: uint(item.ID),
			Status:     domain.ServerStatus(item.Status),
		}

		if err := s.recorder.Record(ctx, &event, PushStaleInterval); err != nil {

			s.logger.Warn(
				"failed to record push event",
				slog.Uint64("id", item.ID),
				slog.Any("error", err),
			)

			pushErr := PushEventError{
				ID:    item.ID,
				Error: "record failed",
			}

			result.Errors = append(result.Errors, pushErr)
			continue
		}

		result.Accepted = append(result.Accepted, item.ID)
	}

	if len(result.Accepted) == 0 {

		result.NextTime = time.Time{}

		if err := s.gate.Release(ctx, sessionID); err != nil {
			s.logger.Warn("failed to release push milestone", slog.Any("error", err))
		}
	}

	return result, nil
}
