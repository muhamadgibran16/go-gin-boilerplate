package user

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/gibran/go-gin-boilerplate/internal/pkg/apperror"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrNotFound = apperror.NotFound("user not found")
	// ErrSelfModification prevents admins from locking themselves (and possibly everyone) out
	ErrSelfModification = apperror.Forbidden("you cannot change your own role or delete your own account")
)

// repository is the data access the service needs. *Repository satisfies it; tests use a fake.
type repository interface {
	FindAll(ctx context.Context, q ListQuery) ([]User, int64, error)
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uuid.UUID) error
}

const minNameLength = 3

// UpdateInput holds the fields that can be changed on a user. Empty fields are left unchanged.
type UpdateInput struct {
	Name string
	Role string
}

// Service handles user management logic
type Service struct {
	repo repository
}

// NewService creates a new Service
func NewService(repo repository) *Service {
	return &Service{repo: repo}
}

// List retrieves a page of users, filtered and sorted by q
func (s *Service) List(ctx context.Context, q ListQuery) ([]User, int64, error) {
	return s.repo.FindAll(ctx, q)
}

// Get retrieves a single user
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*User, error) {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return user, nil
}

// Update modifies user details on behalf of actorID
func (s *Service) Update(ctx context.Context, actorID, id uuid.UUID, in UpdateInput) (*User, error) {
	user, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if actorID == id && in.Role != "" && in.Role != user.Role {
		return nil, ErrSelfModification
	}

	if in.Name != "" {
		name := strings.TrimSpace(in.Name)
		if utf8.RuneCountInString(name) < minNameLength {
			return nil, apperror.Validation(map[string]string{
				"name": fmt.Sprintf("name must be at least %d characters long", minNameLength),
			})
		}
		user.Name = name
	}
	if in.Role != "" {
		user.Role = in.Role
	}

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

// Delete removes a user on behalf of actorID
func (s *Service) Delete(ctx context.Context, actorID, id uuid.UUID) error {
	if actorID == id {
		return ErrSelfModification
	}
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}
