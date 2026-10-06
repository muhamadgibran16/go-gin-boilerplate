package user

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

// User represents the user entity in the database.
// It is never serialized directly; API responses use UserResponse.
type User struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;"`
	Name      string    `gorm:"size:255;not null"`
	Email     string    `gorm:"size:255;not null;uniqueIndex"`
	Password  string    `gorm:"size:255;not null" json:"-"`
	Role      string    `gorm:"size:50;not null;default:'user'"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// BeforeCreate is a GORM hook to set the UUID before creating a record
func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return
}
