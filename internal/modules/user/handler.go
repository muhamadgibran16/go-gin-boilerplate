package user

import (
	"context"

	"github.com/gibran/go-gin-boilerplate/internal/httpx"
	"github.com/gibran/go-gin-boilerplate/internal/pkg/apperror"
	"github.com/gibran/go-gin-boilerplate/internal/pkg/pagination"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// service is the business logic the handler needs. *Service satisfies it; tests use a stub.
type service interface {
	List(ctx context.Context, q ListQuery) ([]User, int64, error)
	Get(ctx context.Context, id uuid.UUID) (*User, error)
	Update(ctx context.Context, actorID, id uuid.UUID, in UpdateInput) (*User, error)
	Delete(ctx context.Context, actorID, id uuid.UUID) error
}

// Handler handles user management requests.
// Errors are passed to httpx.Abort; httpx.ErrorHandler writes the response.
type Handler struct {
	service service
}

// NewHandler creates a new Handler
func NewHandler(s service) *Handler {
	return &Handler{service: s}
}

// GetMany handles GET /users
// @Summary List users
// @Description Get a paginated list of users (Admin only)
// @Tags users
// @Accept json
// @Produce json
// @Param page query int false "Page number (default 1)"
// @Param perPage query int false "Items per page (default 10, max 100)"
// @Param sort query string false "Comma-separated fields, '-' for descending. Fields: name, email, role, createdAt, updatedAt (default -createdAt)"
// @Param search query string false "Case-insensitive search in name and email"
// @Success 200 {object} httpx.PaginatedResponse{data=[]user.UserResponse}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Security BearerAuth
// @Router /users [get]
func (h *Handler) GetMany(c *gin.Context) {
	var query ListQuery
	if !httpx.BindQuery(c, &query) {
		return
	}

	users, total, err := h.service.List(c.Request.Context(), query)
	if err != nil {
		httpx.Abort(c, err)
		return
	}

	httpx.SuccessPaginated(c, "Get users successfully", NewUserResponses(users), pagination.NewMeta(query.Query, len(users), total))
}

// GetOne handles GET /users/:id
// @Summary Get user by ID
// @Description Get detailed information about a specific user (Admin only)
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User UUID"
// @Success 200 {object} httpx.Response{data=user.UserResponse}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Security BearerAuth
// @Router /users/{id} [get]
func (h *Handler) GetOne(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}

	user, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		httpx.Abort(c, err)
		return
	}

	httpx.Success(c, "Get user successfully", NewUserResponse(user))
}

// Update handles PUT /users/:id
// @Summary Update user
// @Description Update user name or role (Admin only). Admins cannot change their own role.
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User UUID"
// @Param request body user.UpdateRequest true "Update details"
// @Success 200 {object} httpx.Response{data=user.UserResponse}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Security BearerAuth
// @Router /users/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}

	var req UpdateRequest
	if !httpx.BindJSON(c, &req) {
		return
	}

	actorID, ok := httpx.CurrentUserID(c)
	if !ok {
		httpx.Abort(c, apperror.Unauthorized("Unauthenticated"))
		return
	}

	user, err := h.service.Update(c.Request.Context(), actorID, id, UpdateInput(req))
	if err != nil {
		httpx.Abort(c, err)
		return
	}

	httpx.Success(c, "User updated successfully", NewUserResponse(user))
}

// Delete handles DELETE /users/:id
// @Summary Delete user
// @Description Soft delete a user by ID (Admin only). Admins cannot delete their own account.
// @Tags users
// @Produce json
// @Param id path string true "User UUID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Security BearerAuth
// @Router /users/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}

	actorID, ok := httpx.CurrentUserID(c)
	if !ok {
		httpx.Abort(c, apperror.Unauthorized("Unauthenticated"))
		return
	}

	if err := h.service.Delete(c.Request.Context(), actorID, id); err != nil {
		httpx.Abort(c, err)
		return
	}

	httpx.Success(c, "User deleted successfully", nil)
}
