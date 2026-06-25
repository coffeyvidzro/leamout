package credits

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var ErrInvalidAmount = errors.New("credit amount must be positive")

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) GetBalance(ctx context.Context, userID uuid.UUID) (*Balance, error) {
	return s.repository.GetBalance(ctx, userID)
}

func (s *Service) TopUp(ctx context.Context, params TopUpParams) (*Balance, error) {
	if params.Amount <= 0 {
		return nil, ErrInvalidAmount
	}

	return s.repository.TopUp(ctx, params)
}

func (s *Service) Debit(ctx context.Context, params DebitParams) (*Balance, error) {
	if params.Amount <= 0 {
		return nil, ErrInvalidAmount
	}

	return s.repository.Debit(ctx, params)
}

func (s *Service) Refund(ctx context.Context, params RefundParams) (*Balance, error) {
	if params.Amount <= 0 {
		return nil, ErrInvalidAmount
	}

	return s.repository.Refund(ctx, params)
}
