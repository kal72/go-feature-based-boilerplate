package dto

// LoginRequest carries credentials for authentication.
type LoginRequest struct {
	Email    string `validate:"required,email"`
	Password string `validate:"required"`
}

// RegisterRequest carries data for new account creation.
type RegisterRequest struct {
	Name     string `validate:"required,min=2,max=100"`
	Email    string `validate:"required,email"`
	Password string `validate:"required,min=8"`
}

// RefreshTokenRequest carries a refresh token to exchange for new tokens.
type RefreshTokenRequest struct {
	RefreshToken string `validate:"required"`
}

// LogoutRequest carries the refresh token to revoke on logout.
type LogoutRequest struct {
	RefreshToken string `validate:"required"`
}
