package auth

import (
	service "github.com/gibran/go-gin-boilerplate/internal/service/auth"
	"github.com/gibran/go-gin-boilerplate/pkg/response"
	"github.com/gin-gonic/gin"
)

// Handler handles authentication requests
type Handler struct {
	service *service.AuthService
}

// NewHandler creates a new Auth Handler
func NewHandler(s *service.AuthService) *Handler {
	return &Handler{service: s}
}

// Register handles POST /auth/register
// @Summary Register a new user
// @Description Create a new user account with name, email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body service.RegisterRequest true "Registration details"
// @Success 201 {object} response.Response{data=model.User}
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req service.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	user, err := h.service.Register(req)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Created(c, "User registered successfully", user)
}

// Login handles POST /auth/login
// @Summary Login user
// @Description Authenticate user and return access & refresh tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param request body service.LoginRequest true "Login credentials"
// @Success 200 {object} response.Response{data=service.AuthResponse}
// @Failure 401 {object} response.ErrorResponse
// @Router /auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req service.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	res, err := h.service.Login(req)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	response.Success(c, "Login successful", res)
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// Refresh handles POST /auth/refresh
// @Summary Refresh access token
// @Description Get a new access token using a refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RefreshRequest true "Refresh token"
// @Success 200 {object} response.Response{data=map[string]string}
// @Failure 401 {object} response.ErrorResponse
// @Router /auth/refresh [post]
func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	accessToken, err := h.service.RefreshToken(req.RefreshToken)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	response.Success(c, "Token refreshed successfully", gin.H{
		"accessToken": accessToken,
	})
}

// Logout handles POST /auth/logout
// @Summary Logout user
// @Description Log out the current user (placeholder for stateless JWT)
// @Tags auth
// @Produce json
// @Success 200 {object} response.Response
// @Security BearerAuth
// @Router /auth/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	// In a stateless JWT setup, logout is usually handled by the client 
	// (deleting the token). For more security, one could blacklist tokens.
	response.Success(c, "Logged out successfully", nil)
}
