package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cuffeyvidzro/leamout/internal/config"
	"github.com/cuffeyvidzro/leamout/internal/modules/checkout"
	"github.com/cuffeyvidzro/leamout/internal/modules/dunning"
	"github.com/cuffeyvidzro/leamout/internal/modules/subscription"
	"github.com/cuffeyvidzro/leamout/internal/platform/database"
	"github.com/cuffeyvidzro/leamout/internal/platform/logger"
	"github.com/cuffeyvidzro/leamout/internal/platform/queue"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

const (
	demoUserEmail     = "demo@leamout.local"
	demoUserName      = "Leamout Demo Merchant"
	demoCustomerName  = "Ama Demo Customer"
	demoCustomerPhone = "+233501234567"
	demoProductName   = "Demo Monthly Plan"
	demoPriceNickname = "Demo Monthly GHS"
	demoCurrency      = "GHS"
	demoUnitAmount    = 5000
)

func main() {
	log := logger.New()
	if err := run(log); err != nil {
		log.Error("demo command failed", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	if len(os.Args) < 2 {
		return usage()
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx := context.Background()
	db, err := database.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	switch os.Args[1] {
	case "migrate":
		return runMigrations(ctx, db)
	case "seed":
		return seed(ctx, db)
	case "scan":
		return scan(ctx, db, cfg, log)
	case "complete":
		return complete(ctx, db)
	case "verify":
		return verify(ctx, db)
	default:
		return usage()
	}
}

func usage() error {
	fmt.Fprintln(os.Stderr, "usage: go run ./cmd/demo <migrate|seed|scan|complete|verify>")
	return errors.New("unknown demo command")
}

func runMigrations(ctx context.Context, db *pgxpool.Pool) error {
	if err := runAppMigrations(ctx, db); err != nil {
		return err
	}

	migrator, err := rivermigrate.New(riverpgxv5.New(db), nil)
	if err != nil {
		return fmt.Errorf("create river migrator: %w", err)
	}
	if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
		return fmt.Errorf("run river migrations: %w", err)
	}

	fmt.Println("migrations applied")
	return nil
}

func runAppMigrations(ctx context.Context, db *pgxpool.Pool) error {
	if _, err := db.Exec(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
	filename TEXT PRIMARY KEY,
	applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	files, err := filepath.Glob("migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(files)

	for _, file := range files {
		filename := filepath.Base(file)
		var applied bool
		if err := db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE filename = $1)`, filename).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %s: %w", filename, err)
		}
		if applied {
			continue
		}

		data, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", filename, err)
		}
		upSQL := upMigrationSQL(string(data))
		if strings.TrimSpace(upSQL) == "" {
			continue
		}

		tx, err := db.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", filename, err)
		}
		if _, err := tx.Exec(ctx, upSQL); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("apply migration %s: %w", filename, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (filename) VALUES ($1)`, filename); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("record migration %s: %w", filename, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration %s: %w", filename, err)
		}

		fmt.Printf("applied %s\n", filename)
	}

	return nil
}

func upMigrationSQL(contents string) string {
	parts := strings.Split(contents, "-- +goose Down")
	return strings.ReplaceAll(parts[0], "-- +goose Up", "")
}

func seed(ctx context.Context, db *pgxpool.Pool) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, `DELETE FROM users WHERE email = $1`, demoUserEmail); err != nil {
		return fmt.Errorf("clear existing demo user: %w", err)
	}

	var userID, productID, priceID, customerID, subscriptionID uuid.UUID
	if err := tx.QueryRow(ctx, `
INSERT INTO users (name, email, email_verified, status)
VALUES ($1, $2, TRUE, 'active')
RETURNING id`, demoUserName, demoUserEmail).Scan(&userID); err != nil {
		return fmt.Errorf("insert demo user: %w", err)
	}
	if err := tx.QueryRow(ctx, `
INSERT INTO products (user_id, name, description, active, metadata)
VALUES ($1, $2, 'Local dunning demo product', TRUE, '{"demo": true}'::jsonb)
RETURNING id`, userID, demoProductName).Scan(&productID); err != nil {
		return fmt.Errorf("insert demo product: %w", err)
	}
	if err := tx.QueryRow(ctx, `
INSERT INTO prices (user_id, product_id, nickname, type, unit_amount, currency, interval, metadata)
VALUES ($1, $2, $3, 'recurring', $4, $5, 'month', '{"demo": true}'::jsonb)
RETURNING id`, userID, productID, demoPriceNickname, demoUnitAmount, demoCurrency).Scan(&priceID); err != nil {
		return fmt.Errorf("insert demo price: %w", err)
	}
	if err := tx.QueryRow(ctx, `
INSERT INTO customers (user_id, name, email, phone, external_id, address, metadata)
VALUES ($1, $2, 'ama.demo@example.com', $3, 'demo-customer', '{}'::jsonb, '{"demo": true}'::jsonb)
RETURNING id`, userID, demoCustomerName, demoCustomerPhone).Scan(&customerID); err != nil {
		return fmt.Errorf("insert demo customer: %w", err)
	}

	periodStart := time.Now().UTC()
	periodEnd := periodStart.Add(dunning.DefaultScanWindow)
	if err := tx.QueryRow(ctx, `
INSERT INTO subscriptions (
	user_id, customer_id, price_id, status, current_period_start, current_period_end, metadata
)
VALUES ($1, $2, $3, 'active', $4, $5, '{"demo": true}'::jsonb)
RETURNING id`, userID, customerID, priceID, periodStart, periodEnd).Scan(&subscriptionID); err != nil {
		return fmt.Errorf("insert demo subscription: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	fmt.Printf("demo_user_id=%s\n", userID)
	fmt.Printf("demo_customer_id=%s\n", customerID)
	fmt.Printf("demo_subscription_id=%s\n", subscriptionID)
	fmt.Printf("demo_subscription_current_period_end=%s\n", periodEnd.Format(time.RFC3339))
	return nil
}

func scan(ctx context.Context, db *pgxpool.Pool, cfg *config.Config, log *slog.Logger) error {
	workers := queue.NewWorkerRegistry()
	dunning.RegisterReminderJobKind(workers)

	queueClient, err := queue.NewClient(db, workers, queue.Config{Enabled: false})
	if err != nil {
		return err
	}

	subscriptions := subscription.NewService(subscription.NewRepository(db))
	scanner := dunning.NewScanner(subscriptions, func(ctx context.Context, args dunning.SendReminderArgs) error {
		return queueClient.Insert(ctx, args, nil)
	}, log)
	result, err := scanner.RunOnce(ctx)
	if err != nil {
		return err
	}

	fmt.Printf("scanned=%d enqueued=%d skipped=%d window_end=%s\n",
		result.Scanned,
		result.Enqueued,
		result.Skipped,
		result.WindowEnd.Format(time.RFC3339),
	)
	return nil
}

func complete(ctx context.Context, db *pgxpool.Pool) error {
	flags := flag.NewFlagSet("complete", flag.ContinueOnError)
	token := flags.String("token", "", "raw dunning token from the mock SMS link")
	if err := flags.Parse(os.Args[2:]); err != nil {
		return err
	}
	if strings.TrimSpace(*token) == "" {
		return errors.New("complete requires -token")
	}

	checkoutService := checkout.NewService(checkout.NewRepository(db))
	dunningService := dunning.NewService(dunning.NewRepository(db), checkoutService)

	session, err := dunningService.OpenRecoveryLink(ctx, *token)
	if err != nil {
		return fmt.Errorf("open dunning recovery link: %w", err)
	}
	confirmed, err := checkoutService.Confirm(ctx, session.ClientSecret)
	if err != nil {
		return fmt.Errorf("confirm checkout: %w", err)
	}

	fmt.Printf("checkout_session_id=%s\n", confirmed.ID)
	fmt.Printf("checkout_status=%s\n", confirmed.Status)
	if confirmed.SubscriptionID != nil {
		fmt.Printf("subscription_id=%s\n", *confirmed.SubscriptionID)
	}
	return nil
}

func verify(ctx context.Context, db *pgxpool.Pool) error {
	const query = `
SELECT
	u.id,
	s.id,
	s.current_period_end,
	COALESCE(da.status::text, 'missing'),
	da.sent_at IS NOT NULL,
	da.paid_at IS NOT NULL,
	COALESCE(cs.status::text, 'missing')
FROM users u
JOIN subscriptions s ON s.user_id = u.id
LEFT JOIN dunning_attempts da ON da.user_id = u.id AND da.subscription_id = s.id
LEFT JOIN checkout_sessions cs ON cs.user_id = u.id AND cs.subscription_id = s.id
WHERE u.email = $1
ORDER BY da.created_at DESC NULLS LAST, cs.created_at DESC NULLS LAST
LIMIT 1`

	var userID, subscriptionID uuid.UUID
	var periodEnd time.Time
	var attemptStatus, checkoutStatus string
	var sent, paid bool
	if err := db.QueryRow(ctx, query, demoUserEmail).Scan(
		&userID,
		&subscriptionID,
		&periodEnd,
		&attemptStatus,
		&sent,
		&paid,
		&checkoutStatus,
	); err != nil {
		return fmt.Errorf("verify demo state: %w", err)
	}

	fmt.Printf("demo_user_id=%s\n", userID)
	fmt.Printf("demo_subscription_id=%s\n", subscriptionID)
	fmt.Printf("subscription_current_period_end=%s\n", periodEnd.Format(time.RFC3339))
	fmt.Printf("dunning_attempt_status=%s sent=%t paid=%t\n", attemptStatus, sent, paid)
	fmt.Printf("checkout_status=%s\n", checkoutStatus)
	return nil
}
