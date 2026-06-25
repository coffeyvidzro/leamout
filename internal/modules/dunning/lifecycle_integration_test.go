package dunning_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/cuffeyvidzro/leamout/internal/modules/checkout"
	"github.com/cuffeyvidzro/leamout/internal/modules/dunning"
	"github.com/cuffeyvidzro/leamout/internal/modules/subscription"
	"github.com/cuffeyvidzro/leamout/internal/sms"
	"github.com/cuffeyvidzro/leamout/internal/testutil/dbtest"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

type fixture struct {
	UserID         uuid.UUID
	ProductID      uuid.UUID
	PriceID        uuid.UUID
	CustomerID     uuid.UUID
	SubscriptionID uuid.UUID
	PeriodEnd      time.Time
}

type fakeSender struct {
	messages []sms.Message
}

func (s *fakeSender) Send(_ context.Context, msg sms.Message) error {
	s.messages = append(s.messages, msg)
	return nil
}

func TestScannerEnqueuesOnlyDueSubscriptionsWithoutActiveAttempts(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewPostgresPool(t)
	due := seedFixture(t, db, "scanner-due", 71*time.Hour)
	withAttempt := seedFixture(t, db, "scanner-existing-attempt", 71*time.Hour)
	_ = seedFixture(t, db, "scanner-not-due", 120*time.Hour)

	dunningService := dunning.NewService(dunning.NewRepository(db), nil)
	_, err := dunningService.CreateOrReuseAttempt(ctx, dunning.CreateAttemptParams{
		UserID:         withAttempt.UserID,
		SubscriptionID: withAttempt.SubscriptionID,
		CustomerID:     &withAttempt.CustomerID,
		Reason:         dunning.AttemptReasonRenewalDue,
		PeriodEnd:      withAttempt.PeriodEnd,
	})
	if err != nil {
		t.Fatalf("create existing attempt: %v", err)
	}

	var enqueued []dunning.SendReminderArgs
	scanner := dunning.NewScanner(
		subscription.NewService(subscription.NewRepository(db)),
		func(_ context.Context, args dunning.SendReminderArgs) error {
			enqueued = append(enqueued, args)
			return nil
		},
		nil,
	)

	result, err := scanner.RunOnce(ctx)
	if err != nil {
		t.Fatalf("run scanner: %v", err)
	}

	if result.Scanned != 1 || result.Enqueued != 1 || len(enqueued) != 1 {
		t.Fatalf("expected one enqueued candidate, got scanned=%d enqueued=%d len=%d", result.Scanned, result.Enqueued, len(enqueued))
	}
	if enqueued[0].SubscriptionID != due.SubscriptionID {
		t.Fatalf("expected due subscription %s, got %s", due.SubscriptionID, enqueued[0].SubscriptionID)
	}
}

func TestSendReminderWorkerCreatesAttemptTokenAndSendsSMS(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewPostgresPool(t)
	fx := seedFixture(t, db, "worker", 71*time.Hour)

	sender := &fakeSender{}
	worker := dunning.NewSendReminderWorker(dunning.NewService(dunning.NewRepository(db), nil), sender, "http://api.local/v1", nil)
	err := worker.Work(ctx, &river.Job[dunning.SendReminderArgs]{
		Args: dunning.SendReminderArgs{
			UserID:           fx.UserID,
			SubscriptionID:   fx.SubscriptionID,
			CustomerID:       fx.CustomerID,
			CurrentPeriodEnd: fx.PeriodEnd,
		},
	})
	if err != nil {
		t.Fatalf("work reminder: %v", err)
	}

	if len(sender.messages) != 1 {
		t.Fatalf("expected one SMS, got %d", len(sender.messages))
	}
	if sender.messages[0].To != "+233501234567" {
		t.Fatalf("unexpected SMS recipient: %s", sender.messages[0].To)
	}
	if !strings.Contains(sender.messages[0].Content, "http://api.local/v1/dunning/") {
		t.Fatalf("SMS did not contain dunning link: %s", sender.messages[0].Content)
	}

	var attemptStatus string
	var sent bool
	var tokenCount int
	if err := db.QueryRow(ctx, `
SELECT a.status, a.sent_at IS NOT NULL, COUNT(t.id)
FROM dunning_attempts a
LEFT JOIN dunning_tokens t ON t.user_id = a.user_id AND t.dunning_attempt_id = a.id
WHERE a.user_id = $1 AND a.subscription_id = $2
GROUP BY a.id`, fx.UserID, fx.SubscriptionID).Scan(&attemptStatus, &sent, &tokenCount); err != nil {
		t.Fatalf("query reminder state: %v", err)
	}
	if attemptStatus != string(dunning.AttemptStatusSent) || !sent || tokenCount != 1 {
		t.Fatalf("unexpected reminder state status=%s sent=%t token_count=%d", attemptStatus, sent, tokenCount)
	}
}

