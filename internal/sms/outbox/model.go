package outbox

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusPending  Status = "pending"
	StatusDebited  Status = "debited"
	StatusSent     Status = "sent"
	StatusFailed   Status = "failed"
	StatusRefunded Status = "refunded"
)

type Message struct {
	ID          uuid.UUID      `json:"id"`
	UserID      uuid.UUID      `json:"user_id"`
	Reference   string         `json:"reference"`
	Destination string         `json:"destination"`
	Sender      string         `json:"sender"`
	Content     string         `json:"content"`
	CountryCode string         `json:"country_code"`
	Provider    string         `json:"provider"`
	Cost        int64          `json:"cost"`
	Status      Status         `json:"status"`
	Error       *string        `json:"error,omitempty"`
	Metadata    map[string]any `json:"metadata"`
	DebitedAt   *time.Time     `json:"debited_at,omitempty"`
	SentAt      *time.Time     `json:"sent_at,omitempty"`
	RefundedAt  *time.Time     `json:"refunded_at,omitempty"`
	FailedAt    *time.Time     `json:"failed_at,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type CreateParams struct {
	UserID      uuid.UUID
	Reference   string
	Destination string
	Sender      string
	Content     string
	CountryCode string
	Provider    string
	Cost        int64
	Metadata    map[string]any
}
