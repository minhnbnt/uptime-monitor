package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"maps"

	"github.com/samber/do/v2"
	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/dto"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/grpcclient"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/k8sclient"
)

type ServerProvider struct {
	client    *grpcclient.EndpointClient
	cache     *ServerMetaCache
	k8sClient *k8sclient.K8sClient
	logger    *slog.Logger
}

func RegisterServerProvider(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ServerProvider, error) {
		return &ServerProvider{
			client:    do.MustInvoke[*grpcclient.EndpointClient](i),
			cache:     do.MustInvoke[*ServerMetaCache](i),
			k8sClient: do.MustInvoke[*k8sclient.K8sClient](i),
			logger:    do.MustInvoke[*slog.Logger](i),
		}, nil
	})
}

func (p *ServerProvider) Get(ctx context.Context, id uint) (*domain.Server, error) {

	results, err := p.GetBatch(ctx, []uint{id})
	if err != nil {
		return nil, err
	}

	result, has := results[id]
	if !has {
		return nil, fmt.Errorf("server not found: %d", id)
	}

	return result, nil
}

func (p *ServerProvider) GetBatch(ctx context.Context, ids []uint) (map[uint]*domain.Server, error) {

	if len(ids) == 0 {
		return make(map[uint]*domain.Server), nil
	}

	servers, err := p.cache.MGet(ctx, ids)
	if err != nil {
		p.logger.Error("failed to get servers from cache", "error", err)
		servers = make(map[uint]*domain.Server)
	}

	missed := lo.Filter(ids, func(id uint, _ int) bool {
		_, has := servers[id]
		return !has
	})

	if len(missed) == 0 {
		return servers, nil
	}

	batch, err := p.client.GetBatch(ctx, missed)
	if err != nil {
		return nil, err
	}

	maps.Copy(servers, batch)

	p.fillK8sFields(ctx, servers)

	if err := p.cache.SetMulti(ctx, lo.Values(servers)); err != nil {
		p.logger.Error("failed to set servers in cache", "error", err)
	}

	return servers, nil
}

func (p *ServerProvider) fillK8sFields(ctx context.Context, servers map[uint]*domain.Server) {

	for _, sv := range servers {

		if sv.K8s == nil {
			sv.K8s = &domain.K8sRuntime{}
		}

		params := &dto.K8sObjectCheckParams{
			Namespace:     sv.Namespace,
			Kind:          sv.Kind,
			ObjectID:      sv.ObjectID,
			ContainerName: sv.ContainerName,
			K8s:           sv.K8s,
		}

		if sv.HTTPConfig == nil && k8sclient.IsWorkloadKind(sv.Kind) && sv.K8s.LabelSelector == "" {

			if selector, err := p.k8sClient.ResolveLabelSelector(ctx, params); err != nil {
				p.logger.Warn(
					"failed to resolve label selector",
					slog.Uint64("server_id", uint64(sv.ID)),
					slog.Any("error", err),
				)
			} else {
				sv.K8s.LabelSelector = selector
			}
		}

		if sv.HTTPConfig != nil && sv.K8s.Domain == "" {

			if domain, err := p.k8sClient.ResolveDomainName(ctx, params); err != nil {
				p.logger.Warn(
					"failed to resolve domain",
					slog.Uint64("server_id", uint64(sv.ID)),
					slog.Any("error", err),
				)
			} else {
				sv.K8s.Domain = domain
			}
		}
	}
}
