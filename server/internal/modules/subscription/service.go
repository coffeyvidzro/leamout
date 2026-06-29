package subscription

import (
	"context"
	"errors"
	"fmt"
	"time"

	subscriptionsm "github.com/cuffeyvidzro/leamout/internal/platform/statemachine/subscription"
	"github.com/google/uuid"
)

var ErrInvalidSubscription = errors.New("invalid subscription")

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, req CreateRequest) (*Subscription, error) {
	return s.repo.Create(ctx, userID, req)
}

func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]Subscription, error) {
	return s.repo.List(ctx, userID)
}

func (s *Service) Get(ctx context.Context, userID, id uuid.UUID) (*Subscription, error) {
	return s.repo.Get(ctx, userID, id)
}

func (s *Service) Update(ctx context.Context, userID, id uuid.UUID, req UpdateRequest) (*Subscription, error) {
	if req.Status != nil {
		subscription, err := s.repo.Get(ctx, userID, id)
		if err != nil {
			return nil, err
		}
		if err := subscriptionsm.ValidateTransition(subscriptionsm.Status(subscription.Status), subscriptionsm.Status(*req.Status)); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidSubscription, err)
		}
	}

	return s.repo.Update(ctx, userID, id, req)
}

func (s *Service) Delete(ctx context.Context, userID, id uuid.UUID) error {
	return s.repo.Delete(ctx, userID, id)
}

func (s *Service) ListDueForDunning(ctx context.Context, windowEnd time.Time) ([]DunningCandidate, error) {
	return s.repo.ListDueForDunning(ctx, windowEnd)
}
