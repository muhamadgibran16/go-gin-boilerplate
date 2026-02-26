package user

import (
	service "github.com/gibran/go-gin-boilerplate/internal/service/user"
	"github.com/gibran/go-gin-boilerplate/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler handles user management requests
type Handler struct {
	service *service.UserService
}

// NewHandler creates a new User Handler
func NewHandler(s *service.UserService) *Handler {
	return &Handler{service: s}
}

type UserQuery struct {
	Page    int `form:"page,default=1" binding:"omitempty,min=1"`
	PerPage int `form:"perPage,default=10" binding:"omitempty,min=1,max=100"`
}

// GetMany handles GET /users
func (h *Handler) GetMany(c *gin.Context) {
	var query UserQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.ValidationError(c, err)
		return
	}

	users, total, err := h.service.GetAllUsers(query.Page, query.PerPage)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	totalPage := int((total + int64(query.PerPage) - 1) / int64(query.PerPage))

	response.SuccessPaginated(c, "Get users successfully", users, response.Meta{
		CurrentPage:      query.Page,
		PerPage:          query.PerPage,
		TotalCurrentPage: len(users),
		TotalPage:        totalPage,
		TotalData:        int(total),
	})
}

// GetOne handles GET /users/:id
func (h *Handler) GetOne(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID format")
		return
	}

	user, err := h.service.GetUserByID(id)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}

	response.Success(c, "Get user successfully", user)
}

// Update handles PUT /users/:id
func (h *Handler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID format")
		return
	}

	var req service.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	user, err := h.service.UpdateUser(id, req)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, "User updated successfully", user)
}

// Delete handles DELETE /users/:id
func (h *Handler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID format")
		return
	}

	err = h.service.DeleteUser(id)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, "User deleted successfully", nil)
}
