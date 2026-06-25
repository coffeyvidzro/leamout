package dunning

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

const defaultRenewalScanWindow = 72 * time.Hour

type ExpiringSubscription struct {
	UserID           uuid.UUID
	SubscriptionID   uuid.UUID
	CustomerID       *uuid.UUID
	CurrentPeriodEnd time.Time
}

type Scanner struct {
	repository *Repository
	river      *river.Client[pgx.Tx]
	log        *slog.Logger
	window     time.Duration
}

func NewScanner(repository *Repository, riverClient *river.Client[pgx.Tx], log *slog.Logger) *Scanner {
	return &Scanner{
		repository: repository,
		river:      riverClient,
		log:        log,
		window:     defaultRenewalScanWindow,
	}
}

func (s *Scanner) RunOnce(ctx context.Context, now time.Time) (int, error) {
	if s.river == nil {
		return 0, fmt.Errorf("river client is not configured")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	expiring, err := s.repository.ListSubscriptionsDueForRenewal(ctx, now, now.Add(s.window))
	if err != nil {
		return 0, err
	}

	inserted := 0
	for _, sub := range expiring {
		attempt, err := s.repository.CreateOrReuseAttempt(ctx, CreateAttemptParams{
			UserID:         sub.UserID,
			SubscriptionID: sub.SubscriptionID,
			CustomerID:     sub.CustomerID,
			Reason:         AttemptReasonRenewalDue,
			PeriodEnd:      sub.CurrentPeriodEnd,
			ExpiresAt:      sub.CurrentPeriodEnd.Add(defaultAttemptTTL),
			Metadata: map[string]any{
				"scanner_window_hours": int(s.window.Hours()),
			},
		})
		if err != nil {
			return inserted, fmt.Errorf("create dunning attempt for subscription %s: %w", sub.SubscriptionID, err)
		}

		_, err = s.river.Insert(ctx, SendRenewalSMSArgs{
			AttemptID: attempt.ID,
			UserID:    attempt.UserID,
		}, nil)
		if err != nil {
			return inserted, fmt.Errorf("insert renewal sms job for attempt %s: %w", attempt.ID, err)
		}
		inserted++
	}

	if s.log != nil {
		s.log.Info("renewal scanner completed", slog.Int("jobs_inserted", inserted), slog.Int("subscriptions_due", len(expiring)))
	}

	return inserted, nil
}
