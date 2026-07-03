package apperrors

import (
	"errors"
	"net/http"

	"github.com/minhnbnt/uptime-monitor/generated/api"
)

func ToAPIError(err error) *api.ErrorResponseStatusCode {

	if errors.Is(err, ErrNotFound) {
		return &api.ErrorResponseStatusCode{
			StatusCode: http.StatusNotFound,
			Response:   errResponse("NOT_FOUND", err.Error()),
		}
	}

	if errors.Is(err, ErrEmailOrUsernameTaken) {
		return &api.ErrorResponseStatusCode{
			StatusCode: http.StatusConflict,
			Response:   errResponse("CONFLICT", err.Error()),
		}
	}

	if errors.Is(err, ErrInvalidCredentials) {
		return &api.ErrorResponseStatusCode{
			StatusCode: http.StatusUnauthorized,
			Response:   errResponse("UNAUTHORIZED", err.Error()),
		}
	}

	if errors.Is(err, ErrInvalidAccessToken) {
		return &api.ErrorResponseStatusCode{
			StatusCode: http.StatusUnauthorized,
			Response:   errResponse("INVALID_ACCESS_TOKEN", err.Error()),
		}
	}

	if errors.Is(err, ErrInvalidRefreshToken) {
		return &api.ErrorResponseStatusCode{
			StatusCode: http.StatusUnauthorized,
			Response:   errResponse("INVALID_REFRESH_TOKEN", err.Error()),
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
