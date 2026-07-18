package utils

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

var (
	ErrNameRequired    = errors.New("server_name is required")
	ErrNameTooLong     = errors.New("server_name must be at most 255 characters")
	ErrURLInvalid      = errors.New("url must start with http:// or https://")
	ErrURLParse        = errors.New("url is invalid")
	ErrIntervalInvalid = errors.New("interval_sec must be a positive integer")
	ErrTimeoutInvalid  = errors.New("timeout_sec must be a positive integer")
	ErrCodeOutOfRange  = errors.New("expected_code must be between 100 and 599")
)

func ValidateServerName(name string) error {

	name = strings.TrimSpace(name)
	if name == "" {
		return ErrNameRequired
	}

	if len(name) > 255 {
		return ErrNameTooLong
	}

	return nil
}

func ValidateURL(u string) error {

	u = strings.TrimSpace(u)
	if u == "" {
		return nil
	}

	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		return ErrURLInvalid
	}

	if _, err := url.Parse(u); err != nil {
		return ErrURLParse
	}

	return nil
}

var validMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "DELETE": true,
	"HEAD": true, "PATCH": true, "OPTIONS": true,
}

func ValidateMethod(method string) (string, error) {
	if method == "" {
		return "GET", nil
	}
	m := strings.ToUpper(method)
	if !validMethods[m] {
		return "", fmt.Errorf("method '%s' is not supported", method)
	}
	return m, nil
}

func ValidateInterval(sec int) error {
	if sec < 1 {
		return ErrIntervalInvalid
	}
	return nil
}

func ValidateTimeout(sec int) error {
	if sec < 1 {
		return ErrTimeoutInvalid
	}
	return nil
}

func ValidateExpectedCode(code int) error {
	if code < 100 || code > 599 {
		return ErrCodeOutOfRange
	}
	return nil
}
