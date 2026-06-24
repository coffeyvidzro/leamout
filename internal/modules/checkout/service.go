package checkout

import (
	"context"
	"errors"
	"time"

	"github.com/cuffeyvidzro/leamout/internal/modules/dunning"
)

const defaultSessionTTL = 30 * time.Minute

var ErrInvalidToken = errors.New("invalid checkout token")

type Service struct {
	repository     *Repository
	dunningService *dunning.Service
}

func NewService(repository *Repository, dunningService *dunning.Service) *Service {
	return &Service{repository: repository, dunningService: dunningService}
}

func (s *Service) StartFromToken(ctx context.Context, rawToken string) (*Session, error) {
	tokenWithAttempt, err := s.validToken(ctx, rawToken)
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().UTC().Add(defaultSessionTTL)
	if tokenWithAttempt.Token.ExpiresAt.Before(expiresAt) {
		expiresAt = tokenWithAttempt.Token.ExpiresAt
	}

	return s.repository.CreateOrReuseFromDunning(ctx, CreateSessionParams{
		UserID:           tokenWithAttempt.Attempt.UserID,
		CustomerID:       tokenWithAttempt.Attempt.CustomerID,
		SubscriptionID:   tokenWithAttempt.Attempt.SubscriptionID,
		DunningAttemptID: tokenWithAttempt.Attempt.ID,
		DunningTokenID:   tokenWithAttempt.Token.ID,
		ExpiresAt:        expiresAt,
	})
}

func (s *Service) CompleteMockPayment(ctx context.Context, rawToken string) (*Session, error) {
	session, err := s.StartFromToken(ctx, rawToken)
	if err != nil {
		return nil, err
	}

	if _, err := s.dunningService.ConsumeToken(ctx, rawToken); err != nil {
		return nil, err
	}

	completed, err := s.repository.Complete(ctx, session.ID)
	if err != nil {
		return nil, err
	}

	if err := s.dunningService.MarkAttemptPaid(ctx, completed.DunningAttemptID); err != nil {
		return nil, err
	}

	return completed, nil
}

func (s *Service) validToken(ctx context.Context, rawToken string) (*dunning.TokenWithAttempt, error) {
	if rawToken == "" {
		return nil, ErrInvalidToken
	}

	tokenWithAttempt, err := s.dunningService.GetByToken(ctx, rawToken)
	if err != nil {
		return nil, err
	}
	if tokenWithAttempt.Token.UsedAt != nil || !tokenWithAttempt.Token.ExpiresAt.After(time.Now().UTC()) {
		return nil, ErrInvalidToken
	}
	if tokenWithAttempt.Attempt.Status == dunning.AttemptStatusPaid ||
		tokenWithAttempt.Attempt.Status == dunning.AttemptStatusCanceled ||
		tokenWithAttempt.Attempt.Status == dunning.AttemptStatusExpired ||
		!tokenWithAttempt.Attempt.ExpiresAt.After(time.Now().UTC()) {
		return nil, ErrInvalidToken
	}

	return tokenWithAttempt, nil
}
