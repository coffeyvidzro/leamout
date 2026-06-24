package auth

import (
	"time"

	"github.com/google/uuid"
)

type AuthUser struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Email         string    `json:"email"`
	EmailVerified bool      `json:"email_verified"`
	AvatarURL     *string   `json:"avatar_url,omitempty"`
	Status        string    `json:"status"`
}

type AuthSession struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type AuthResponse struct {
	User    AuthUser    `json:"user"`
	Session AuthSession `json:"session"`
}

type OAuthLoginRequest struct {
	Provider  string
	Code      string
	State     string
	UserAgent string
	IPAddress string
}
