package billing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/cuffeyvidzro/leamout/internal/modules/checkout"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvalidCheckoutCompletion = errors.New("invalid checkout completion")
	ErrCheckoutNotFound         = errors.New("checkout session not found")
)

type UsageCreditApplier interface {
	ApplySubscriptionCredits(ctx context.Context, tx pgx.Tx, userID, subscriptionID, checkoutID uuid.UUID, fallbackCustomerID *uuid.UUID) error
}

type Service struct {
	db           *pgxpool.Pool
	usageCredits UsageCreditApplier
	transactions TransactionCreator
	wallet       WalletCreditor
}

func NewService(db *pgxpool.Pool, usageCredits ...UsageCreditApplier) *Service {
	service := &Service{db: db}
	if len(usageCredits) > 0 {
		service.usageCredits = usageCredits[0]
	}
	return service
}

// CompletePaidCheckout owns the business flow that happens after a checkout has
// been paid. Checkout remains responsible for checkout state; billing coordinates
// cross-module effects such as subscription renewal, dunning settlement, benefit
// grants, and usage-credit grants.
func (s *Service) CompletePaidCheckout(ctx context.Context, checkoutID uuid.UUID) error {
	if s.db == nil {
		return errors.New("billing database is not configured")
	}
	if checkoutID == uuid.Nil {
		return fmt.Errorf("%w: checkout id is required", ErrInvalidCheckoutCompletion)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin billing checkout completion: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	session, err := s.lockCheckoutSession(ctx, tx, checkoutID)
	if err != nil {
		return err
	}
	if session.Status == checkout.StatusCompleted {
		if err := s.fulfillSubscriptionBenefits(ctx, tx, session); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	if session.Status != checkout.StatusOpen {
		return tx.Commit(ctx)
	}

	session, err = s.completeCheckoutState(ctx, tx, checkoutID)
	if err != nil {
		return err
	}
	if isDunningRenewal(session) {
		if err := s.completeDunningRenewal(ctx, tx, session); err != nil {
			return err
		}
	}
	if err := s.fulfillSubscriptionBenefits(ctx, tx, session); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit billing checkout completion: %w", err)
	}
	return nil
}

func (s *Service) lockCheckoutSession(ctx context.Context, tx pgx.Tx, checkoutID uuid.UUID) (*checkout.Session, error) {
	const query = `
SELECT id, user_id, customer_id, subscription_id, mode, source, label, amount, currency,
	client_secret_hash, success_url, return_url, status, expires_at, completed_at, canceled_at,
	metadata, created_at, updated_at
FROM checkout_sessions
WHERE id = $1
FOR UPDATE`

	session, err := scanCheckoutSession(tx.QueryRow(ctx, query, checkoutID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCheckoutNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock checkout session for billing completion: %w", err)
	}
	return session, nil
}

func (s *Service) completeCheckoutState(ctx context.Context, tx pgx.Tx, checkoutID uuid.UUID) (*checkout.Session, error) {
	const query = `
UPDATE checkout_sessions
SET status = 'completed', completed_at = COALESCE(completed_at, NOW())
WHERE id = $1
  AND status = 'open'
RETURNING id, user_id, customer_id, subscription_id, mode, source, label, amount, currency,
	client_secret_hash, success_url, return_url, status, expires_at, completed_at, canceled_at,
	metadata, created_at, updated_at`

	session, err := scanCheckoutSession(tx.QueryRow(ctx, query, checkoutID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCheckoutNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("complete checkout state: %w", err)
	}
	return session, nil
}

func (s *Service) completeDunningRenewal(ctx context.Context, tx pgx.Tx, session *checkout.Session) error {
	if !isDunningRenewal(session) {
		return nil
	}

	attemptID, err := metadataUUID(session.Metadata, "dunning_attempt_id")
	if err != nil {
		return err
	}
	tokenID, err := metadataUUID(session.Metadata, "dunning_token_id")
	if err != nil {
		return err
	}

	const extendSubscription = `
UPDATE subscriptions s
SET current_period_start = s.current_period_end,
	current_period_end = CASE p.interval
		WHEN 'day' THEN s.current_period_end + INTERVAL '1 day'
		WHEN 'week' THEN s.current_period_end + INTERVAL '1 week'
		WHEN 'month' THEN s.current_period_end + INTERVAL '1 month'
		WHEN 'year' THEN s.current_period_end + INTERVAL '1 year'
		ELSE s.current_period_end
	END
FROM prices p
WHERE s.user_id = $1
  AND s.id = $2
  AND p.user_id = s.user_id
  AND p.id = s.price_id
  AND p.type = 'recurring'
  AND p.interval IS NOT NULL
  AND s.status = 'active'
  AND s.cancel_at_period_end = FALSE`

	tag, err := tx.Exec(ctx, extendSubscription, session.UserID, *session.SubscriptionID)
	if err != nil {
		return fmt.Errorf("renew subscription period from billing checkout completion: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("%w: subscription renewal did not match one active subscription", ErrInvalidCheckoutCompletion)
	}

	if err := setDunningTransitionContext(ctx, tx, "billing", "renewal_paid", map[string]any{
		"source":              "billing_checkout_completion",
		"checkout_session_id": session.ID.String(),
	}); err != nil {
		return err
	}

	const markAttemptPaid = `
UPDATE dunning_attempts
SET status = 'paid', sent_at = COALESCE(sent_at, NOW()), paid_at = COALESCE(paid_at, NOW())
WHERE user_id = $1
  AND id = $2
  AND subscription_id = $3
  AND status IN ('pending', 'sent')`

	tag, err = tx.Exec(ctx, markAttemptPaid, session.UserID, attemptID, *session.SubscriptionID)
	if err != nil {
		return fmt.Errorf("mark dunning attempt paid from billing checkout completion: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("%w: dunning attempt paid transition did not match one attempt", ErrInvalidCheckoutCompletion)
	}

	const revokeToken = `
UPDATE dunning_tokens
SET revoked_at = COALESCE(revoked_at, NOW())
WHERE user_id = $1
  AND id = $2
  AND dunning_attempt_id = $3
  AND revoked_at IS NULL`

	tag, err = tx.Exec(ctx, revokeToken, session.UserID, tokenID, attemptID)
	if err != nil {
		return fmt.Errorf("revoke dunning token from billing checkout completion: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("%w: dunning token revocation did not match one token", ErrInvalidCheckoutCompletion)
	}

	return nil
}

func (s *Service) fulfillSubscriptionBenefits(ctx context.Context, tx pgx.Tx, session *checkout.Session) error {
	if session == nil || session.SubscriptionID == nil {
		return nil
	}

	const query = `
INSERT INTO benefit_grants (
	user_id,
	benefit_id,
	customer_id,
	product_id,
	subscription_id,
	source_type,
	source_id,
	status,
	starts_at,
	ends_at,
	properties,
	metadata
)
SELECT
	s.user_id,
	b.id,
	COALESCE(s.customer_id, $3),
	p.product_id,
	s.id,
	'subscription',
	s.id,
	'active',
	s.current_period_start,
	s.current_period_end,
	b.properties,
	jsonb_build_object(
		'source', 'billing_checkout_completion',
		'checkout_session_id', $4::text,
		'subscription_id', s.id::text,
		'product_id', p.product_id::text
	)
FROM subscriptions s
JOIN prices p
  ON p.user_id = s.user_id
 AND p.id = s.price_id
JOIN product_benefits pb
  ON pb.user_id = s.user_id
 AND pb.product_id = p.product_id
JOIN benefits b
  ON b.user_id = pb.user_id
 AND b.id = pb.benefit_id
 AND b.archived_at IS NULL
WHERE s.user_id = $1
  AND s.id = $2
  AND COALESCE(s.customer_id, $3) IS NOT NULL
ON CONFLICT (user_id, customer_id, benefit_id, source_type, source_id)
DO UPDATE SET
	product_id = EXCLUDED.product_id,
	subscription_id = EXCLUDED.subscription_id,
	status = 'active',
	starts_at = EXCLUDED.starts_at,
	ends_at = EXCLUDED.ends_at,
	properties = EXCLUDED.properties,
	metadata = benefit_grants.metadata || EXCLUDED.metadata,
	revoked_at = NULL,
	updated_at = NOW()`

	if _, err := tx.Exec(ctx, query, session.UserID, *session.SubscriptionID, session.CustomerID, session.ID); err != nil {
		return fmt.Errorf("fulfill subscription benefits from billing checkout completion: %w", err)
	}

	if s.usageCredits != nil {
		if err := s.usageCredits.ApplySubscriptionCredits(ctx, tx, session.UserID, *session.SubscriptionID, session.ID, session.CustomerID); err != nil {
			return err
		}
	}

	return nil
}

func scanCheckoutSession(row pgx.Row) (*checkout.Session, error) {
	var session checkout.Session
	var metadataBytes []byte

	if err := row.Scan(
		&session.ID,
		&session.UserID,
		&session.CustomerID,
		&session.SubscriptionID,
		&session.Mode,
		&session.Source,
		&session.Label,
		&session.Amount,
		&session.Currency,
		&session.ClientSecretHash,
		&session.SuccessURL,
		&session.ReturnURL,
		&session.Status,
		&session.ExpiresAt,
		&session.CompletedAt,
		&session.CanceledAt,
		&metadataBytes,
		&session.CreatedAt,
		&session.UpdatedAt,
	); err != nil {
		return nil, err
	}

	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &session.Metadata); err != nil {
			return nil, fmt.Errorf("decode checkout metadata for billing completion: %w", err)
		}
	}
	if session.Metadata == nil {
		session.Metadata = map[string]any{}
	}

	return &session, nil
}

func setDunningTransitionContext(ctx context.Context, tx pgx.Tx, actor, reason string, metadata map[string]any) error {
	metadataBytes, err := json.Marshal(defaultMetadata(metadata))
	if err != nil {
		return fmt.Errorf("encode dunning transition metadata: %w", err)
	}

	const query = `
SELECT
	set_config('leamout.dunning_transition_actor', $1, true),
	set_config('leamout.dunning_transition_reason', $2, true),
	set_config('leamout.dunning_transition_metadata', $3, true)`

	if _, err := tx.Exec(ctx, query, actor, reason, string(metadataBytes)); err != nil {
		return fmt.Errorf("set billing dunning transition context: %w", err)
	}
	return nil
}

func defaultMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		return map[string]any{}
	}
	return metadata
}

func isDunningRenewal(session *checkout.Session) bool {
	return session != nil && session.Mode == checkout.ModeRenewal && session.Source == checkout.SourceDunning && session.SubscriptionID != nil
}

func metadataUUID(metadata map[string]any, key string) (uuid.UUID, error) {
	raw, ok := metadata[key]
	if !ok {
		return uuid.Nil, fmt.Errorf("%w: missing %s", ErrInvalidCheckoutCompletion, key)
	}
	value, ok := raw.(string)
	if !ok || strings.TrimSpace(value) == "" {
		return uuid.Nil, fmt.Errorf("%w: invalid %s", ErrInvalidCheckoutCompletion, key)
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: invalid %s", ErrInvalidCheckoutCompletion, key)
	}
	return id, nil
}
