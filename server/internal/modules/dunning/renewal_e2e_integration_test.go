package dunning

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cuffeyvidzro/leamout/internal/modules/billing"
	"github.com/cuffeyvidzro/leamout/internal/modules/checkout"
	"github.com/cuffeyvidzro/leamout/internal/modules/customer"
	"github.com/cuffeyvidzro/leamout/internal/modules/customermeter"
	"github.com/cuffeyvidzro/leamout/internal/modules/price"
	"github.com/cuffeyvidzro/leamout/internal/modules/product"
	"github.com/cuffeyvidzro/leamout/internal/modules/subscription"
	"github.com/cuffeyvidzro/leamout/internal/sms"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

type recordingSMSSender struct {
	messages []sms.Message
}

func (s *recordingSMSSender) Send(_ context.Context, message sms.Message) error {
	s.messages = append(s.messages, message)
	return nil
}

func TestRenewalDunningEndToEnd(t *testing.T) {
	ctx := context.Background()
	pool := openRenewalDunningTestDB(t)
	userID := createRenewalDunningTestUser(t, pool)

	productService := product.NewService(product.NewRepository(pool))
	customerService := customer.NewService(customer.NewRepository(pool))
	subscriptionService := subscription.NewService(subscription.NewRepository(pool))
	checkoutRepo := checkout.NewRepository(pool)
	checkoutService := checkout.NewService(checkoutRepo, nil)
	billingService := billing.NewService(pool, customermeter.NewRepository(pool))
	dunningService := NewService(NewRepository(pool), checkoutService)

	interval := price.IntervalMonth
	createdProduct, err := productService.Create(ctx, userID, product.CreateRequest{
		Name: "Creator Pro",
		Prices: []price.CreateRequest{
			{
				Nickname:   "Monthly",
				Type:       price.TypeRecurring,
				UnitAmount: 5000,
				Currency:   "GHS",
				Interval:   &interval,
			},
		},
	})
	if err != nil {
		t.Fatalf("create product with recurring price: %v", err)
	}
	if len(createdProduct.Prices) != 1 {
		t.Fatalf("expected one recurring price, got %d", len(createdProduct.Prices))
	}

	externalID := "customer_renewal_e2e_" + userID.String()
	createdCustomer, err := customerService.Create(ctx, userID, customer.CreateRequest{
		Name:       "Renewal E2E Customer",
		Phone:      "+233241234567",
		ExternalID: &externalID,
	})
	if err != nil {
		t.Fatalf("create customer: %v", err)
	}

	periodStart := time.Now().UTC().Add(-28 * 24 * time.Hour).Truncate(time.Microsecond)
	periodEnd := time.Now().UTC().Add(48 * time.Hour).Truncate(time.Microsecond)
	createdSubscription, err := subscriptionService.Create(ctx, userID, subscription.CreateRequest{
		CustomerID:         &createdCustomer.ID,
		PriceID:            createdProduct.Prices[0].ID,
		Status:             subscription.StatusActive,
		CurrentPeriodStart: periodStart,
		CurrentPeriodEnd:   periodEnd,
	})
	if err != nil {
		t.Fatalf("create subscription: %v", err)
	}

	candidates, err := subscriptionService.ListDueForDunning(ctx, time.Now().UTC().Add(72*time.Hour))
	if err != nil {
		t.Fatalf("scan subscriptions due for dunning: %v", err)
	}
	candidate := findDunningCandidate(t, candidates, createdSubscription.ID)
	if candidate.CustomerID == nil || *candidate.CustomerID != createdCustomer.ID {
		t.Fatalf("expected dunning candidate customer %s, got %v", createdCustomer.ID, candidate.CustomerID)
	}

	sender := &recordingSMSSender{}
	worker := NewSendReminderWorker(dunningService, sender, "https://lmt.test", nil)
	if err := worker.Work(ctx, &river.Job[SendReminderArgs]{Args: SendReminderArgs{
		UserID:           candidate.UserID,
		SubscriptionID:   candidate.ID,
		CustomerID:       *candidate.CustomerID,
		CurrentPeriodEnd: candidate.CurrentPeriodEnd,
	}}); err != nil {
		t.Fatalf("send dunning reminder: %v", err)
	}
	if len(sender.messages) != 1 {
		t.Fatalf("expected one dunning SMS, got %d", len(sender.messages))
	}
	if sender.messages[0].To != createdCustomer.Phone {
		t.Fatalf("expected SMS to %s, got %s", createdCustomer.Phone, sender.messages[0].To)
	}
	if !strings.Contains(sender.messages[0].Reference, "dunning_sms:") {
		t.Fatalf("expected dunning SMS reference, got %q", sender.messages[0].Reference)
	}

	attempts, err := dunningService.List(ctx, userID)
	if err != nil {
		t.Fatalf("list dunning attempts: %v", err)
	}
	if len(attempts) != 1 {
		t.Fatalf("expected one dunning attempt, got %d", len(attempts))
	}
	attempt := attempts[0]
	if attempt.Status != AttemptStatusSent {
		t.Fatalf("expected dunning attempt sent, got %s", attempt.Status)
	}
	if attempt.SentAt == nil {
		t.Fatal("expected dunning attempt sent_at to be set")
	}

	rawToken := extractReminderToken(t, sender.messages[0].Content)
	checkoutSession, err := dunningService.OpenRecoveryLink(ctx, rawToken)
	if err != nil {
		t.Fatalf("open dunning recovery link: %v", err)
	}
	if checkoutSession.Mode != checkout.ModeRenewal {
		t.Fatalf("expected renewal checkout mode, got %s", checkoutSession.Mode)
	}
	if checkoutSession.Source != checkout.SourceDunning {
		t.Fatalf("expected dunning checkout source, got %s", checkoutSession.Source)
	}
	if checkoutSession.SubscriptionID == nil || *checkoutSession.SubscriptionID != createdSubscription.ID {
		t.Fatalf("expected checkout subscription %s, got %v", createdSubscription.ID, checkoutSession.SubscriptionID)
	}
	if checkoutSession.Amount != createdProduct.Prices[0].UnitAmount {
		t.Fatalf("expected checkout amount %d, got %d", createdProduct.Prices[0].UnitAmount, checkoutSession.Amount)
	}

	usedToken, err := dunningService.GetByToken(ctx, rawToken)
	if err != nil {
		t.Fatalf("get used dunning token: %v", err)
	}
	if usedToken.Token.LastUsedAt == nil {
		t.Fatal("expected dunning token last_used_at to be set after opening recovery link")
	}

	if err := billingService.CompletePaidCheckout(ctx, checkoutSession.ID); err != nil {
		t.Fatalf("complete paid dunning checkout: %v", err)
	}

	renewedSubscription, err := subscriptionService.Get(ctx, userID, createdSubscription.ID)
	if err != nil {
		t.Fatalf("get renewed subscription: %v", err)
	}
	if !renewedSubscription.CurrentPeriodStart.Equal(createdSubscription.CurrentPeriodEnd) {
		t.Fatalf("expected renewed period start %s, got %s", createdSubscription.CurrentPeriodEnd, renewedSubscription.CurrentPeriodStart)
	}
	if !renewedSubscription.CurrentPeriodEnd.After(createdSubscription.CurrentPeriodEnd) {
		t.Fatalf("expected renewed period end after %s, got %s", createdSubscription.CurrentPeriodEnd, renewedSubscription.CurrentPeriodEnd)
	}

	paidAttempt, err := dunningService.Get(ctx, userID, attempt.ID)
	if err != nil {
		t.Fatalf("get paid dunning attempt: %v", err)
	}
	if paidAttempt.Status != AttemptStatusPaid {
		t.Fatalf("expected dunning attempt paid, got %s", paidAttempt.Status)
	}
	if paidAttempt.PaidAt == nil {
		t.Fatal("expected dunning attempt paid_at to be set")
	}

	revokedToken, err := dunningService.GetByToken(ctx, rawToken)
	if err != nil {
		t.Fatalf("get revoked dunning token: %v", err)
	}
	if revokedToken.Token.RevokedAt == nil {
		t.Fatal("expected dunning token to be revoked after paid renewal")
	}

	completedCheckout, err := checkoutService.Get(ctx, userID, checkoutSession.ID)
	if err != nil {
		t.Fatalf("get completed checkout: %v", err)
	}
	if completedCheckout.Status != checkout.StatusCompleted {
		t.Fatalf("expected checkout completed, got %s", completedCheckout.Status)
	}
	if completedCheckout.CompletedAt == nil {
		t.Fatal("expected checkout completed_at to be set")
	}
}

func openRenewalDunningTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("set DATABASE_URL to run renewal dunning end-to-end integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping test database: %v", err)
	}

	return pool
}

func createRenewalDunningTestUser(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID := uuid.New()
	_, err := pool.Exec(ctx, `
INSERT INTO users (id, name, email, email_verified, status)
VALUES ($1, 'Renewal Dunning E2E User', $2, TRUE, 'active')`,
		userID,
		fmt.Sprintf("renewal-dunning-e2e-%s@example.com", userID),
	)
	if err != nil {
		t.Fatalf("insert test user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})

	return userID
}

func findDunningCandidate(t *testing.T, candidates []subscription.DunningCandidate, subscriptionID uuid.UUID) subscription.DunningCandidate {
	t.Helper()

	for _, candidate := range candidates {
		if candidate.ID == subscriptionID {
			return candidate
		}
	}

	t.Fatalf("subscription %s was not returned by dunning scan", subscriptionID)
	return subscription.DunningCandidate{}
}

func extractReminderToken(t *testing.T, message string) string {
	t.Helper()

	idx := strings.LastIndex(message, "/r/")
	if idx < 0 {
		t.Fatalf("expected reminder message to contain recovery link, got %q", message)
	}
	rawToken := message[idx+len("/r/"):]
	if spaceIdx := strings.IndexAny(rawToken, " \n\t"); spaceIdx >= 0 {
		rawToken = rawToken[:spaceIdx]
	}
	decoded, err := url.PathUnescape(rawToken)
	if err != nil {
		t.Fatalf("decode reminder token: %v", err)
	}
	if decoded == "" {
		t.Fatal("expected non-empty reminder token")
	}

	return decoded
}
