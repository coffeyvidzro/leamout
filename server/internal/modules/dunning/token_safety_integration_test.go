package dunning_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cuffeyvidzro/leamout/internal/modules/checkout"
	"github.com/cuffeyvidzro/leamout/internal/modules/customer"
	dunning "github.com/cuffeyvidzro/leamout/internal/modules/dunning"
	"github.com/cuffeyvidzro/leamout/internal/modules/price"
	"github.com/cuffeyvidzro/leamout/internal/modules/product"
	"github.com/cuffeyvidzro/leamout/internal/modules/subscription"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type dunningTokenSafetyFixture struct {
	UserID         uuid.UUID
	CustomerID     uuid.UUID
	SubscriptionID uuid.UUID
	Attempt        *dunning.Attempt
	Service        *dunning.Service
	Pool           *pgxpool.Pool
}

func TestOpenRecoveryLinkRejectsExpiredToken(t *testing.T) {
	fixture := createDunningTokenSafetyFixture(t)
	rawToken, token, err := fixture.Service.CreateToken(context.Background(), fixture.Attempt)
	if err != nil {
		t.Fatalf("create dunning token: %v", err)
	}
	expireDunningToken(t, fixture.Pool, fixture.UserID, token.ID)

	_, err = fixture.Service.OpenRecoveryLink(context.Background(), rawToken)
	if !errors.Is(err, dunning.ErrNotFound) {
		t.Fatalf("expected expired dunning token to return ErrNotFound, got %v", err)
	}
	assertNoCheckoutCreatedForAttempt(t, fixture.Pool, fixture.Attempt.ID)

	stored, err := fixture.Service.GetByToken(context.Background(), rawToken)
	if err != nil {
		t.Fatalf("get expired dunning token: %v", err)
	}
	if stored.Token.LastUsedAt != nil {
		t.Fatal("expected expired dunning token not to record last_used_at")
	}
}

func TestOpenRecoveryLinkRejectsRevokedToken(t *testing.T) {
	fixture := createDunningTokenSafetyFixture(t)
	rawToken, token, err := fixture.Service.CreateToken(context.Background(), fixture.Attempt)
	if err != nil {
		t.Fatalf("create dunning token: %v", err)
	}
	if err := fixture.Service.RevokeToken(context.Background(), rawToken); err != nil {
		t.Fatalf("revoke dunning token: %v", err)
	}

	_, err = fixture.Service.OpenRecoveryLink(context.Background(), rawToken)
	if !errors.Is(err, dunning.ErrNotFound) {
		t.Fatalf("expected revoked dunning token to return ErrNotFound, got %v", err)
	}
	assertNoCheckoutCreatedForAttempt(t, fixture.Pool, fixture.Attempt.ID)

	stored, err := fixture.Service.GetByToken(context.Background(), rawToken)
	if err != nil {
		t.Fatalf("get revoked dunning token: %v", err)
	}
	if stored.Token.ID != token.ID {
		t.Fatalf("expected token id %s, got %s", token.ID, stored.Token.ID)
	}
	if stored.Token.RevokedAt == nil {
		t.Fatal("expected revoked dunning token to have revoked_at")
	}
	if stored.Token.LastUsedAt != nil {
		t.Fatal("expected revoked dunning token not to record last_used_at")
	}
}

func TestCreateTokenRejectsDuplicateActiveTokenForAttempt(t *testing.T) {
	fixture := createDunningTokenSafetyFixture(t)
	if _, _, err := fixture.Service.CreateToken(context.Background(), fixture.Attempt); err != nil {
		t.Fatalf("create first dunning token: %v", err)
	}

	_, _, err := fixture.Service.CreateToken(context.Background(), fixture.Attempt)
	if !errors.Is(err, dunning.ErrActiveTokenExists) {
		t.Fatalf("expected duplicate active dunning token to return ErrActiveTokenExists, got %v", err)
	}
	assertNoCheckoutCreatedForAttempt(t, fixture.Pool, fixture.Attempt.ID)
}

