package serverclient

import (
	"time"

	"github.com/google/uuid"

	serverv1 "github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/server/v1"
	"github.com/minhnbnt/uptime-monitor-microservices/importer-service/internal/dto"
)

func toServerInputs(rows []dto.ImportRow, userID uuid.UUID) []*serverv1.ServerWithEndpointInput {

	inputs := make([]*serverv1.ServerWithEndpointInput, 0, len(rows))
	for _, r := range rows {
		inputs = append(inputs, &serverv1.ServerWithEndpointInput{
			Row:           int32(r.Row),
			Name:          r.Name,
			Namespace:     r.Namespace,
			Kind:          r.Kind,
			ObjectId:      r.ObjectID,
			ContainerName: r.ContainerName,
			IntervalMs:    int64(r.Interval) * 1000,
			TimeoutMs:     int64(r.Timeout) * 1000,
			UserId:        userID.String(),
			HttpConfig:    httpConfigFromRow(r),
		})
	}

	return inputs
}

func toImportResults(results []*serverv1.BatchCreateServerResult) ([]dto.ImportSuccess, []dto.ImportError) {

	var (
		successes   []dto.ImportSuccess
		batchErrors []dto.ImportError
	)

	for _, r := range results {
		if r.Error == "" {
			successes = append(successes, dto.ImportSuccess{
				ServerID: uint(r.ServerId),
				Row:      int(r.Row),
				Name:     r.Name,
			})
		} else {
			batchErrors = append(batchErrors, dto.ImportError{Message: r.Error})
		}
	}

	return successes, batchErrors
}

func toServerDto(p *serverv1.ServerWithEndpoint, _ int) dto.Server {
	return dto.Server{
		ID:            uint(p.Id),
		Name:          p.Name,
		Namespace:     p.Namespace,
		Kind:          p.Kind,
		ObjectID:      p.ObjectId,
		ContainerName: p.ContainerName,
		Interval:      time.Duration(p.IntervalMs) * time.Millisecond,
		Timeout:       time.Duration(p.TimeoutMs) * time.Millisecond,
		HTTPConfig:    dtoHTTPConfig(p.HttpConfig),
	}
}

func dtoHTTPConfig(in *serverv1.HttpConfigInput) *dto.HTTPConfig {

	if in == nil {
		return nil
	}

	return &dto.HTTPConfig{
		Port:          int(in.Port),
		EndpointPath:  in.EndpointPath,
		ExpectedCode:  int(in.ExpectedCode),
		BodyCheckExpr: in.BodyCheckExpr,
		Method:        in.Method,
	}
}

func httpConfigFromRow(r dto.ImportRow) *serverv1.HttpConfigInput {

	if r.HTTPPort == 0 &&
		r.HTTPPath == "" &&
		r.HTTPMethod == "" &&

		r.HTTPBodyCheck == "" &&
		r.HTTPExpectedCode == 0 {
		return nil
	}

	return &serverv1.HttpConfigInput{
		Port:          int32(r.HTTPPort),
		EndpointPath:  r.HTTPPath,
		ExpectedCode:  int32(r.HTTPExpectedCode),
		BodyCheckExpr: r.HTTPBodyCheck,
		Method:        r.HTTPMethod,
	}
}
