package handler

import (
	"context"
	"io"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
	"github.com/minhnbnt/uptime-monitor/internal/features/auth/middleware"
	"github.com/minhnbnt/uptime-monitor/internal/features/importer/dto"
	importer "github.com/minhnbnt/uptime-monitor/internal/features/importer/service"
)

type ImportHandler struct {
	importService ImportService
}

type ImportService interface {
	ImportServers(ctx context.Context, userID uint, file io.Reader) (*dto.ImportResult, error)
	GenerateTemplate() (io.ReadCloser, error)
}

func RegisterImportHandler(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ImportHandler, error) {
		return &ImportHandler{
			importService: do.MustInvoke[*importer.ImportService](i),
		}, nil
	})
}

func (h *ImportHandler) ImportServers(ctx context.Context, req *api.ImportServersReq) (*api.ImportServersResponse, error) {

	userID := middleware.GetUserID(ctx)

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

var _ ImportService = (*importer.ImportService)(nil)

func (h *ImportHandler) DownloadImportTemplate(ctx context.Context) (api.DownloadImportTemplateOK, error) {

	reader, err := h.importService.GenerateTemplate()
	if err != nil {
		return api.DownloadImportTemplateOK{}, apperrors.ToAPIError(err)
	}

	return api.DownloadImportTemplateOK{Data: reader}, nil
}
