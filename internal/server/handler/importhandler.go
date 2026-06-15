package handler

import (
	"bytes"
	"context"
	"net/http"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	"github.com/minhnbnt/uptime-monitor/internal/server/middleware"
	"github.com/minhnbnt/uptime-monitor/internal/server/service"
)

type ImportHandler struct {
	importService ImportService
}

func RegisterImportHandler(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ImportHandler, error) {
		return &ImportHandler{
			importService: do.MustInvoke[*service.ImportService](i),
		}, nil
	})
}

func (h *ImportHandler) ImportServers(ctx context.Context, req *api.ImportServersReq) (*api.ImportServersResponse, error) {

	userID := middleware.GetUserID(ctx)

	result, err := h.importService.ImportServers(ctx, userID, req.File.File)
	if err != nil {
		return nil, &api.ErrorResponseStatusCode{
			StatusCode: http.StatusBadRequest,
			Response:   errResponse("IMPORT_FAILED", err.Error()),
		}
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

func (h *ImportHandler) DownloadImportTemplate(ctx context.Context) (api.DownloadImportTemplateOK, error) {

	buf := new(bytes.Buffer)

	if err := h.importService.GenerateTemplate(buf); err != nil {
		return api.DownloadImportTemplateOK{}, &api.ErrorResponseStatusCode{
			StatusCode: http.StatusInternalServerError,
			Response:   errResponse("TEMPLATE_ERROR", err.Error()),
		}
	}

	return api.DownloadImportTemplateOK{Data: buf}, nil
}
