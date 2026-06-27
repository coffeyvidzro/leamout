package wallet

import (
	"time"

	"github.com/google/uuid"
)

type Direction string
type BalanceType string
type Reason string

const (
	DirectionCredit Direction = "credit"
	DirectionDebit  Direction = "debit"

	BalanceTypePending   BalanceType = "pending"
	BalanceTypeAvailable BalanceType = "available"

	ReasonPaymentCaptured Reason = "payment_captured"
	ReasonPaymentSettled  Reason = "payment_settled"
	ReasonRefund          Reason = "refund"
	ReasonWithdrawal      Reason = "withdrawal"
	ReasonAdjustment      Reason = "adjustment"
)

type Wallet struct {
	ID               uuid.UUID `json:"id"`
	UserID           uuid.UUID `json:"user_id"`
	Country          string    `json:"country"`
	Currency         string    `json:"currency"`
	PendingBalance   int64     `json:"pending_balance"`
	AvailableBalance int64     `json:"available_balance"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type LedgerEntry struct {
	ID            uuid.UUID      `json:"id"`
	WalletID      uuid.UUID      `json:"wallet_id"`
	UserID        uuid.UUID      `json:"user_id"`
	PaymentID     *uuid.UUID     `json:"payment_id,omitempty"`
	TransactionID *uuid.UUID     `json:"transaction_id,omitempty"`
	Direction     Direction      `json:"direction"`
	BalanceType   BalanceType    `json:"balance_type"`
	Reason        Reason         `json:"reason"`
	Country       string         `json:"country"`
	Currency      string         `json:"currency"`
	Amount        int64          `json:"amount"`
	BalanceAfter  int64          `json:"balance_after"`
	Metadata      map[string]any `json:"metadata"`
	CreatedAt     time.Time      `json:"created_at"`
}

type CreditPaymentCaptureParams struct {
	UserID        uuid.UUID
	PaymentID     uuid.UUID
	TransactionID uuid.UUID
	Country       string
	Currency      string
	Amount        int64
	Metadata      map[string]any
}

type ListWalletsParams struct {
	UserID   uuid.UUID
	Country  string
	Currency string
}

type ListLedgerParams struct {
	UserID   uuid.UUID
	Country  string
	Currency string
	Limit    int
	Offset   int
}
