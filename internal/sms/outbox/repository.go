package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("sms outbox message not found")

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateOrGet(ctx context.Context, params CreateParams) (*Message, bool, error) {
	metadata, err := json.Marshal(defaultMetadata(params.Metadata))
	if err != nil {
		return nil, false, fmt.Errorf("encode sms outbox metadata: %w", err)
	}

	const insert = `
INSERT INTO sms_messages (user_id, reference, destination, sender, content, country_code, provider, cost, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (user_id, reference) DO NOTHING
RETURNING id, user_id, reference, destination, sender, content, country_code, provider, cost, status, error, metadata, debited_at, sent_at, refunded_at, failed_at, created_at, updated_at`

	message, err := scanMessage(r.db.QueryRow(
		ctx,
		insert,
		params.UserID,
		params.Reference,
		params.Destination,
		params.Sender,
		params.Content,
		params.CountryCode,
		params.Provider,
		params.Cost,
		metadata,
	))
	if err == nil {
		return message, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, false, fmt.Errorf("create sms outbox message: %w", err)
	}

	message, err = r.GetByReference(ctx, params.UserID, params.Reference)
	if err != nil {
		return nil, false, err
	}

	return message, false, nil
}

func (r *Repository) GetByReference(ctx context.Context, userID uuid.UUID, reference string) (*Message, error) {
	const query = `
SELECT id, user_id, reference, destination, sender, content, country_code, provider, cost, status, error, metadata, debited_at, sent_at, refunded_at, failed_at, created_at, updated_at
FROM sms_messages
WHERE user_id = $1 AND reference = $2`

	message, err := scanMessage(r.db.QueryRow(ctx, query, userID, reference))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get sms outbox message: %w", err)
	}

	return message, nil
}

func (r *Repository) MarkDebited(ctx context.Context, id uuid.UUID) error {
	const query = `
UPDATE sms_messages
SET status = 'debited', debited_at = COALESCE(debited_at, NOW()), error = NULL
WHERE id = $1 AND status IN ('pending', 'debited')`
	return execStatusUpdate(ctx, r.db, "mark sms outbox debited", query, id)
}

func (r *Repository) MarkSent(ctx context.Context, id uuid.UUID) error {
	const query = `
UPDATE sms_messages
SET status = 'sent', sent_at = COALESCE(sent_at, NOW()), error = NULL
WHERE id = $1 AND status IN ('debited', 'sent')`
	return execStatusUpdate(ctx, r.db, "mark sms outbox sent", query, id)
}

func (r *Repository) MarkRefunded(ctx context.Context, id uuid.UUID, sendErr error) error {
	const query = `
UPDATE sms_messages
SET status = 'refunded', refunded_at = COALESCE(refunded_at, NOW()), failed_at = COALESCE(failed_at, NOW()), error = $2
WHERE id = $1 AND status IN ('pending', 'debited', 'failed', 'refunded')`
	return execStatusUpdate(ctx, r.db, "mark sms outbox refunded", query, id, errorString(sendErr))
}

func execStatusUpdate(ctx context.Context, db *pgxpool.Pool, label, query string, args ...any) error {
	result, err := db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanMessage(row pgx.Row) (*Message, error) {
	var message Message
	var metadata []byte
	if err := row.Scan(
		&message.ID,
		&message.UserID,
		&message.Reference,
		&message.Destination,
		&message.Sender,
		&message.Content,
		&message.CountryCode,
		&message.Provider,
		&message.Cost,
		&message.Status,
		&message.Error,
		&metadata,
		&message.DebitedAt,
		&message.SentAt,
		&message.RefundedAt,
		&message.FailedAt,
		&message.CreatedAt,
		&message.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &message.Metadata); err != nil {
			return nil, fmt.Errorf("decode sms outbox metadata: %w", err)
		}
	}
	if message.Metadata == nil {
		message.Metadata = map[string]any{}
	}
	return &message, nil
}

func defaultMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		return map[string]any{}
	}
	return metadata
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
