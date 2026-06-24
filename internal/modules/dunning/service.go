package dunning

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	defaultAttemptTTL = 7 * 24 * time.Hour
	defaultTokenTTL   = 72 * time.Hour
	tokenBytes        = 32
)

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) CreateOrReuseAttempt(ctx context.Context, params CreateAttemptParams) (*Attempt, error) {
	if params.Reason == "" {
		params.Reason = AttemptReasonRenewalDue
	}
	if params.ExpiresAt.IsZero() {
		params.ExpiresAt = time.Now().UTC().Add(defaultAttemptTTL)
	}

	attempt, err := s.repository.CreateOrReuseAttempt(ctx, params)
	if err != nil {
		return nil, err
	}

	return attempt, nil
}

func (s *Service) CreateToken(ctx context.Context, attempt *Attempt) (string, *Token, error) {
	rawToken, err := newToken(tokenBytes)
	if err != nil {
		return "", nil, fmt.Errorf("create dunning token: %w", err)
	}

	token, err := s.repository.CreateToken(ctx, CreateTokenParams{
		UserID:           attempt.UserID,
		DunningAttemptID: attempt.ID,
		TokenHash:        HashToken(rawToken),
		ExpiresAt:        tokenExpiry(attempt.ExpiresAt),
	})
	if err != nil {
		return "", nil, err
	}

	return rawToken, token, nil
}

func (s *Service) GetByToken(ctx context.Context, rawToken string) (*TokenWithAttempt, error) {
	return s.repository.GetByTokenHash(ctx, HashToken(rawToken))
}

func (s *Service) ConsumeToken(ctx context.Context, rawToken string) (*TokenWithAttempt, error) {
	return s.repository.ConsumeToken(ctx, HashToken(rawToken))
}

func (s *Service) MarkAttemptSent(ctx context.Context, attemptID uuid.UUID) error {
	return s.repository.MarkAttemptSent(ctx, attemptID)
}

func (s *Service) MarkAttemptPaid(ctx context.Context, attemptID uuid.UUID) error {
	return s.repository.MarkAttemptPaid(ctx, attemptID)
}

func HashToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}

func newToken(size int) (string, error) {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func tokenExpiry(attemptExpiry time.Time) time.Time {
	expiresAt := time.Now().UTC().Add(defaultTokenTTL)
	if !attemptExpiry.IsZero() && attemptExpiry.Before(expiresAt) {
		return attemptExpiry
	}

	return expiresAt
}
