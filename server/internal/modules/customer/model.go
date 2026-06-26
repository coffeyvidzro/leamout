package customer

import (
	"time"

	"github.com/google/uuid"
)

type Customer struct {
	ID         uuid.UUID      `json:"id"`
	UserID     uuid.UUID      `json:"user_id"`
	Name       string         `json:"name"`
	Email      *string        `json:"email,omitempty"`
	Phone      string         `json:"phone"`
	ExternalID *string        `json:"external_id,omitempty"`
	Address    Address        `json:"address"`
	Metadata   map[string]any `json:"metadata"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

type Address struct {
	Line1      string `json:"line1,omitempty"`
	Line2      string `json:"line2,omitempty"`
	City       string `json:"city,omitempty"`
	State      string `json:"state,omitempty"`
	PostalCode string `json:"postal_code,omitempty"`
	Country    string `json:"country,omitempty"`
}

type CreateRequest struct {
	Name       string         `json:"name" binding:"required,min=1,max=160"`
	Email      *string        `json:"email" binding:"omitempty,email"`
	Phone      string         `json:"phone" binding:"required,min=3,max=40"`
	ExternalID *string        `json:"external_id" binding:"omitempty,max=160"`
	Address    Address        `json:"address"`
	Metadata   map[string]any `json:"metadata"`
}

type UpdateRequest struct {
	Name       *string        `json:"name,omitempty" binding:"omitempty,min=1,max=160"`
	Email      *string        `json:"email,omitempty" binding:"omitempty,email"`
	Phone      *string        `json:"phone,omitempty" binding:"omitempty,min=3,max=40"`
	ExternalID *string        `json:"external_id,omitempty" binding:"omitempty,max=160"`
	Address    *Address       `json:"address,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}
