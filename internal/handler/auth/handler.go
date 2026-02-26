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
func (h *Handler) Logout(c *gin.Context) {
	// In a stateless JWT setup, logout is usually handled by the client 
	// (deleting the token). For more security, one could blacklist tokens.
	response.Success(c, "Logged out successfully", nil)
}
