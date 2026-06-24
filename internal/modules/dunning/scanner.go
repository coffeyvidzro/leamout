package dunning

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/cuffeyvidzro/leamout/internal/modules/subscription"
)

const DefaultScanWindow = 72 * time.Hour

type Scanner struct {
	subscriptions *subscription.Service
	enqueue       EnqueueReminderFunc
	window        time.Duration
	log           *slog.Logger
}

type EnqueueReminderFunc func(context.Context, SendReminderArgs) error

type ScannerResult struct {
	WindowEnd time.Time
	Scanned   int
	Enqueued  int
	Skipped   int
}

func NewScanner(subscriptions *subscription.Service, enqueue EnqueueReminderFunc, log *slog.Logger) *Scanner {
	return &Scanner{
		subscriptions: subscriptions,
		enqueue:       enqueue,
		window:        DefaultScanWindow,
		log:           log,
	}
}

func (s *Scanner) RunOnce(ctx context.Context) (*ScannerResult, error) {
	if s.subscriptions == nil {
		return nil, fmt.Errorf("subscription service is not configured")
	}
	if s.enqueue == nil {
		return nil, fmt.Errorf("dunning reminder enqueue function is not configured")
	}

	windowEnd := time.Now().UTC().Add(s.window)
	candidates, err := s.subscriptions.ListDueForDunning(ctx, windowEnd)
	if err != nil {
		return nil, fmt.Errorf("list subscriptions due for dunning: %w", err)
	}

	result := &ScannerResult{
		WindowEnd: windowEnd,
		Scanned:   len(candidates),
	}
	for _, candidate := range candidates {
		if candidate.CustomerID == nil {
			result.Skipped++
			continue
		}

		err := s.enqueue(ctx, SendReminderArgs{
			UserID:           candidate.UserID,
			SubscriptionID:   candidate.ID,
			CustomerID:       *candidate.CustomerID,
			CurrentPeriodEnd: candidate.CurrentPeriodEnd,
		})
		if err != nil {
			return nil, fmt.Errorf("enqueue dunning reminder for subscription %s: %w", candidate.ID, err)
		}

		result.Enqueued++
	}

	if s.log != nil {
		s.log.Info(
			"dunning scanner completed",
			slog.String("window_end", result.WindowEnd.Format(time.RFC3339)),
			slog.Int("scanned", result.Scanned),
			slog.Int("enqueued", result.Enqueued),
			slog.Int("skipped", result.Skipped),
		)
	}

	return result, nil
}

func (s *Scanner) SetWindow(window time.Duration) {
	if window <= 0 {
		return
	}

	s.window = window
}
