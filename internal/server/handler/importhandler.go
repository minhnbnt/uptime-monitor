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

	n := len(result.RowErrors) + len(result.BatchErrors)
	apiErrors := make([]api.ImportServerRowError, 0, n)

	for _, e := range result.RowErrors {
		apiErrors = append(apiErrors, api.ImportServerRowError{
			Row:     api.NewOptInt(e.Row),
			Message: api.NewOptString(e.Message),
		})
	}

	for _, e := range result.BatchErrors {
		apiErrors = append(apiErrors, api.ImportServerRowError{
			Message: api.NewOptString(e.Message),
		})
	}

	return &api.ImportServersResponse{
		Imported: api.NewOptInt(result.Imported),
		Errors:   apiErrors,
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
