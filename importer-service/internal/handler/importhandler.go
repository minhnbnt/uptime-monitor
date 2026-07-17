package handler

import (
	"context"
	"io"
	"log/slog"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/common/authclient"
	"github.com/minhnbnt/uptime-monitor-microservices/importer-service/generated/api"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/importer-service/internal/errors"
	"github.com/minhnbnt/uptime-monitor-microservices/importer-service/internal/dto"
	"github.com/minhnbnt/uptime-monitor-microservices/importer-service/internal/service"
)

type ImportHandler struct {
	importService ImportService
	logger        *slog.Logger
}

type ImportService interface {
	ImportServers(ctx context.Context, userID uint, file io.Reader) (*dto.ImportResult, error)
	GenerateTemplate() (io.ReadCloser, error)
	ExportServers(ctx context.Context, userID uint, q string, from, to int, sortBy, sortOrder string) (io.ReadCloser, error)
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

	successes := make([]api.ImportServerSuccess, len(result.Successes))
	for i, s := range result.Successes {
		successes[i] = api.ImportServerSuccess{
			Row:      api.NewOptInt(s.Row),
			Name:     api.NewOptString(s.Name),
			URL:      api.NewOptString(s.URL),
			ServerID: api.NewOptInt(int(s.ServerID)),
		}
	}

	failed := make([]api.ImportServerRowError, 0, len(result.RowErrors)+len(result.BatchErrors))

	for _, e := range result.RowErrors {
		failed = append(failed, api.ImportServerRowError{
			Row:     api.NewOptInt(e.Row),
			Message: api.NewOptString(e.Message),
		})
	}

	for _, e := range result.BatchErrors {
		failed = append(failed, api.ImportServerRowError{
			Message: api.NewOptString(e.Message),
		})
	}

	return &api.ImportServersResponse{
		SuccessCount: len(result.Successes),
		Successes:    successes,
		FailedCount:  len(failed),
		Failed:       failed,
	}, nil
}

var _ ImportService = (*service.ImportService)(nil)

func (h *ImportHandler) DownloadImportTemplate(ctx context.Context) (api.DownloadImportTemplateOK, error) {

	reader, err := h.importService.GenerateTemplate()
	if err != nil {
		return api.DownloadImportTemplateOK{}, apperrors.ToAPIError(err)
	}

	return api.DownloadImportTemplateOK{Data: reader}, nil
}

func (h *ImportHandler) ExportServers(ctx context.Context, params api.ExportServersParams) (*api.ExportServersOKHeaders, error) {

	userID := authclient.GetUserID(ctx)

	reader, err := h.importService.ExportServers(
		ctx, userID,
		params.Q.Or(""),
		params.From.Or(0),
		params.To.Or(100),
		string(params.SortBy.Or(api.ExportServersSortByName)),
		string(params.SortOrder.Or(api.ExportServersSortOrderAsc)),
	)
	if err != nil {
		return nil, apperrors.ToAPIError(err)
	}

	return &api.ExportServersOKHeaders{
		ContentDisposition: api.NewOptString(`attachment; filename="servers.xlsx"`),
		Response:           api.ExportServersOK{Data: reader},
	}, nil
}
