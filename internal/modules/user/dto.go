package user

import (
	"time"

	"github.com/gibran/go-gin-boilerplate/internal/pkg/pagination"
	"github.com/google/uuid"
)

// ListQuery is the query string of GET /users:
// ?page=1&perPage=10&sort=-createdAt,name&search=bob
type ListQuery struct {
	pagination.Query
	Sort   string `form:"sort" binding:"omitempty,max=200"`
	Search string `form:"search" binding:"omitempty,max=100"`
}

// UpdateRequest is the body of PUT /users/:id
type UpdateRequest struct {
	// The minimum length is checked by the service after trimming spaces
	Name string `json:"name" binding:"omitempty,max=255"`
	Role string `json:"role" binding:"omitempty,oneof=admin user"`
}

// UserResponse is the public representation of a user
type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// NewUserResponse maps a User to its public representation
func NewUserResponse(u *User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		Role:      u.Role,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

// NewUserResponses maps a list of users to their public representation
func NewUserResponses(users []User) []UserResponse {
	res := make([]UserResponse, len(users))
	for i := range users {
		res[i] = NewUserResponse(&users[i])
	}
	return res
}
