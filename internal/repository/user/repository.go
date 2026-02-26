package repository

import (
	"github.com/gibran/go-gin-boilerplate/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRepository handles database operations for Users
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new UserRepository
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// FindAll retrieves users with pagination
func (r *UserRepository) FindAll(page, perPage int) ([]model.User, int64, error) {
	var users []model.User
	var total int64

	r.db.Model(&model.User{}).Count(&total)

	offset := (page - 1) * perPage
	err := r.db.Limit(perPage).Offset(offset).Order("created_at desc").Find(&users).Error

	return users, total, err
}

// FindByID retrieves a user by their UUID
func (r *UserRepository) FindByID(id uuid.UUID) (*model.User, error) {
	var user model.User
	err := r.db.First(&user, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByEmail retrieves a user by their email
func (r *UserRepository) FindByEmail(email string) (*model.User, error) {
	var user model.User
	err := r.db.First(&user, "email = ?", email).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Create inserts a new user record
func (r *UserRepository) Create(user *model.User) error {
	return r.db.Create(user).Error
}

// Update modifies an existing user record
func (r *UserRepository) Update(user *model.User) error {
	return r.db.Save(user).Error
}

// Delete removes a user record (soft delete due to GORM model)
func (r *UserRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&model.User{}, "id = ?", id).Error
}
