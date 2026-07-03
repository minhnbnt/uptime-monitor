package apperrors

import (
	"errors"
	"net/http"
	"testing"
)

func TestToAPIError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "not found",
			err:        ErrNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   "NOT_FOUND",
		},
		{
			name:       "email or username taken",
			err:        ErrEmailOrUsernameTaken,
			wantStatus: http.StatusConflict,
			wantCode:   "CONFLICT",
		},
		{
			name:       "invalid credentials",
			err:        ErrInvalidCredentials,
			wantStatus: http.StatusUnauthorized,
			wantCode:   "UNAUTHORIZED",
		},
		{
			name:       "invalid access token",
			err:        ErrInvalidAccessToken,
			wantStatus: http.StatusUnauthorized,
			wantCode:   "INVALID_ACCESS_TOKEN",
		},
		{
			name:       "invalid refresh token",
			err:        ErrInvalidRefreshToken,
			wantStatus: http.StatusUnauthorized,
			wantCode:   "INVALID_REFRESH_TOKEN",
		},
		{
			name:       "bad request",
			err:        ErrBadRequest,
			wantStatus: http.StatusBadRequest,
			wantCode:   "BAD_REQUEST",
		},
		{
			name:       "internal error",
			err:        ErrInternal,
			wantStatus: http.StatusInternalServerError,
			wantCode:   "INTERNAL_ERROR",
		},
		{
			name:       "wrapped not found",
			err:        errors.New("server 42: resource not found"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "INTERNAL_ERROR",
		},
		{
			name:       "unknown error",
			err:        errors.New("something went wrong"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToAPIError(tt.err)
			if got.StatusCode != tt.wantStatus {
				t.Errorf("StatusCode = %d, want %d", got.StatusCode, tt.wantStatus)
			}
			if got.Response.Error.Code != tt.wantCode {
				t.Errorf("Code = %q, want %q", got.Response.Error.Code, tt.wantCode)
			}
		})
	}
}

func TestToAPIError_usesErrorsIs(t *testing.T) {
	sentinel := errors.New("custom: resource not found")
	wrapped := &wrapError{msg: "wrapped: resource not found", err: ErrNotFound}

	result := ToAPIError(wrapped)
	if result.StatusCode != http.StatusNotFound {
		t.Errorf("expected StatusNotFound for wrapped ErrNotFound, got %d", result.StatusCode)
	}

	result2 := ToAPIError(sentinel)
	if result2.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected StatusInternalServerError for non-wrapped sentinel, got %d", result2.StatusCode)
	}
}

type wrapError struct {
	msg string
	err error
}

func (w *wrapError) Error() string { return w.msg }
func (w *wrapError) Unwrap() error { return w.err }

func TestErrResponse(t *testing.T) {
	resp := errResponse("TEST_CODE", "test message")
	if resp.Error.Code != "TEST_CODE" {
		t.Errorf("Code = %q, want %q", resp.Error.Code, "TEST_CODE")
	}
	if resp.Error.Message != "test message" {
		t.Errorf("Message = %q, want %q", resp.Error.Message, "test message")
	}
}
