package product

import (
	"time"

	"github.com/cuffeyvidzro/leamout/internal/modules/benefit"
	"github.com/cuffeyvidzro/leamout/internal/modules/price"
	"github.com/google/uuid"
)

type Product struct {
	ID          uuid.UUID         `json:"id"`
	UserID      uuid.UUID         `json:"user_id"`
	Name        string            `json:"name"`
	Description *string           `json:"description,omitempty"`
	Active      bool              `json:"active"`
	Metadata    map[string]any    `json:"metadata"`
	Prices      []price.Price     `json:"prices"`
	Benefits    []benefit.Benefit `json:"benefits"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type CreateRequest struct {
	Name        string                `json:"name" binding:"required,min=1,max=160"`
	Description *string               `json:"description" binding:"omitempty,max=1000"`
	Active      *bool                 `json:"active"`
	Metadata    map[string]any        `json:"metadata"`
	Prices      []price.CreateRequest `json:"prices" binding:"omitempty,dive"`
}

type UpdateRequest struct {
	Name        *string        `json:"name,omitempty" binding:"omitempty,min=1,max=160"`
	Description *string        `json:"description,omitempty" binding:"omitempty,max=1000"`
	Active      *bool          `json:"active,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type UpdateBenefitsRequest struct {
	Benefits []uuid.UUID `json:"benefits" binding:"required"`
}
