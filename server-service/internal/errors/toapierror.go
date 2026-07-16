package apperrors

import (
	"errors"
	"net/http"

	"github.com/minhnbnt/uptime-monitor-microservices/server-service/generated/api"
)

func ToAPIError(err error) *api.ErrorResponseStatusCode {

	if errors.Is(err, ErrNotFound) {
		return &api.ErrorResponseStatusCode{
			StatusCode: http.StatusNotFound,
			Response:   errResponse("NOT_FOUND", err.Error()),
		}
	}

	if errors.Is(err, ErrForbidden) {
		return &api.ErrorResponseStatusCode{
			StatusCode: http.StatusForbidden,
			Response:   errResponse("FORBIDDEN", err.Error()),
		}
	}

	if errors.Is(err, ErrBadRequest) {
		return &api.ErrorResponseStatusCode{
			StatusCode: http.StatusBadRequest,
			Response:   errResponse("BAD_REQUEST", err.Error()),
		}
	}

	return &api.ErrorResponseStatusCode{
		StatusCode: http.StatusInternalServerError,
		Response:   errResponse("INTERNAL_ERROR", err.Error()),
	}
}

func errResponse(code, msg string) api.ErrorResponse {
	return api.ErrorResponse{
		Error: api.ErrorResponseError{
			Code:    code,
			Message: msg,
		},
	}
}
