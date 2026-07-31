package handler

import (
	"context"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/common/authclient"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/generated/api"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/dto"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/errors"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/service"
)

type K8sObjectHandler struct {
	k8sObjectService K8sObjectService
}

func RegisterK8sObjectHandler(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*K8sObjectHandler, error) {
		return &K8sObjectHandler{
			k8sObjectService: do.MustInvoke[*service.K8sObjectService](i),
		}, nil
	})
}

func (h *K8sObjectHandler) CreateK8sObject(
	ctx context.Context,
	req *api.CreateK8sObjectRequest,
) (*api.K8sObjectResponse, error) {

	userID := authclient.GetUserID(ctx)
	request := ToCreateK8sObjectRequest(req)
	if len(request.Containers) == 0 {
		return nil, apperrors.ErrBadRequest
	}

	for _, ctr := range request.Containers {
		if ctr.Name == "" || ctr.Image == "" {
			return nil, apperrors.ErrBadRequest
		}
	}

	result, err := h.k8sObjectService.CreateK8sObject(ctx, request, userID)
	if err != nil {
		return nil, apperrors.ToAPIError(err)
	}

	server := ToAPIServer(result)
	return &api.K8sObjectResponse{Data: server}, nil
}

func (h *K8sObjectHandler) DeleteK8sObject(
	ctx context.Context,
	params api.DeleteK8sObjectParams,
) error {

	object := dto.DeleteK8sObjectRequest{
		Namespace: params.Namespace,
		ObjectID:  params.ObjectID,
	}

	if err := h.k8sObjectService.DeleteK8sObject(ctx, object); err != nil {
		return apperrors.ToAPIError(err)
	}

	return nil
}

var _ K8sObjectService = (*service.K8sObjectService)(nil)
