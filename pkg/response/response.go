package response

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// Meta holds pagination metadata
type Meta struct {
	CurrentPage      int `json:"currentPage"`
	PerPage          int `json:"perPage"`
	TotalCurrentPage int `json:"totalCurrentPage"`
	TotalPage        int `json:"totalPage"`
	TotalData        int `json:"totalData"`
}

// Response is the standard API response structure
type Response struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// PaginatedResponse is the standard API response for list/paginated data
type PaginatedResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Meta    Meta        `json:"meta"`
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
func SuccessPaginated(c *gin.Context, message string, data interface{}, meta Meta) {
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

// ValidationError sends a 400 Bad Request response with detailed field errors
func ValidationError(c *gin.Context, err error) {
	errs := make(map[string]string)

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrors {
			field := e.Field()
			switch e.Tag() {
			case "required":
				errs[field] = fmt.Sprintf("%s is required", field)
			case "email":
				errs[field] = fmt.Sprintf("%s must be a valid email address", field)
			case "min":
				errs[field] = fmt.Sprintf("%s must be at least %s characters long", field, e.Param())
			case "max":
				errs[field] = fmt.Sprintf("%s must be at most %s characters long", field, e.Param())
			case "oneof":
				errs[field] = fmt.Sprintf("%s must be one of: %s", field, e.Param())
			default:
				errs[field] = fmt.Sprintf("%s is invalid", field)
			}
		}
	} else {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Status:  "error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusBadRequest, ErrorResponse{
		Status:  "error",
		Message: "Validation failed",
		Errors:  errs,
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
