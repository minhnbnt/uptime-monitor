package dto

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

type RefreshRequest struct {
	RefreshToken string
}

type UserProfile struct {
	ID       uint
	Email    string
	Username string
	Name     string
}
