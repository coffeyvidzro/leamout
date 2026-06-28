package benefit

import (
	"time"

	"github.com/google/uuid"
)

type Type string

type GrantSourceType string

type GrantStatus string

const (
	TypeCustom      Type = "custom"
	TypeFeature     Type = "feature"
	TypeMeterCredit Type = "meter_credit"

	GrantSourceSubscription GrantSourceType = "subscription"
	GrantSourceManual       GrantSourceType = "manual"

	GrantStatusActive  GrantStatus = "active"
	GrantStatusRevoked GrantStatus = "revoked"
	GrantStatusExpired GrantStatus = "expired"
)

type Benefit struct {
	ID          uuid.UUID      `json:"id"`
	UserID      uuid.UUID      `json:"user_id"`
	Type        Type           `json:"type"`
	Name        string         `json:"name"`
	Code        string         `json:"code"`
	Description *string        `json:"description,omitempty"`
	Properties  map[string]any `json:"properties"`
	Metadata    map[string]any `json:"metadata"`
	ArchivedAt  *time.Time     `json:"archived_at,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type Grant struct {
	ID             uuid.UUID       `json:"id"`
	UserID         uuid.UUID       `json:"user_id"`
	BenefitID      uuid.UUID       `json:"benefit_id"`
	CustomerID     uuid.UUID       `json:"customer_id"`
	ProductID      *uuid.UUID      `json:"product_id,omitempty"`
	SubscriptionID *uuid.UUID      `json:"subscription_id,omitempty"`
	SourceType     GrantSourceType `json:"source_type"`
	SourceID       uuid.UUID       `json:"source_id"`
	Status         GrantStatus     `json:"status"`
	StartsAt       *time.Time      `json:"starts_at,omitempty"`
	EndsAt         *time.Time      `json:"ends_at,omitempty"`
	GrantedAt      time.Time       `json:"granted_at"`
	RevokedAt      *time.Time      `json:"revoked_at,omitempty"`
	Properties     map[string]any  `json:"properties"`
	Metadata       map[string]any  `json:"metadata"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type CreateRequest struct {
	Type        Type           `json:"type" binding:"required,oneof=custom feature meter_credit"`
	Name        string         `json:"name" binding:"required,min=3,max=160"`
	Code        string         `json:"code" binding:"required,min=2,max=120"`
	Description *string        `json:"description,omitempty" binding:"omitempty,max=1000"`
	Properties  map[string]any `json:"properties,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type UpdateRequest struct {
	Name        *string        `json:"name,omitempty" binding:"omitempty,min=3,max=160"`
	Code        *string        `json:"code,omitempty" binding:"omitempty,min=2,max=120"`
	Description *string        `json:"description,omitempty" binding:"omitempty,max=1000"`
	Properties  map[string]any `json:"properties,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	Archived    *bool          `json:"archived,omitempty"`
}

type ListParams struct {
	UserID          uuid.UUID
	Type            Type
	IncludeArchived bool
	Page            int
	Limit           int
}

type ListGrantsParams struct {
	UserID     uuid.UUID
	BenefitID  uuid.UUID
	CustomerID *uuid.UUID
	Status     GrantStatus
	Page       int
	Limit      int
}

type ListResponse struct {
	Items      []Benefit  `json:"items"`
	Pagination Pagination `json:"pagination"`
}

type ListGrantsResponse struct {
	Items      []Grant    `json:"items"`
	Pagination Pagination `json:"pagination"`
}

type Pagination struct {
	TotalCount int `json:"total_count"`
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	MaxPage    int `json:"max_page"`
}
