package user

import (
	"context"

	"github.com/gibran/go-gin-boilerplate/internal/database"
	"github.com/gibran/go-gin-boilerplate/internal/pkg/pagination"
	"github.com/gibran/go-gin-boilerplate/internal/pkg/search"
	"github.com/gibran/go-gin-boilerplate/internal/pkg/sorting"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// sortFields maps the sort fields clients may use to database columns
var sortFields = map[string]string{
	"name":      "name",
	"email":     "email",
	"role":      "role",
	"createdAt": "created_at",
	"updatedAt": "updated_at",
}

// searchColumns are matched by the search query parameter
var searchColumns = []string{"name", "email"}

// Repository handles database operations for users
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new Repository
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// FindAll retrieves a page of users matching the search, in the requested order
func (r *Repository) FindAll(ctx context.Context, q ListQuery) ([]User, int64, error) {
	db, err := r.listQuery(ctx, q)
	if err != nil {
		return nil, 0, err
	}
	return pagination.Find[User](db, q.Query)
}

// listQuery builds the filtered and ordered query of FindAll, before pagination
func (r *Repository) listQuery(ctx context.Context, q ListQuery) (*gorm.DB, error) {
	db, err := sorting.Apply(database.Conn(ctx, r.db), q.Sort, sortFields, "-createdAt")
	if err != nil {
		return nil, err
	}
	return db.Scopes(search.Scope(q.Search, searchColumns...)), nil
}

// FindByID retrieves a user by their UUID
func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var user User
	if err := database.Conn(ctx, r.db).First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByEmail retrieves a user by their email
func (r *Repository) FindByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	if err := database.Conn(ctx, r.db).First(&user, "email = ?", email).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// Create inserts a new user record
func (r *Repository) Create(ctx context.Context, user *User) error {
	return database.Conn(ctx, r.db).Create(user).Error
}

// Update modifies an existing user record
func (r *Repository) Update(ctx context.Context, user *User) error {
	return database.Conn(ctx, r.db).Save(user).Error
}

// Delete removes a user record (soft delete due to gorm.DeletedAt)
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	return database.Conn(ctx, r.db).Delete(&User{}, "id = ?", id).Error
}
