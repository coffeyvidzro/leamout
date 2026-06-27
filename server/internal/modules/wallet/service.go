package wallet

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
)

var ErrInvalidWalletRequest = errors.New("invalid wallet request")

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Get(ctx context.Context, userID uuid.UUID, currency string) (*Wallet, error) {
	if userID == uuid.Nil || strings.TrimSpace(currency) == "" {
		return nil, ErrInvalidWalletRequest
	}
	return s.repository.Get(ctx, userID, currency)
}

func (s *Service) ListLedger(ctx context.Context, params ListLedgerParams) ([]LedgerEntry, error) {
	if params.UserID == uuid.Nil {
		return nil, ErrInvalidWalletRequest
	}
	return s.repository.ListLedger(ctx, params)
}

func (s *Service) CreditPaymentCapture(ctx context.Context, params CreditPaymentCaptureParams) error {
	if params.UserID == uuid.Nil || params.PaymentID == uuid.Nil || params.TransactionID == uuid.Nil {
		return ErrInvalidWalletRequest
	}
	if params.Amount <= 0 || strings.TrimSpace(params.Currency) == "" {
		return ErrInvalidWalletRequest
	}
	return s.repository.CreditPaymentCapture(ctx, params)
}
