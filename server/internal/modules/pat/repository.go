package pat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("personal access token not found")

type CreateParams struct {
	Name      string
	TokenHash string
	ExpiresAt *time.Time
	Metadata  map[string]any
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, userID uuid.UUID, params CreateParams) (*Token, error) {
	metadata, err := encodeJSON(defaultMetadata(params.Metadata))
	if err != nil {
		return nil, err
	}

	const query = `
INSERT INTO personal_access_tokens (user_id, name, token_hash, expires_at, metadata)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, user_id, name, last_used_at, expires_at, revoked_at, metadata, created_at, updated_at`

	token, err := scanToken(r.db.QueryRow(ctx, query, userID, params.Name, params.TokenHash, params.ExpiresAt, metadata))
	if err != nil {
		return nil, fmt.Errorf("create personal access token: %w", err)
	}

	return token, nil
}

func (r *Repository) List(ctx context.Context, userID uuid.UUID) ([]Token, error) {
	const query = `
SELECT id, user_id, name, last_used_at, expires_at, revoked_at, metadata, created_at, updated_at
FROM personal_access_tokens
WHERE user_id = $1
ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list personal access tokens: %w", err)
	}
	defer rows.Close()

	tokens := make([]Token, 0)
	for rows.Next() {
		token, err := scanToken(rows)
		if err != nil {
			return nil, fmt.Errorf("scan personal access token: %w", err)
		}
		tokens = append(tokens, *token)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate personal access tokens: %w", err)
	}

	return tokens, nil
}

func (r *Repository) GetActiveByHash(ctx context.Context, tokenHash string) (*Token, error) {
	const query = `
UPDATE personal_access_tokens
SET last_used_at = NOW()
WHERE token_hash = $1
  AND revoked_at IS NULL
  AND (expires_at IS NULL OR expires_at > NOW())
RETURNING id, user_id, name, last_used_at, expires_at, revoked_at, metadata, created_at, updated_at`

	token, err := scanToken(r.db.QueryRow(ctx, query, tokenHash))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get personal access token: %w", err)
	}

	return token, nil
}

func (r *Repository) Revoke(ctx context.Context, userID, tokenID uuid.UUID) error {
	const query = `
UPDATE personal_access_tokens
SET revoked_at = COALESCE(revoked_at, NOW())
WHERE user_id = $1 AND id = $2 AND revoked_at IS NULL`

	tag, err := r.db.Exec(ctx, query, userID, tokenID)
	if err != nil {
		return fmt.Errorf("revoke personal access token: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func scanToken(row pgx.Row) (*Token, error) {
	var token Token
	var metadataBytes []byte
	if err := row.Scan(
		&token.ID,
		&token.UserID,
		&token.Name,
		&token.LastUsedAt,
		&token.ExpiresAt,
		&token.RevokedAt,
		&metadataBytes,
		&token.CreatedAt,
		&token.UpdatedAt,
	); err != nil {
		return nil, err
	}

	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &token.Metadata); err != nil {
			return nil, fmt.Errorf("decode personal access token metadata: %w", err)
		}
	}
	if token.Metadata == nil {
		token.Metadata = map[string]any{}
	}

	return &token, nil
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
