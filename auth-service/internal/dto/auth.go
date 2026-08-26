package dto

import (
	"time"

	"github.com/google/uuid"
)

type RegisterRequest struct {
	Email    string
	Username string
	Password string
	Name     string
}

type LoginRequest struct {
	Login    string
	Password string
}

type AuthResponse struct {
	AccessToken  string
	RefreshToken string
	User         UserProfile
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	JTI          uuid.UUID
}

type RefreshRequest struct {
	RefreshToken string
}

type UserProfile struct {
	ID       uint
	Email    string
	Username string
	Name     string
}

type SessionInfo struct {
	ID        string
	Scopes    []string
	Current   bool
	CreatedAt time.Time
	ExpiresAt time.Time
}
