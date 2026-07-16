package apperrors

import "errors"

var (
	ErrNotFound   = errors.New("resource not found")
	ErrInternal   = errors.New("an unexpected error occurred")
	ErrBadRequest = errors.New("invalid request")
	ErrForbidden  = errors.New("forbidden")
)
