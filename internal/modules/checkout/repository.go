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

var ErrNotFound = errors.New("checkout session not found")

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
  AND expires_at > NOW()`

	return r.get(ctx, query, clientSecretHash)
}

func (r *Repository) ConfirmByClientSecretHash(ctx context.Context, clientSecretHash string) (*Session, error) {
	const query = `
UPDATE checkout_sessions
SET status = 'completed', completed_at = COALESCE(completed_at, NOW())
WHERE client_secret_hash = $1
  AND status = 'open'
  AND expires_at > NOW()
RETURNING id, user_id, customer_id, subscription_id, mode, source, label, amount, currency,
	client_secret_hash, success_url, return_url, status, expires_at, completed_at, canceled_at,
	metadata, created_at, updated_at`

	return r.get(ctx, query, clientSecretHash)
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
