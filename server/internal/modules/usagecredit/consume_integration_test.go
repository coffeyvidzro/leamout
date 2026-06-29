package usagecredit

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type usageCreditConcurrencyFixture struct {
	UserID     uuid.UUID
	CustomerID uuid.UUID
	MeterID    uuid.UUID
	GrantID    uuid.UUID
}

func TestConsumeUsageEventConcurrentSameIdempotencyKeyDebitsOnce(t *testing.T) {
	pool := openUsageCreditTestDB(t)
	fixture := createUsageCreditConcurrencyFixture(t, pool, 100)
	eventID := createUsageCreditTestEvent(t, pool, fixture, "same-key")
	idempotencyKey := fmt.Sprintf("usage_event:%s:meter:%s", eventID, fixture.MeterID)

	consumeConcurrently(t, pool, 24, func(worker int) error {
		return consumeUsageCreditInTx(pool, fixture, eventID, 10, idempotencyKey)
	})

	remaining, ledgerQuantity, ledgerCount := readUsageCreditGrantStats(t, pool, fixture)
	if remaining != 90 {
		t.Fatalf("expected remaining quantity 90 after idempotent concurrent consume, got %v", remaining)
	}
	if ledgerQuantity != 10 {
		t.Fatalf("expected one debited quantity of 10, got %v", ledgerQuantity)
	}
	if ledgerCount != 1 {
		t.Fatalf("expected one debit ledger entry, got %d", ledgerCount)
	}
}

func TestConsumeUsageEventConcurrentUniqueKeysSerializesGrantDebits(t *testing.T) {
	pool := openUsageCreditTestDB(t)
	fixture := createUsageCreditConcurrencyFixture(t, pool, 100)

	consumeConcurrently(t, pool, 10, func(worker int) error {
		eventID := createUsageCreditTestEvent(t, pool, fixture, fmt.Sprintf("unique-key-%d", worker))
		idempotencyKey := fmt.Sprintf("usage_event:%s:meter:%s", eventID, fixture.MeterID)
		return consumeUsageCreditInTx(pool, fixture, eventID, 7, idempotencyKey)
	})

	remaining, ledgerQuantity, ledgerCount := readUsageCreditGrantStats(t, pool, fixture)
	if remaining != 30 {
		t.Fatalf("expected remaining quantity 30 after concurrent unique consumes, got %v", remaining)
	}
	if ledgerQuantity != 70 {
		t.Fatalf("expected total debited quantity 70, got %v", ledgerQuantity)
	}
	if ledgerCount != 10 {
		t.Fatalf("expected ten debit ledger entries, got %d", ledgerCount)
	}
}

func openUsageCreditTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("set DATABASE_URL to run usage credit concurrency integration tests")
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

