package checkout

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound          = errors.New("checkout session not found")
	ErrRenewalUnmatched  = errors.New("checkout session is not a dunning renewal")
	ErrRenewalIncomplete = errors.New("failed to complete dunning renewal")
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, userID uuid.UUID, req CreateRequest, clientSecretHash string) (*Session, error) {
	metadata, err := encodeJSON(defaultMetadata(req.Metadata))
	if err != nil {
		return nil, err
	}

	mode := req.Mode
	if mode == "" {
		mode = ModePayment
	}
	source := req.Source
	if source == "" {
		source = SourceAPI
	}

	const query = `
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
	success_url,
	return_url,
	expires_at,
	metadata
)
VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7, $8, $9, NULLIF($10, ''), NULLIF($11, ''), $12, $13)
RETURNING id, user_id, customer_id, subscription_id, mode, source, label, amount, currency,
	client_secret_hash, success_url, return_url, status, expires_at, completed_at, canceled_at,
	metadata, created_at, updated_at`

	session, err := scanSession(r.db.QueryRow(
		ctx,
		query,
		userID,
		req.CustomerID,
		req.SubscriptionID,
		mode,
		source,
		optionalString(req.Label),
		req.Amount,
		strings.ToUpper(strings.TrimSpace(req.Currency)),
		clientSecretHash,
		optionalString(req.SuccessURL),
		optionalString(req.ReturnURL),
		req.ExpiresAt,
		metadata,
	))
	if err != nil {
		return nil, fmt.Errorf("create checkout session: %w", err)
	}

	return session, nil
}

