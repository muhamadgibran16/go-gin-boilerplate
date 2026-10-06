package auth

import "github.com/gibran/go-gin-boilerplate/internal/modules/user"

// RegisterRequest is the body of POST /auth/register
type RegisterRequest struct {
	Name  string `json:"name" binding:"required,max=255"`
	Email string `json:"email" binding:"required,email,max=255"`
	// Length rules (at least 6 characters, at most 72 bytes) are checked by the service
	Password string `json:"password" binding:"required"`
}

// LoginRequest is the body of POST /auth/login
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RefreshRequest is the body of POST /auth/refresh
type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// LoginResponse is returned by POST /auth/login
type LoginResponse struct {
	AccessToken  string            `json:"accessToken"`
	RefreshToken string            `json:"refreshToken"`
	User         user.UserResponse `json:"user"`
}

// RefreshResponse is returned by POST /auth/refresh
type RefreshResponse struct {
	AccessToken string `json:"accessToken"`
}