func createUsageCreditConcurrencyFixture(t *testing.T, pool *pgxpool.Pool, quantity float64) usageCreditConcurrencyFixture {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fixture := usageCreditConcurrencyFixture{
		UserID:     uuid.New(),
		CustomerID: uuid.New(),
		MeterID:    uuid.New(),
		GrantID:    uuid.New(),
	}
	benefitID := uuid.New()
	benefitGrantID := uuid.New()
	manualSourceID := uuid.New()

	_, err := pool.Exec(ctx, `
INSERT INTO users (id, name, email, email_verified, status)
VALUES ($1, 'Usage Credit Test User', $2, TRUE, 'active')`,
		fixture.UserID,
		fmt.Sprintf("usage-credit-%s@example.com", fixture.UserID),
	)
	if err != nil {
		t.Fatalf("insert test user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, fixture.UserID)
	})

	_, err = pool.Exec(ctx, `
INSERT INTO customers (id, user_id, name, phone, external_id, address, metadata)
VALUES ($1, $2, 'Usage Credit Test Customer', $3, $4, '{}'::jsonb, '{}'::jsonb)`,
		fixture.CustomerID,
		fixture.UserID,
		fmt.Sprintf("+233%s", strings.ReplaceAll(fixture.CustomerID.String(), "-", "")[:9]),
		fmt.Sprintf("customer_%s", fixture.CustomerID),
	)
	if err != nil {
		t.Fatalf("insert test customer: %v", err)
	}

	_, err = pool.Exec(ctx, `
INSERT INTO meters (id, user_id, name, event_filter, aggregation)
VALUES ($1, $2, $3, '{"conjunction":"and","clauses":[]}'::jsonb, '{"func":"count"}'::jsonb)`,
		fixture.MeterID,
		fixture.UserID,
		fmt.Sprintf("API Calls %s", fixture.MeterID.String()[:8]),
	)
	if err != nil {
		t.Fatalf("insert test meter: %v", err)
	}

	_, err = pool.Exec(ctx, `
INSERT INTO benefits (id, user_id, type, name, code, properties, metadata)
VALUES ($1, $2, 'meter_credit', 'API Calls', $3, jsonb_build_object('meter_id', $4::text, 'quantity', '100'), '{}'::jsonb)`,
		benefitID,
		fixture.UserID,
		fmt.Sprintf("api_calls_%s", benefitID.String()[:8]),
		fixture.MeterID,
	)
	if err != nil {
		t.Fatalf("insert test benefit: %v", err)
	}

	_, err = pool.Exec(ctx, `
INSERT INTO benefit_grants (
	id,
	user_id,
	benefit_id,
	customer_id,
	source_type,
	source_id,
	status,
	starts_at,
	properties,
	metadata
)
VALUES ($1, $2, $3, $4, 'manual', $5, 'active', NOW(), jsonb_build_object('meter_id', $6::text, 'quantity', '100'), '{}'::jsonb)`,
		benefitGrantID,
		fixture.UserID,
		benefitID,
		fixture.CustomerID,
		manualSourceID,
		fixture.MeterID,
	)
	if err != nil {
		t.Fatalf("insert test benefit grant: %v", err)
	}

	_, err = pool.Exec(ctx, `
INSERT INTO meter_credit_grants (
	id,
	user_id,
	customer_id,
	meter_id,
	benefit_grant_id,
	source_type,
	source_id,
	status,
	quantity,
	remaining_quantity,
	starts_at,
	metadata
)
VALUES ($1, $2, $3, $4, $5, 'manual', $6, 'active', $7, $7, NOW(), '{}'::jsonb)`,
		fixture.GrantID,
		fixture.UserID,
		fixture.CustomerID,
		fixture.MeterID,
		benefitGrantID,
		uuid.New(),
		quantity,
	)
	if err != nil {
		t.Fatalf("insert test usage credit grant: %v", err)
	}

	return fixture
}

func createUsageCreditTestEvent(t *testing.T, pool *pgxpool.Pool, fixture usageCreditConcurrencyFixture, suffix string) uuid.UUID {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	eventID := uuid.New()
	_, err := pool.Exec(ctx, `
INSERT INTO usage_events (id, user_id, name, source, customer_id, external_id, metadata)
VALUES ($1, $2, 'api_call', 'user', $3, $4, '{}'::jsonb)`,
		eventID,
		fixture.UserID,
		fixture.CustomerID,
		fmt.Sprintf("usage_credit_%s_%s", eventID, suffix),
	)
	if err != nil {
		t.Fatalf("insert test usage event: %v", err)
	}

	return eventID
}

func consumeConcurrently(t *testing.T, pool *pgxpool.Pool, workers int, consume func(worker int) error) {
	t.Helper()

	start := make(chan struct{})
	errs := make(chan error, workers)
	var wg sync.WaitGroup

	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			<-start
			if err := consume(worker); err != nil {
				errs <- err
			}
		}(worker)
	}

	close(start)
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent consume failed: %v", err)
		}
	}
}

func consumeUsageCreditInTx(pool *pgxpool.Pool, fixture usageCreditConcurrencyFixture, eventID uuid.UUID, quantity float64, idempotencyKey string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin consume tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	repo := NewRepository(pool)
	if err := repo.ConsumeUsageEvent(ctx, tx, fixture.UserID, fixture.CustomerID, eventID, fixture.MeterID, quantity, idempotencyKey); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit consume tx: %w", err)
	}
	return nil
}

func readUsageCreditGrantStats(t *testing.T, pool *pgxpool.Pool, fixture usageCreditConcurrencyFixture) (float64, float64, int64) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var remaining float64
	if err := pool.QueryRow(ctx, `
SELECT remaining_quantity::float8
FROM meter_credit_grants
WHERE user_id = $1 AND id = $2`, fixture.UserID, fixture.GrantID).Scan(&remaining); err != nil {
		t.Fatalf("read remaining grant quantity: %v", err)
	}

	var ledgerQuantity float64
	var ledgerCount int64
	if err := pool.QueryRow(ctx, `
SELECT COALESCE(SUM(quantity), 0)::float8, COUNT(*)
FROM meter_credit_ledger_entries
WHERE user_id = $1
  AND grant_id = $2
  AND direction = 'debit'
  AND reason = 'consume'`, fixture.UserID, fixture.GrantID).Scan(&ledgerQuantity, &ledgerCount); err != nil {
		t.Fatalf("read usage credit ledger stats: %v", err)
	}

	return remaining, ledgerQuantity, ledgerCount
}