func TestOpenRecoveryLinkRejectsMalformedToken(t *testing.T) {
	fixture := createDunningTokenSafetyFixture(t)
	_, err := fixture.Service.OpenRecoveryLink(context.Background(), "not-a-real-dunning-token")
	if !errors.Is(err, dunning.ErrNotFound) {
		t.Fatalf("expected malformed dunning token to return ErrNotFound, got %v", err)
	}
	assertNoCheckoutCreatedForAttempt(t, fixture.Pool, fixture.Attempt.ID)
}

func createDunningTokenSafetyFixture(t *testing.T) dunningTokenSafetyFixture {
	t.Helper()

	ctx := context.Background()
	pool := openRenewalDunningTestDB(t)
	userID := createRenewalDunningTestUser(t, pool)

	productService := product.NewService(product.NewRepository(pool))
	customerService := customer.NewService(customer.NewRepository(pool))
	subscriptionService := subscription.NewService(subscription.NewRepository(pool))
	checkoutRepo := checkout.NewRepository(pool)
	checkoutService := checkout.NewService(checkoutRepo, nil)
	dunningService := dunning.NewService(dunning.NewRepository(pool), checkoutService)

	interval := price.IntervalMonth
	createdProduct, err := productService.Create(ctx, userID, product.CreateRequest{
		Name:   "Token Safety Plan",
		Prices: []price.CreateRequest{{Nickname: "Monthly", Type: price.TypeRecurring, UnitAmount: 5000, Currency: "GHS", Interval: &interval}},
	})
	if err != nil {
		t.Fatalf("create product with recurring price: %v", err)
	}
	if len(createdProduct.Prices) != 1 {
		t.Fatalf("expected one recurring price, got %d", len(createdProduct.Prices))
	}

	externalID := "customer_token_safety_" + userID.String()
	createdCustomer, err := customerService.Create(ctx, userID, customer.CreateRequest{Name: "Token Safety Customer", Phone: "+233241234567", ExternalID: &externalID})
	if err != nil {
		t.Fatalf("create customer: %v", err)
	}

	periodEnd := time.Now().UTC().Add(48 * time.Hour).Truncate(time.Microsecond)
	createdSubscription, err := subscriptionService.Create(ctx, userID, subscription.CreateRequest{
		CustomerID: &createdCustomer.ID, PriceID: createdProduct.Prices[0].ID, Status: subscription.StatusActive, CurrentPeriodStart: time.Now().UTC().Add(-28 * 24 * time.Hour).Truncate(time.Microsecond), CurrentPeriodEnd: periodEnd,
	})
	if err != nil {
		t.Fatalf("create subscription: %v", err)
	}

	attempt, err := dunningService.CreateOrReuseAttempt(ctx, dunning.CreateAttemptParams{
		UserID: userID, SubscriptionID: createdSubscription.ID, CustomerID: &createdCustomer.ID, Reason: dunning.AttemptReasonRenewalDue, PeriodEnd: periodEnd, ExpiresAt: time.Now().UTC().Add(24 * time.Hour), Metadata: map[string]any{"source": "token_safety_test"},
	})
	if err != nil {
		t.Fatalf("create dunning attempt: %v", err)
	}

	return dunningTokenSafetyFixture{UserID: userID, CustomerID: createdCustomer.ID, SubscriptionID: createdSubscription.ID, Attempt: attempt, Service: dunningService, Pool: pool}
}

func expireDunningToken(t *testing.T, pool *pgxpool.Pool, userID, tokenID uuid.UUID) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := pool.Exec(ctx, `
UPDATE dunning_tokens
SET created_at = NOW() - INTERVAL '2 hours',
	expires_at = NOW() - INTERVAL '1 hour',
	updated_at = NOW()
WHERE user_id = $1 AND id = $2`, userID, tokenID)
	if err != nil {
		t.Fatalf("expire dunning token: %v", err)
	}
}

func assertNoCheckoutCreatedForAttempt(t *testing.T, pool *pgxpool.Pool, attemptID uuid.UUID) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var count int
	if err := pool.QueryRow(ctx, `
SELECT COUNT(*)
FROM checkout_sessions
WHERE metadata->>'dunning_attempt_id' = $1`, attemptID.String()).Scan(&count); err != nil {
		t.Fatalf("count recovery checkout sessions: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no recovery checkout sessions, got %d", count)
	}
}
