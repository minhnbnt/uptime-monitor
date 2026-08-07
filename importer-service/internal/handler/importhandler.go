package handler

import (
	"context"
	"io"
	"log/slog"

	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor-microservices/common/authclient"
	"github.com/minhnbnt/uptime-monitor-microservices/importer-service/generated/api"
	"github.com/minhnbnt/uptime-monitor-microservices/importer-service/internal/dto"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/importer-service/internal/errors"
	"github.com/minhnbnt/uptime-monitor-microservices/importer-service/internal/service"
)

type ImportHandler struct {
	importService ImportService
	logger        *slog.Logger
}

type ImportService interface {
	ExportServers(ctx context.Context, userID uuid.UUID, params dto.SearchServersParams) (io.ReadCloser, error)
	ImportServers(ctx context.Context, userID uuid.UUID, file io.Reader) (*dto.ImportResult, error)
	GenerateTemplate() (io.ReadCloser, error)
}

func RegisterImportHandler(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ImportHandler, error) {
		return &ImportHandler{
			importService: do.MustInvoke[*service.ImportService](i),
			logger:        do.MustInvoke[*slog.Logger](i),
		}, nil
	})
}

func (h *ImportHandler) NewError(_ context.Context, err error) *api.ErrorResponseStatusCode {
	h.logger.Error("unhandled error", slog.Any("error", err))
	return apperrors.ToAPIError(err)
}

func (h *ImportHandler) ImportServers(ctx context.Context, req *api.ImportServersReq) (*api.ImportServersResponse, error) {

	userID := authclient.GetUserID(ctx)

	result, err := h.importService.ImportServers(ctx, userID, req.File.File)
	if err != nil {
		return nil, apperrors.ToAPIError(err)
	}

	failed := toAPIRowErrors(result)
	successes := lo.Map(result.Successes, toAPISuccesses)

	return &api.ImportServersResponse{
		SuccessCount: len(result.Successes),
		Successes:    successes,
		FailedCount:  len(failed),
		Failed:       failed,
	}, nil
}

var _ ImportService = (*service.ImportService)(nil)

func (h *ImportHandler) DownloadImportTemplate(_ context.Context) (api.DownloadImportTemplateOK, error) {

	reader, err := h.importService.GenerateTemplate()
	if err != nil {
		return api.DownloadImportTemplateOK{}, apperrors.ToAPIError(err)
	}

	return api.DownloadImportTemplateOK{Data: reader}, nil
}

func (h *ImportHandler) ExportServers(ctx context.Context, params api.ExportServersParams) (*api.ExportServersOKHeaders, error) {

	userID := authclient.GetUserID(ctx)
	searchParams := dto.SearchServersParams{
		Q:         params.Q.Or(""),
		From:      params.From.Or(0),
		To:        params.To.Or(100),
		SortBy:    string(params.SortBy.Or(api.ExportServersSortByName)),
		SortOrder: string(params.SortOrder.Or(api.ExportServersSortOrderAsc)),
	}

	reader, err := h.importService.ExportServers(ctx, userID, searchParams)
	if err != nil {
		return nil, apperrors.ToAPIError(err)
	}

	return &api.ExportServersOKHeaders{
		ContentDisposition: api.NewOptString(`attachment; filename="servers.xlsx"`),
		Response:           api.ExportServersOK{Data: reader},
	}, nil
}
