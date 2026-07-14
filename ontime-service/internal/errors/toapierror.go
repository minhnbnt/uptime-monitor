package apperrors

import (
	"errors"
	"net/http"
)

type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func ToAPIError(err error) (int, map[string]any) {

	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound, map[string]any{"error": "NOT_FOUND", "message": err.Error()}

	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden, map[string]any{"error": "FORBIDDEN", "message": err.Error()}

	case errors.Is(err, ErrBadRequest):
		return http.StatusBadRequest, map[string]any{"error": "BAD_REQUEST", "message": err.Error()}

	case errors.Is(err, ErrEmailOrUsernameTaken):
		return http.StatusConflict, map[string]any{"error": "CONFLICT", "message": err.Error()}

	case errors.Is(err, ErrInvalidCredentials):
		return http.StatusUnauthorized, map[string]any{"error": "UNAUTHORIZED", "message": err.Error()}

	default:
		return http.StatusInternalServerError, map[string]any{"error": "INTERNAL_ERROR", "message": err.Error()}
	}
}
