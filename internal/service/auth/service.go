package service

import (
	"errors"

	"github.com/gibran/go-gin-boilerplate/config"
	"github.com/gibran/go-gin-boilerplate/internal/model"
	repository "github.com/gibran/go-gin-boilerplate/internal/repository/user"
	"github.com/gibran/go-gin-boilerplate/pkg/security"
)

// AuthService handles authentication logic
type AuthService struct {
	repo   *repository.UserRepository
	config *config.Config
}

// NewAuthService creates a new AuthService
func NewAuthService(repo *repository.UserRepository, cfg *config.Config) *AuthService {
	return &AuthService{repo: repo, config: cfg}
}

type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	AccessToken  string      `json:"accessToken"`
	RefreshToken string      `json:"refreshToken"`
	User         *model.User `json:"user"`
}

// Register creates a new user and returns their data
func (s *AuthService) Register(req RegisterRequest) (*model.User, error) {
	// Check if user already exists
	existing, _ := s.repo.FindByEmail(req.Email)
	if existing != nil {
		return nil, errors.New("email already registered")
	}

	// Hash password
	hashedPassword, err := security.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: hashedPassword,
		Role:     "user",
	}

	err = s.repo.Create(user)
	return user, err
}

// Login validates credentials and generates JWT tokens
func (s *AuthService) Login(req LoginRequest) (*AuthResponse, error) {
	user, err := s.repo.FindByEmail(req.Email)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	// Compare password
	if !security.ComparePassword(user.Password, req.Password) {
		return nil, errors.New("invalid email or password")
	}

	// Generate tokens
	accessToken, err := security.GenerateToken(user.ID, user.Role, s.config.JWTSecret, s.config.JWTAccessExpireHours)
	if err != nil {
		return nil, err
	}

	// For boilerplate simplicity, we'll use a double-length JWT as refresh token
	// In production, consider a separate storage or UUID for refresh tokens
	refreshToken, err := security.GenerateToken(user.ID, user.Role, s.config.JWTSecret, s.config.JWTRefreshExpireDays*24)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}

// RefreshToken validates a refresh token and generates a new access token
func (s *AuthService) RefreshToken(tokenString string) (string, error) {
	claims, err := security.ValidateToken(tokenString, s.config.JWTSecret)
	if err != nil {
		return "", errors.New("invalid or expired refresh token")
	}

	// Generate new access token
	accessToken, err := security.GenerateToken(claims.UserID, claims.Role, s.config.JWTSecret, s.config.JWTAccessExpireHours)
	if err != nil {
		return "", err
	}

	return accessToken, nil
}
