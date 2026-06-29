package dunning_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	dunning "github.com/cuffeyvidzro/leamout/internal/modules/dunning"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestGetConversionMetrics(t *testing.T) {
	ctx := context.Background()
	pool := openRenewalDunningTestDB(t)
	userID := createRenewalDunningTestUser(t, pool)
	fixture := createDunningMetricsFixture(t, pool, userID)
	repository := dunning.NewRepository(pool)

	insertDunningMetricsAttempt(t, pool, fixture, dunning.AttemptStatusSent, false, false)
	insertDunningMetricsAttempt(t, pool, fixture, dunning.AttemptStatusSent, true, false)
	checkoutStartedAttemptID := insertDunningMetricsAttempt(t, pool, fixture, dunning.AttemptStatusSent, true, false)
	insertDunningMetricsCheckout(t, pool, fixture, checkoutStartedAttemptID)
	insertDunningMetricsAttempt(t, pool, fixture, dunning.AttemptStatusPaid, false, false)
	insertDunningMetricsAttempt(t, pool, fixture, dunning.AttemptStatusCanceled, false, false)
	insertDunningMetricsAttempt(t, pool, fixture, dunning.AttemptStatusExpired, false, true)

	metrics, err := repository.GetConversionMetrics(ctx, userID)
	if err != nil {
		t.Fatalf("get dunning conversion metrics: %v", err)
	}

	assertConversionMetrics(t, metrics, dunning.ConversionMetrics{
		Sent:            4,
		Clicked:         2,
		CheckoutStarted: 1,
		Paid:            1,
		Failed:          1,
		Expired:         1,
	})
}

type dunningMetricsFixture struct {
	UserID         uuid.UUID
	CustomerID     uuid.UUID
	ProductID      uuid.UUID
	PriceID        uuid.UUID
	SubscriptionID uuid.UUID
}

func createDunningMetricsFixture(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID) dunningMetricsFixture {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fixture := dunningMetricsFixture{
		UserID:         userID,
		CustomerID:     uuid.New(),
		ProductID:      uuid.New(),
		PriceID:        uuid.New(),
		SubscriptionID: uuid.New(),
	}

	_, err := pool.Exec(ctx, `
INSERT INTO products (id, user_id, name, active, metadata)
VALUES ($1, $2, 'Dunning Metrics Plan', TRUE, '{}'::jsonb)`, fixture.ProductID, fixture.UserID)
	if err != nil {
		t.Fatalf("insert metrics product: %v", err)
	}

	_, err = pool.Exec(ctx, `
INSERT INTO prices (id, user_id, product_id, nickname, type, unit_amount, currency, interval, metadata)
VALUES ($1, $2, $3, 'Monthly', 'recurring', 5000, 'GHS', 'month', '{}'::jsonb)`, fixture.PriceID, fixture.UserID, fixture.ProductID)
	if err != nil {
		t.Fatalf("insert metrics price: %v", err)
	}

	_, err = pool.Exec(ctx, `
INSERT INTO customers (id, user_id, name, phone, external_id, address, metadata)
VALUES ($1, $2, 'Dunning Metrics Customer', '+233241234567', $3, '{}'::jsonb, '{}'::jsonb)`,
		fixture.CustomerID,
		fixture.UserID,
		fmt.Sprintf("metrics_customer_%s", fixture.CustomerID),
	)
	if err != nil {
		t.Fatalf("insert metrics customer: %v", err)
	}

	_, err = pool.Exec(ctx, `
INSERT INTO subscriptions (id, user_id, customer_id, price_id, status, current_period_start, current_period_end, metadata)
VALUES ($1, $2, $3, $4, 'active', NOW() - INTERVAL '28 days', NOW() + INTERVAL '2 days', '{}'::jsonb)`,
		fixture.SubscriptionID,
		fixture.UserID,
		fixture.CustomerID,
		fixture.PriceID,
	)
	if err != nil {
		t.Fatalf("insert metrics subscription: %v", err)
	}

	return fixture
}

