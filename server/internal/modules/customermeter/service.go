package customermeter

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrInvalidCustomerMeter = errors.New("invalid customer meter")

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Get(ctx context.Context, userID, id uuid.UUID) (*CustomerMeter, error) {
	if userID == uuid.Nil || id == uuid.Nil {
		return nil, ErrInvalidCustomerMeter
	}

	return s.repo.Get(ctx, userID, id)
}

func (s *Service) List(ctx context.Context, params ListParams) (*ListResponse, error) {
	if params.UserID == uuid.Nil {
		return nil, ErrInvalidCustomerMeter
	}
	params.ExternalCustomerID = strings.TrimSpace(params.ExternalCustomerID)

	return s.repo.List(ctx, params)
}

func (s *Service) RefreshCreditsForSubscription(ctx context.Context, tx pgx.Tx, userID, subscriptionID uuid.UUID, fallbackCustomerID *uuid.UUID) error {
	if userID == uuid.Nil || subscriptionID == uuid.Nil {
		return ErrInvalidCustomerMeter
	}

	return s.repo.RefreshCreditsForSubscription(ctx, tx, userID, subscriptionID, fallbackCustomerID)
}
