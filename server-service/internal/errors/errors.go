package apperrors

import "errors"

var (
	ErrNotFound     = errors.New("resource not found")
	ErrInternal     = errors.New("an unexpected error occurred")
	ErrBadRequest   = errors.New("invalid request")
	ErrForbidden    = errors.New("forbidden")
	ErrYAMLInvalid  = errors.New("invalid k8s resource YAML")
	ErrPodMonitored = errors.New("pod is still monitored by a server record")
)