func (r *Repository) List(ctx context.Context, userID uuid.UUID) ([]Session, error) {
	const query = `
SELECT id, user_id, customer_id, subscription_id, mode, source, label, amount, currency,
	client_secret_hash, success_url, return_url, status, expires_at, completed_at, canceled_at,
	metadata, created_at, updated_at
FROM checkout_sessions
WHERE user_id = $1
ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list checkout sessions: %w", err)
	}
	defer rows.Close()

	sessions := make([]Session, 0)
	for rows.Next() {
		session, err := scanSession(rows)
		if err != nil {
			return nil, fmt.Errorf("scan checkout session: %w", err)
		}
		sessions = append(sessions, *session)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate checkout sessions: %w", err)
	}

	return sessions, nil
}

func (r *Repository) Get(ctx context.Context, userID, id uuid.UUID) (*Session, error) {
	const query = `
SELECT id, user_id, customer_id, subscription_id, mode, source, label, amount, currency,
	client_secret_hash, success_url, return_url, status, expires_at, completed_at, canceled_at,
	metadata, created_at, updated_at
FROM checkout_sessions
WHERE user_id = $1 AND id = $2`

	return r.get(ctx, query, userID, id)
}

func (r *Repository) GetByClientSecretHash(ctx context.Context, clientSecretHash string) (*Session, error) {
	const query = `
SELECT id, user_id, customer_id, subscription_id, mode, source, label, amount, currency,
	client_secret_hash, success_url, return_url, status, expires_at, completed_at, canceled_at,
	metadata, created_at, updated_at
FROM checkout_sessions
WHERE client_secret_hash = $1
  AND status = 'open'
  AND expires_at > NOW()`

	return r.get(ctx, query, clientSecretHash)
}

func (r *Repository) ConfirmByClientSecretHash(ctx context.Context, clientSecretHash string) (*Session, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin checkout confirmation: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	session, err := r.getForUpdate(ctx, tx, clientSecretHash)
	if err != nil {
		return nil, err
	}

	if session.Status != StatusOpen {
		return nil, ErrNotFound
	}

	session, err = r.completeSession(ctx, tx, session.ID)
	if err != nil {
		return nil, err
	}

	if isDunningRenewal(session) {
		if err := r.completeDunningRenewal(ctx, tx, session); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit checkout confirmation: %w", err)
	}

	return session, nil
}

func (r *Repository) Update(ctx context.Context, userID, id uuid.UUID, req UpdateRequest) (*Session, error) {
	query, args, err := buildUpdateQuery([]any{userID, id}, req)
	if err != nil {
		return nil, err
	}
	if query == "" {
		return r.Get(ctx, userID, id)
	}

	return r.get(ctx, query, args...)
}

func (r *Repository) get(ctx context.Context, query string, args ...any) (*Session, error) {
	session, err := scanSession(r.db.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get checkout session: %w", err)
	}

	return session, nil
}

func (r *Repository) getForUpdate(ctx context.Context, tx pgx.Tx, clientSecretHash string) (*Session, error) {
	const query = `
SELECT id, user_id, customer_id, subscription_id, mode, source, label, amount, currency,
	client_secret_hash, success_url, return_url, status, expires_at, completed_at, canceled_at,
	metadata, created_at, updated_at
FROM checkout_sessions
WHERE client_secret_hash = $1
  AND status = 'open'
  AND expires_at > NOW()
FOR UPDATE`

	session, err := scanSession(tx.QueryRow(ctx, query, clientSecretHash))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock checkout session: %w", err)
	}

	return session, nil
}

func (r *Repository) completeSession(ctx context.Context, tx pgx.Tx, sessionID uuid.UUID) (*Session, error) {
	const query = `
UPDATE checkout_sessions
SET status = 'completed', completed_at = COALESCE(completed_at, NOW())
WHERE id = $1
  AND status = 'open'
RETURNING id, user_id, customer_id, subscription_id, mode, source, label, amount, currency,
	client_secret_hash, success_url, return_url, status, expires_at, completed_at, canceled_at,
	metadata, created_at, updated_at`

	session, err := scanSession(tx.QueryRow(ctx, query, sessionID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("complete checkout session: %w", err)
	}

	return session, nil
}

func (r *Repository) completeDunningRenewal(ctx context.Context, tx pgx.Tx, session *Session) error {
	if !isDunningRenewal(session) {
		return ErrRenewalUnmatched
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
		return fmt.Errorf("extend dunning subscription: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrRenewalIncomplete
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
		return fmt.Errorf("mark dunning attempt paid: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrRenewalIncomplete
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
		return fmt.Errorf("revoke dunning token: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrRenewalIncomplete
	}

	return nil
}

func (r *Repository) CompletePaidCheckout(ctx context.Context, checkoutID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin paid checkout completion: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	session, err := r.getSessionByIDForUpdate(ctx, tx, checkoutID)
	if err != nil {
		return err
	}
	if session.Status == StatusCompleted {
		return tx.Commit(ctx)
	}
	if session.Status != StatusOpen {
		return tx.Commit(ctx)
	}

	session, err = r.completeSession(ctx, tx, checkoutID)
	if err != nil {
		return err
	}
	if isDunningRenewal(session) {
		if err := r.completeDunningRenewal(ctx, tx, session); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit paid checkout completion: %w", err)
	}
	return nil
}

func (r *Repository) getSessionByIDForUpdate(ctx context.Context, tx pgx.Tx, sessionID uuid.UUID) (*Session, error) {
	const query = `
SELECT id, user_id, customer_id, subscription_id, mode, source, label, amount, currency,
	client_secret_hash, success_url, return_url, status, expires_at, completed_at, canceled_at,
	metadata, created_at, updated_at
FROM checkout_sessions
WHERE id = $1
FOR UPDATE`

	session, err := scanSession(tx.QueryRow(ctx, query, sessionID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock checkout session by id: %w", err)
	}
	return session, nil
}

func isDunningRenewal(session *Session) bool {
	return session.Mode == ModeRenewal && session.Source == SourceDunning && session.SubscriptionID != nil
}

func metadataUUID(metadata map[string]any, key string) (uuid.UUID, error) {
	raw, ok := metadata[key]
	if !ok {
		return uuid.Nil, fmt.Errorf("%w: missing %s", ErrRenewalIncomplete, key)
	}

	value, ok := raw.(string)
	if !ok || strings.TrimSpace(value) == "" {
		return uuid.Nil, fmt.Errorf("%w: invalid %s", ErrRenewalIncomplete, key)
	}

	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: invalid %s", ErrRenewalIncomplete, key)
	}

	return id, nil
}

func buildUpdateQuery(args []any, req UpdateRequest) (string, []any, error) {
	updates := make([]string, 0, 6)

	if req.Status != nil {
		updates = append(updates, fmt.Sprintf("status = $%d", len(args)+1))
		args = append(args, *req.Status)
		if *req.Status == StatusCompleted {
			updates = append(updates, "completed_at = COALESCE(completed_at, NOW())")
		}
		if *req.Status == StatusCanceled {
			updates = append(updates, "canceled_at = COALESCE(canceled_at, NOW())")
		}
	}
	if req.Label != nil {
		updates = append(updates, fmt.Sprintf("label = NULLIF($%d, '')", len(args)+1))
		args = append(args, strings.TrimSpace(*req.Label))
	}
	if req.SuccessURL != nil {
		updates = append(updates, fmt.Sprintf("success_url = NULLIF($%d, '')", len(args)+1))
		args = append(args, strings.TrimSpace(*req.SuccessURL))
	}
	if req.ReturnURL != nil {
		updates = append(updates, fmt.Sprintf("return_url = NULLIF($%d, '')", len(args)+1))
		args = append(args, strings.TrimSpace(*req.ReturnURL))
	}
	if req.ExpiresAt != nil {
		updates = append(updates, fmt.Sprintf("expires_at = $%d", len(args)+1))
		args = append(args, *req.ExpiresAt)
	}
	if req.CanceledAt != nil {
		updates = append(updates, fmt.Sprintf("canceled_at = $%d", len(args)+1))
		args = append(args, *req.CanceledAt)
	}
	if req.Metadata != nil {
		metadata, err := encodeJSON(req.Metadata)
		if err != nil {
			return "", nil, err
		}
		updates = append(updates, fmt.Sprintf("metadata = $%d", len(args)+1))
		args = append(args, metadata)
	}
	if len(updates) == 0 {
		return "", args, nil
	}

	query := fmt.Sprintf(`
UPDATE checkout_sessions
SET %s
WHERE user_id = $1 AND id = $2
RETURNING id, user_id, customer_id, subscription_id, mode, source, label, amount, currency,
	client_secret_hash, success_url, return_url, status, expires_at, completed_at, canceled_at,
	metadata, created_at, updated_at`, strings.Join(updates, ", "))

	return query, args, nil
}

func scanSession(row pgx.Row) (*Session, error) {
	var session Session
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
			return nil, fmt.Errorf("decode checkout session metadata: %w", err)
		}
	}
	if session.Metadata == nil {
		session.Metadata = map[string]any{}
	}

	return &session, nil
}

func encodeJSON(value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode json: %w", err)
	}

	return data, nil
}

func defaultMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		return map[string]any{}
	}

	return metadata
}

func optionalString(value *string) string {
	if value == nil {
		return ""
	}

	return strings.TrimSpace(*value)
}
