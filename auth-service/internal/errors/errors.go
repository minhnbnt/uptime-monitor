package apperrors

import "errors"

var (
	ErrNotFound             = errors.New("resource not found")
	ErrInternal             = errors.New("an unexpected error occurred")
	ErrEmailOrUsernameTaken = errors.New("email or username already exists")
	ErrInvalidCredentials   = errors.New("invalid email/username or password")
	ErrInvalidAccessToken   = errors.New("invalid or expired access token")
	ErrInvalidRefreshToken  = errors.New("invalid or expired refresh token")
	// ErrSessionRotated means the presented session was already rotated away by
	// another request — a replayed or duplicated refresh.
	ErrSessionRotated = errors.New("session already rotated")
	ErrBadRequest     = errors.New("invalid request")
	ErrForbidden      = errors.New("forbidden")
)
