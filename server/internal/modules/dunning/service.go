package dunning

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/cuffeyvidzro/leamout/internal/modules/checkout"
	"github.com/google/uuid"
)

const (
	defaultAttemptTTL = 7 * 24 * time.Hour
	defaultTokenTTL   = 72 * time.Hour
	tokenBytes        = 32
)

var ErrInvalidRecoveryLink = errors.New("invalid or expired recovery link")

type Service struct {
	repository      *Repository
	checkoutService *checkout.Service
}

func NewService(repository *Repository, checkoutService *checkout.Service) *Service {
	return &Service{repository: repository, checkoutService: checkoutService}
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

func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]Attempt, error) {
	return s.repository.List(ctx, userID)
}

func (s *Service) Get(ctx context.Context, userID, attemptID uuid.UUID) (*Attempt, error) {
	return s.repository.Get(ctx, userID, attemptID)
}

func (s *Service) GetByToken(ctx context.Context, rawToken string) (*TokenWithAttempt, error) {
	return s.repository.GetByTokenHash(ctx, HashToken(rawToken))
}

func (s *Service) GetReminderDetails(ctx context.Context, userID, subscriptionID uuid.UUID) (*ReminderDetails, error) {
	return s.repository.GetReminderDetails(ctx, userID, subscriptionID)
}

func (s *Service) OpenRecoveryLink(ctx context.Context, rawToken string) (*checkout.Session, error) {
	if s.checkoutService == nil {
		return nil, errors.New("checkout service is not configured")
	}

	result, err := s.RecordTokenUse(ctx, rawToken)
	if err != nil {
		return nil, err
	}
	if !canOpenRecoveryAttempt(result.Attempt) {
		return nil, ErrInvalidRecoveryLink
	}

	details, err := s.repository.GetCheckoutDetails(ctx, result.Attempt.UserID, result.Attempt.ID)
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().UTC().Add(30 * time.Minute)
	if result.Token.ExpiresAt.Before(expiresAt) {
		expiresAt = result.Token.ExpiresAt
	}

	label := "Renew subscription"
	subscriptionID := result.Attempt.SubscriptionID
	metadata := map[string]any{
		"source":             "sms_dunning",
		"dunning_attempt_id": result.Attempt.ID.String(),
		"dunning_token_id":   result.Token.ID.String(),
		"subscription_id":    result.Attempt.SubscriptionID.String(),
	}

	return s.checkoutService.Create(ctx, result.Attempt.UserID, checkout.CreateRequest{
		CustomerID:     result.Attempt.CustomerID,
		SubscriptionID: &subscriptionID,
		Mode:           checkout.ModeRenewal,
		Source:         checkout.SourceDunning,
		Label:          &label,
		Amount:         details.Amount,
		Currency:       details.Currency,
		ExpiresAt:      expiresAt,
		Metadata:       metadata,
	})
}

func (s *Service) RecordTokenUse(ctx context.Context, rawToken string) (*TokenWithAttempt, error) {
	result, err := s.repository.RecordTokenUse(ctx, HashToken(rawToken))
	if err != nil {
		return nil, err
	}
	if err := s.repository.MarkAttemptClicked(ctx, result.Attempt.ID); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *Service) RevokeToken(ctx context.Context, rawToken string) error {
	return s.repository.RevokeToken(ctx, HashToken(rawToken))
}

func (s *Service) RevokeAttemptTokens(ctx context.Context, userID, attemptID uuid.UUID) error {
	return s.repository.RevokeAttemptTokens(ctx, userID, attemptID)
}

func (s *Service) MarkAttemptSent(ctx context.Context, attemptID uuid.UUID) error {
	return s.repository.MarkAttemptSent(ctx, attemptID)
}

func (s *Service) MarkAttemptClicked(ctx context.Context, attemptID uuid.UUID) error {
	return s.repository.MarkAttemptClicked(ctx, attemptID)
}

func (s *Service) MarkAttemptPaid(ctx context.Context, attemptID uuid.UUID) error {
	return s.repository.MarkAttemptPaid(ctx, attemptID)
}

func HashToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}

func canOpenRecoveryAttempt(attempt Attempt) bool {
	if attempt.Status != AttemptStatusPending && attempt.Status != AttemptStatusSent {
		return false
	}

	return attempt.ExpiresAt.After(time.Now().UTC())
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