func insertDunningMetricsAttempt(t *testing.T, pool *pgxpool.Pool, fixture dunningMetricsFixture, status dunning.AttemptStatus, clicked bool, expiredTime bool) uuid.UUID {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	attemptID := uuid.New()
	now := time.Now().UTC()
	createdAt := now
	expiresAt := now.Add(24 * time.Hour)
	periodEnd := now.Add(48*time.Hour + time.Duration(attemptID[0])*time.Second)
	if expiredTime {
		createdAt = now.Add(-2 * time.Hour)
		expiresAt = now.Add(-1 * time.Hour)
	}

	var sentAt any
	var clickedAt any
	var paidAt any
	var canceledAt any
	if status == dunning.AttemptStatusSent || status == dunning.AttemptStatusPaid {
		sentAt = now
	}
	if clicked {
		clickedAt = now
	}
	if status == dunning.AttemptStatusPaid {
		paidAt = now
	}
	if status == dunning.AttemptStatusCanceled {
		canceledAt = now
	}

	_, err := pool.Exec(ctx, `
INSERT INTO dunning_attempts (
	id,
	user_id,
	subscription_id,
	customer_id,
	status,
	reason,
	period_end,
	expires_at,
	sent_at,
	clicked_at,
	paid_at,
	canceled_at,
	metadata,
	created_at,
	updated_at
)
VALUES ($1, $2, $3, $4, $5, 'renewal_due', $6, $7, $8, $9, $10, $11, '{}'::jsonb, $12, $12)`,
		attemptID,
		fixture.UserID,
		fixture.SubscriptionID,
		fixture.CustomerID,
		status,
		periodEnd,
		expiresAt,
		sentAt,
		clickedAt,
		paidAt,
		canceledAt,
		createdAt,
	)
	if err != nil {
		t.Fatalf("insert %s dunning attempt: %v", status, err)
	}

	return attemptID
}

func insertDunningMetricsCheckout(t *testing.T, pool *pgxpool.Pool, fixture dunningMetricsFixture, attemptID uuid.UUID) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := pool.Exec(ctx, `
INSERT INTO checkout_sessions (
	user_id,
	customer_id,
	subscription_id,
	mode,
	source,
	label,
	amount,
	currency,
	client_secret_hash,
	expires_at,
	metadata
)
VALUES ($1, $2, $3, 'renewal', 'dunning', 'Renew subscription', 5000, 'GHS', $4, NOW() + INTERVAL '30 minutes', jsonb_build_object('dunning_attempt_id', $5::text))`,
		fixture.UserID,
		fixture.CustomerID,
		fixture.SubscriptionID,
		"metrics_checkout_"+attemptID.String(),
		attemptID,
	)
	if err != nil {
		t.Fatalf("insert dunning metrics checkout: %v", err)
	}
}

func assertConversionMetrics(t *testing.T, actual *dunning.ConversionMetrics, expected dunning.ConversionMetrics) {
	t.Helper()

	if actual == nil {
		t.Fatal("expected conversion metrics")
	}
	if actual.Sent != expected.Sent {
		t.Fatalf("expected sent %d, got %d", expected.Sent, actual.Sent)
	}
	if actual.Clicked != expected.Clicked {
		t.Fatalf("expected clicked %d, got %d", expected.Clicked, actual.Clicked)
	}
	if actual.CheckoutStarted != expected.CheckoutStarted {
		t.Fatalf("expected checkout_started %d, got %d", expected.CheckoutStarted, actual.CheckoutStarted)
	}
	if actual.Paid != expected.Paid {
		t.Fatalf("expected paid %d, got %d", expected.Paid, actual.Paid)
	}
	if actual.Failed != expected.Failed {
		t.Fatalf("expected failed %d, got %d", expected.Failed, actual.Failed)
	}
	if actual.Expired != expected.Expired {
		t.Fatalf("expected expired %d, got %d", expected.Expired, actual.Expired)
	}
}
