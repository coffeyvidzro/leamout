package credits

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound            = errors.New("credit balance not found")
	ErrInsufficientBalance = errors.New("insufficient communication credits")
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetBalance(ctx context.Context, userID uuid.UUID) (*Balance, error) {
	const query = `
SELECT user_id, balance, currency, created_at, updated_at
FROM credits
WHERE user_id = $1`

	balance, err := scanBalance(r.db.QueryRow(ctx, query, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get credit balance: %w", err)
	}

	return balance, nil
}

func (r *Repository) ListLedger(ctx context.Context, params ListLedgerParams) ([]LedgerEntry, error) {
	const query = `
SELECT id, user_id, type, amount, balance_after, provider, destination, reference, description, metadata, created_at
FROM credit_ledger
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, query, params.UserID, params.Limit, params.Offset)
	if err != nil {
		return nil, fmt.Errorf("list credit ledger: %w", err)
	}
	defer rows.Close()

	var entries []LedgerEntry
	for rows.Next() {
		entry, err := scanLedgerEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate credit ledger: %w", err)
	}

	return entries, nil
}

func (r *Repository) TopUp(ctx context.Context, params TopUpParams) (*Balance, error) {
	if params.Description == "" {
		params.Description = "Communication credit top-up"
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin credit top-up: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const upsert = `
INSERT INTO credits (user_id, balance, currency)
VALUES ($1, $2, 'GHS')
ON CONFLICT (user_id)
DO UPDATE SET balance = credits.balance + EXCLUDED.balance
RETURNING user_id, balance, currency, created_at, updated_at`

	balance, err := scanBalance(tx.QueryRow(ctx, upsert, params.UserID, params.Amount))
	if err != nil {
		return nil, fmt.Errorf("top up credits: %w", err)
	}
	if err := insertLedger(ctx, tx, LedgerEntry{
		UserID:       params.UserID,
		Type:         LedgerTypeTopUp,
		Amount:       params.Amount,
		BalanceAfter: balance.Balance,
		Reference:    optionalString(params.Reference),
		Description:  params.Description,
		Metadata:     params.Metadata,
	}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit credit top-up: %w", err)
	}

	return balance, nil
}

func (r *Repository) Debit(ctx context.Context, params DebitParams) (*Balance, error) {
	if params.Description == "" {
		params.Description = "Communication credit debit"
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin credit debit: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const debit = `
UPDATE credits
SET balance = balance - $2
WHERE user_id = $1
  AND currency = 'GHS'
  AND balance >= $2
RETURNING user_id, balance, currency, created_at, updated_at`

	balance, err := scanBalance(tx.QueryRow(ctx, debit, params.UserID, params.Amount))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInsufficientBalance
	}
	if err != nil {
		return nil, fmt.Errorf("debit credits: %w", err)
	}
	if err := insertLedger(ctx, tx, LedgerEntry{
		UserID:       params.UserID,
		Type:         LedgerTypeDebit,
		Amount:       -params.Amount,
		BalanceAfter: balance.Balance,
		Provider:     optionalString(params.Provider),
		Destination:  optionalString(params.Destination),
		Reference:    optionalString(params.Reference),
		Description:  params.Description,
		Metadata:     params.Metadata,
	}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit credit debit: %w", err)
	}

	return balance, nil
}

func (r *Repository) Refund(ctx context.Context, params RefundParams) (*Balance, error) {
	if params.Description == "" {
		params.Description = "Communication credit refund"
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin credit refund: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const refund = `
INSERT INTO credits (user_id, balance, currency)
VALUES ($1, $2, 'GHS')
ON CONFLICT (user_id)
DO UPDATE SET balance = credits.balance + EXCLUDED.balance
RETURNING user_id, balance, currency, created_at, updated_at`

	balance, err := scanBalance(tx.QueryRow(ctx, refund, params.UserID, params.Amount))
	if err != nil {
		return nil, fmt.Errorf("refund credits: %w", err)
	}
	if err := insertLedger(ctx, tx, LedgerEntry{
		UserID:       params.UserID,
		Type:         LedgerTypeRefund,
		Amount:       params.Amount,
		BalanceAfter: balance.Balance,
		Provider:     optionalString(params.Provider),
		Destination:  optionalString(params.Destination),
		Reference:    optionalString(params.Reference),
		Description:  params.Description,
		Metadata:     params.Metadata,
	}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit credit refund: %w", err)
	}

	return balance, nil
}

type txIface interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func insertLedger(ctx context.Context, tx txIface, entry LedgerEntry) error {
	metadata, err := json.Marshal(defaultMetadata(entry.Metadata))
	if err != nil {
		return fmt.Errorf("encode credit metadata: %w", err)
	}

	const query = `
INSERT INTO credit_ledger (
	user_id, type, amount, balance_after, provider, destination, reference, description, metadata
)
VALUES ($1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, ''), NULLIF($7, ''), $8, $9)`

	if _, err := tx.Exec(
		ctx,
		query,
		entry.UserID,
		entry.Type,
		entry.Amount,
		entry.BalanceAfter,
		optionalValue(entry.Provider),
		optionalValue(entry.Destination),
		optionalValue(entry.Reference),
		entry.Description,
		metadata,
	); err != nil {
		return fmt.Errorf("insert credit ledger: %w", err)
	}

	return nil
}

func scanBalance(row pgx.Row) (*Balance, error) {
	var balance Balance
	if err := row.Scan(
		&balance.UserID,
		&balance.Balance,
		&balance.Currency,
		&balance.CreatedAt,
		&balance.UpdatedAt,
	); err != nil {
		return nil, err
	}

	return &balance, nil
}

func scanLedgerEntry(row pgx.Row) (LedgerEntry, error) {
	var entry LedgerEntry
	var metadata []byte
	if err := row.Scan(
		&entry.ID,
		&entry.UserID,
		&entry.Type,
		&entry.Amount,
		&entry.BalanceAfter,
		&entry.Provider,
		&entry.Destination,
		&entry.Reference,
		&entry.Description,
		&metadata,
		&entry.CreatedAt,
	); err != nil {
		return LedgerEntry{}, fmt.Errorf("scan credit ledger entry: %w", err)
	}
	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &entry.Metadata); err != nil {
			return LedgerEntry{}, fmt.Errorf("decode credit ledger metadata: %w", err)
		}
	}
	if entry.Metadata == nil {
		entry.Metadata = map[string]any{}
	}

	return entry, nil
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	return &value
}

func optionalValue(value *string) string {
	if value == nil {
		return ""
	}

	return strings.TrimSpace(*value)
}

func defaultMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		return map[string]any{}
	}

	return metadata
}
