package price

import (
	"time"

	"github.com/google/uuid"
)

const (
	TypeOneTime   = "one_time"
	TypeRecurring = "recurring"
	TypeUsage     = "usage"

	IntervalDay   = "day"
	IntervalWeek  = "week"
	IntervalMonth = "month"
	IntervalYear  = "year"
)

type Price struct {
	ID         uuid.UUID      `json:"id"`
	UserID     uuid.UUID      `json:"user_id"`
	ProductID  uuid.UUID      `json:"product_id"`
	Nickname   string         `json:"nickname"`
	Type       string         `json:"type"`
	LookupKey  *string        `json:"lookup_key,omitempty"`
	UnitAmount int64          `json:"unit_amount"`
	Currency   string         `json:"currency"`
	Interval   *string        `json:"interval,omitempty"`
	Metadata   map[string]any `json:"metadata"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

type CreateRequest struct {
	Nickname   string         `json:"nickname" binding:"required,min=1,max=160"`
	Type       string         `json:"type" binding:"required,oneof=one_time recurring usage"`
	LookupKey  *string        `json:"lookup_key" binding:"omitempty,max=160"`
	UnitAmount int64          `json:"unit_amount" binding:"required,gt=0"`
	Currency   string         `json:"currency" binding:"required,len=3,uppercase"`
	Interval   *string        `json:"interval" binding:"omitempty,oneof=day week month year"`
	Metadata   map[string]any `json:"metadata"`
}

type UpdateRequest struct {
	Nickname   *string        `json:"nickname,omitempty" binding:"omitempty,min=1,max=160"`
	LookupKey  *string        `json:"lookup_key,omitempty" binding:"omitempty,max=160"`
	UnitAmount *int64         `json:"unit_amount,omitempty" binding:"omitempty,gt=0"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}
