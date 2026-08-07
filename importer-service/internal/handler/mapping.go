package handler

import (
	"github.com/minhnbnt/uptime-monitor-microservices/importer-service/generated/api"
	"github.com/minhnbnt/uptime-monitor-microservices/importer-service/internal/dto"
)

func toAPIRowErrors(result *dto.ImportResult) []api.ImportServerRowError {

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

	return failed
}

func toAPISuccesses(s dto.ImportSuccess, _ int) api.ImportServerSuccess {
	return api.ImportServerSuccess{
		Row:      api.NewOptInt(s.Row),
		Name:     api.NewOptString(s.Name),
		ServerID: api.NewOptInt(int(s.ServerID)),
	}
}
