package checkout

import (
	"context"

	"github.com/google/uuid"
)

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, req CreateRequest) (*Session, error) {
	return s.repository.CreateOrReuseFromDunning(ctx, CreateSessionParams{
		UserID:           userID,
		CustomerID:       req.CustomerID,
		SubscriptionID:   req.SubscriptionID,
		DunningAttemptID: req.DunningAttemptID,
		DunningTokenID:   req.DunningTokenID,
		ExpiresAt:        req.ExpiresAt,
		Metadata:         req.Metadata,
	})
}

func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]Session, error) {
	return s.repository.List(ctx, userID)
}

func (s *Service) Get(ctx context.Context, userID, id uuid.UUID) (*Session, error) {
	return s.repository.Get(ctx, userID, id)
}

func (s *Service) Update(ctx context.Context, userID, id uuid.UUID, req UpdateRequest) (*Session, error) {
	return s.repository.Update(ctx, userID, id, req)
}
