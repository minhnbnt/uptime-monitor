package apperrors

import "errors"

var (
	ErrNotFound    = errors.New("resource not found")
	ErrInternal    = errors.New("an unexpected error occurred")
	ErrBadRequest  = errors.New("invalid request")
	ErrForbidden   = errors.New("forbidden")
	ErrNotManaged  = errors.New("object is not managed by this system")
	ErrYAMLInvalid = errors.New("invalid k8s resource YAML")
)