func TestDunningCheckoutConfirmationIsIdempotent(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewPostgresPool(t)
	fx := seedFixture(t, db, "checkout-idempotency", 71*time.Hour)

	checkoutService := checkout.NewService(checkout.NewRepository(db))
	dunningService := dunning.NewService(dunning.NewRepository(db), checkoutService)

	attempt, err := dunningService.CreateOrReuseAttempt(ctx, dunning.CreateAttemptParams{
		UserID:         fx.UserID,
		SubscriptionID: fx.SubscriptionID,
		CustomerID:     &fx.CustomerID,
		Reason:         dunning.AttemptReasonRenewalDue,
		PeriodEnd:      fx.PeriodEnd,
	})
	if err != nil {
		t.Fatalf("create attempt: %v", err)
	}
	rawToken, _, err := dunningService.CreateToken(ctx, attempt)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	session, err := dunningService.OpenRecoveryLink(ctx, rawToken)
	if err != nil {
		t.Fatalf("open recovery link: %v", err)
	}
	if _, err := checkoutService.Confirm(ctx, session.ClientSecret); err != nil {
		t.Fatalf("first confirm: %v", err)
	}
	afterFirst := subscriptionPeriodEnd(t, db, fx.SubscriptionID)

	if _, err := checkoutService.Confirm(ctx, session.ClientSecret); err != nil {
		t.Fatalf("second confirm: %v", err)
	}
	afterSecond := subscriptionPeriodEnd(t, db, fx.SubscriptionID)

	if !afterFirst.Equal(afterSecond) {
		t.Fatalf("second confirmation changed period end: first=%s second=%s", afterFirst, afterSecond)
	}
	if !afterFirst.After(fx.PeriodEnd) {
		t.Fatalf("confirmation did not extend subscription: before=%s after=%s", fx.PeriodEnd, afterFirst)
	}

	var attemptStatus string
	var paid, revoked bool
	if err := db.QueryRow(ctx, `
SELECT a.status, a.paid_at IS NOT NULL, t.revoked_at IS NOT NULL
FROM dunning_attempts a
JOIN dunning_tokens t ON t.user_id = a.user_id AND t.dunning_attempt_id = a.id
WHERE a.id = $1`, attempt.ID).Scan(&attemptStatus, &paid, &revoked); err != nil {
		t.Fatalf("query completion state: %v", err)
	}
	if attemptStatus != string(dunning.AttemptStatusPaid) || !paid || !revoked {
		t.Fatalf("unexpected completion state status=%s paid=%t revoked=%t", attemptStatus, paid, revoked)
	}
}

func seedFixture(t *testing.T, db *pgxpool.Pool, suffix string, periodOffset time.Duration) fixture {
	t.Helper()

	ctx := context.Background()
	email := "merchant-" + suffix + "@example.test"
	var fx fixture
	if err := db.QueryRow(ctx, `
INSERT INTO users (name, email, email_verified, status)
VALUES ($1, $2, TRUE, 'active')
RETURNING id`, "Merchant "+suffix, email).Scan(&fx.UserID); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if err := db.QueryRow(ctx, `
INSERT INTO products (user_id, name, active, metadata)
VALUES ($1, $2, TRUE, '{}'::jsonb)
RETURNING id`, fx.UserID, "Product "+suffix).Scan(&fx.ProductID); err != nil {
		t.Fatalf("insert product: %v", err)
	}
	if err := db.QueryRow(ctx, `
INSERT INTO prices (user_id, product_id, nickname, type, unit_amount, currency, interval, metadata)
VALUES ($1, $2, $3, 'recurring', 5000, 'GHS', 'month', '{}'::jsonb)
RETURNING id`, fx.UserID, fx.ProductID, "Monthly "+suffix).Scan(&fx.PriceID); err != nil {
		t.Fatalf("insert price: %v", err)
	}
	if err := db.QueryRow(ctx, `
INSERT INTO customers (user_id, name, email, phone, address, metadata)
VALUES ($1, $2, $3, '+233501234567', '{}'::jsonb, '{}'::jsonb)
RETURNING id`, fx.UserID, "Customer "+suffix, "customer-"+suffix+"@example.test").Scan(&fx.CustomerID); err != nil {
		t.Fatalf("insert customer: %v", err)
	}

	periodStart := time.Now().UTC()
	fx.PeriodEnd = periodStart.Add(periodOffset)
	if err := db.QueryRow(ctx, `
INSERT INTO subscriptions (user_id, customer_id, price_id, status, current_period_start, current_period_end, metadata)
VALUES ($1, $2, $3, 'active', $4, $5, '{}'::jsonb)
RETURNING id`, fx.UserID, fx.CustomerID, fx.PriceID, periodStart, fx.PeriodEnd).Scan(&fx.SubscriptionID); err != nil {
		t.Fatalf("insert subscription: %v", err)
	}

	return fx
}

func subscriptionPeriodEnd(t *testing.T, db *pgxpool.Pool, subscriptionID uuid.UUID) time.Time {
	t.Helper()

	var periodEnd time.Time
	if err := db.QueryRow(context.Background(), `SELECT current_period_end FROM subscriptions WHERE id = $1`, subscriptionID).Scan(&periodEnd); err != nil {
		t.Fatalf("query subscription period end: %v", err)
	}
	return periodEnd
}
