package apperrors

import "errors"

var (
	ErrNotFound             = errors.New("resource not found")
	ErrInternal             = errors.New("an unexpected error occurred")
	ErrEmailOrUsernameTaken = errors.New("email or username already exists")
	ErrInvalidCredentials   = errors.New("invalid email/username or password")
	ErrInvalidAccessToken   = errors.New("invalid or expired access token")
	ErrInvalidRefreshToken  = errors.New("invalid or expired refresh token")
	ErrBadRequest           = errors.New("invalid request")
)
