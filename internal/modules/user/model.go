package user

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusActive    = "active"
	StatusSuspended = "suspended"
	StatusDeleted   = "deleted"
)

type User struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Email         string    `json:"email"`
	EmailVerified bool      `json:"email_verified"`
	AvatarURL     *string   `json:"avatar_url,omitempty"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type UpdateUserRequest struct {
	Name      *string `json:"name" binding:"omitempty,min=1,max=120"`
	AvatarURL *string `json:"avatar_url" binding:"omitempty,url"`
}
