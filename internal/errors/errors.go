package apperrors

import "errors"

var (
	ErrNotFound             = errors.New("resource not found")
	ErrInternal             = errors.New("an unexpected error occurred")
	ErrEmailOrUsernameTaken = errors.New("email or username already exists")
	ErrInvalidCredentials   = errors.New("invalid email/username or password")
	ErrBadRequest           = errors.New("invalid request")
)
