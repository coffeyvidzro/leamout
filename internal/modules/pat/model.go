package pat

import (
	"time"

	"github.com/google/uuid"
)

const TokenPrefix = "lmt_pat_"

type Token struct {
	ID         uuid.UUID      `json:"id"`
	UserID     uuid.UUID      `json:"user_id"`
	Name       string         `json:"name"`
	LastUsedAt *time.Time     `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time     `json:"expires_at,omitempty"`
	RevokedAt  *time.Time     `json:"revoked_at,omitempty"`
	Metadata   map[string]any `json:"metadata"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

type CreateRequest struct {
	Name      string         `json:"name" binding:"required,min=1,max=160"`
	ExpiresAt *time.Time     `json:"expires_at"`
	Metadata  map[string]any `json:"metadata"`
}

type CreateResponse struct {
	Token    Token  `json:"token"`
	RawToken string `json:"raw_token"`
}
