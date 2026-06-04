package dto

type RegisterRequest struct {
	Email    string `validate:"required,email"`
	Username string `validate:"required,min=3,max=100"`
	Password string `validate:"required,min=8"`
	Name     string `validate:"required,min=1,max=255"`
}

type LoginRequest struct {
	Login    string `validate:"required"`
	Password string `validate:"required"`
}

type AuthResponse struct {
	Token string
	User  UserProfile
}

type UserProfile struct {
	ID       uint
	Email    string
	Username string
	Name     string
}
