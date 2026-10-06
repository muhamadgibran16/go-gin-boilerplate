package auth

import (
	"context"

	"github.com/gibran/go-gin-boilerplate/internal/httpx"
	"github.com/gibran/go-gin-boilerplate/internal/modules/user"
	"github.com/gin-gonic/gin"
)

// service is the business logic the handler needs. *Service satisfies it.
type service interface {
	Register(ctx context.Context, name, email, password string) (*user.User, error)
	Login(ctx context.Context, email, password string) (*user.User, *Tokens, error)
	Refresh(ctx context.Context, refreshToken string) (string, error)
}

// Handler handles authentication requests.
// Errors are passed to httpx.Abort; httpx.ErrorHandler writes the response.
type Handler struct {
	service service
}

// NewHandler creates a new Handler
func NewHandler(s service) *Handler {
	return &Handler{service: s}
}

// Register handles POST /auth/register
// @Summary Register a new user
// @Description Create a new user account with name, email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body auth.RegisterRequest true "Registration details"
// @Success 201 {object} httpx.Response{data=user.UserResponse}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if !httpx.BindJSON(c, &req) {
		return
	}

	u, err := h.service.Register(c.Request.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		httpx.Abort(c, err)
		return
	}

	httpx.Created(c, "User registered successfully", user.NewUserResponse(u))
}

// Login handles POST /auth/login
// @Summary Login user
// @Description Authenticate user and return access & refresh tokens.
// @Description After 5 failed attempts on an email within 15 minutes, logins to it are refused (429) until the window ends.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body auth.LoginRequest true "Login credentials"
// @Success 200 {object} httpx.Response{data=auth.LoginResponse}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 429 {object} httpx.ErrorResponse "Too many requests from this IP, or too many failed logins for this email"
// @Router /auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if !httpx.BindJSON(c, &req) {
		return
	}

	u, tokens, err := h.service.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		httpx.Abort(c, err)
		return
	}

	httpx.Success(c, "Login successful", LoginResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User:         user.NewUserResponse(u),
	})
}

// Refresh handles POST /auth/refresh
// @Summary Refresh access token
// @Description Get a new access token using a refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body auth.RefreshRequest true "Refresh token"
// @Success 200 {object} httpx.Response{data=auth.RefreshResponse}
// @Failure 401 {object} httpx.ErrorResponse
// @Router /auth/refresh [post]
func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if !httpx.BindJSON(c, &req) {
		return
	}

	accessToken, err := h.service.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		httpx.Abort(c, err)
		return
	}

	httpx.Success(c, "Token refreshed successfully", RefreshResponse{AccessToken: accessToken})
}

// Logout handles POST /auth/logout
// @Summary Logout user
// @Description Log out the current user (placeholder for stateless JWT)
// @Tags auth
// @Produce json
// @Success 200 {object} httpx.Response
// @Security BearerAuth
// @Router /auth/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	// In a stateless JWT setup, logout is usually handled by the client
	// (deleting the token). For more security, one could blacklist tokens.
	httpx.Success(c, "Logged out successfully", nil)
}
