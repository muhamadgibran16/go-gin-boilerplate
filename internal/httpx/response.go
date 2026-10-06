package httpx

import (
	"net/http"

	"github.com/gibran/go-gin-boilerplate/internal/pkg/pagination"
	"github.com/gin-gonic/gin"
)

// Response is the standard API response structure
type Response struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// PaginatedResponse is the standard API response for list/paginated data
type PaginatedResponse struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Data    interface{}     `json:"data"`
	Meta    pagination.Meta `json:"meta"`
}

// ErrorResponse is the standard API error response structure
type ErrorResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Errors  interface{} `json:"errors,omitempty"`
}

// Success sends a 200 OK response with data
func Success(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Status:  "success",
		Message: message,
		Data:    data,
	})
}

// SuccessPaginated sends a 200 OK response with paginated data and meta
func SuccessPaginated(c *gin.Context, message string, data interface{}, meta pagination.Meta) {
	c.JSON(http.StatusOK, PaginatedResponse{
		Status:  "success",
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

// Created sends a 201 Created response with data
func Created(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusCreated, Response{
		Status:  "success",
		Message: message,
		Data:    data,
	})
}

// BadRequest sends a 400 Bad Request response
func BadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, ErrorResponse{
		Status:  "error",
		Message: message,
	})
}

// Unauthorized sends a 401 Unauthorized response
func Unauthorized(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, ErrorResponse{
		Status:  "error",
		Message: message,
	})
}

// Forbidden sends a 403 Forbidden response
func Forbidden(c *gin.Context, message string) {
	c.JSON(http.StatusForbidden, ErrorResponse{
		Status:  "error",
		Message: message,
	})
}

// NotFound sends a 404 Not Found response
func NotFound(c *gin.Context, message string) {
	c.JSON(http.StatusNotFound, ErrorResponse{
		Status:  "error",
		Message: message,
	})
}

// InternalServerError sends a 500 Internal Server Error response
func InternalServerError(c *gin.Context, message string) {
	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Status:  "error",
		Message: message,
	})
}

// TooManyRequests sends a 429 Too Many Requests response
func TooManyRequests(c *gin.Context, message string) {
	c.JSON(http.StatusTooManyRequests, ErrorResponse{
		Status:  "error",
		Message: message,
	})
}

// Conflict sends a 409 Conflict response
func Conflict(c *gin.Context, message string) {
	c.JSON(http.StatusConflict, ErrorResponse{
		Status:  "error",
		Message: message,
	})
}

// ServiceUnavailable sends a 503 Service Unavailable response
func ServiceUnavailable(c *gin.Context, message string) {
	c.JSON(http.StatusServiceUnavailable, ErrorResponse{
		Status:  "error",
		Message: message,
	})
}
