package usagecredit

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrInvalidUsageCredit = errors.New("invalid usage credit request")

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ApplySubscriptionCredits(ctx context.Context, tx pgx.Tx, userID, subscriptionID, checkoutID uuid.UUID, fallbackCustomerID *uuid.UUID) error {
	if userID == uuid.Nil || subscriptionID == uuid.Nil || checkoutID == uuid.Nil {
		return ErrInvalidUsageCredit
	}
	return s.repo.ApplySubscriptionCredits(ctx, tx, userID, subscriptionID, checkoutID, fallbackCustomerID)
}

func (s *Service) ListGrants(ctx context.Context, params ListGrantsParams) (*ListGrantsResponse, error) {
	if params.UserID == uuid.Nil {
		return nil, ErrInvalidUsageCredit
	}
	return s.repo.ListGrants(ctx, params)
}

func (s *Service) ListLedger(ctx context.Context, params ListLedgerParams) (*ListLedgerResponse, error) {
	if params.UserID == uuid.Nil {
		return nil, ErrInvalidUsageCredit
	}
	return s.repo.ListLedger(ctx, params)
}
