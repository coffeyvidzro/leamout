package credits

import (
	"time"

	"github.com/google/uuid"
)

type LedgerType string

const (
	CurrencyGHS = "GHS"

	LedgerTypeTopUp  LedgerType = "topup"
	LedgerTypeDebit  LedgerType = "debit"
	LedgerTypeRefund LedgerType = "refund"
)

type Balance struct {
	UserID    uuid.UUID `json:"user_id"`
	Balance   int64     `json:"balance"`
	Currency  string    `json:"currency"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type LedgerEntry struct {
	ID           uuid.UUID      `json:"id"`
	UserID       uuid.UUID      `json:"user_id"`
	Type         LedgerType     `json:"type"`
	Amount       int64          `json:"amount"`
	BalanceAfter int64          `json:"balance_after"`
	Provider     *string        `json:"provider,omitempty"`
	Destination  *string        `json:"destination,omitempty"`
	Reference    *string        `json:"reference,omitempty"`
	Description  string         `json:"description"`
	Metadata     map[string]any `json:"metadata"`
	CreatedAt    time.Time      `json:"created_at"`
}

type TopUpParams struct {
	UserID      uuid.UUID
	Amount      int64
	Reference   string
	Description string
	Metadata    map[string]any
}

type ListLedgerParams struct {
	UserID uuid.UUID
	Limit  int
	Offset int
}

type DebitParams struct {
	UserID      uuid.UUID
	Amount      int64
	Provider    string
	Destination string
	Reference   string
	Description string
	Metadata    map[string]any
}

type RefundParams struct {
	UserID      uuid.UUID
	Amount      int64
	Provider    string
	Destination string
	Reference   string
	Description string
	Metadata    map[string]any
}
