package wallet

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

var ErrNotFound = errors.New("wallet not found")

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) List(ctx context.Context, params ListWalletsParams) ([]Wallet, error) {
	query := `
SELECT id, user_id, country, currency, pending_balance, available_balance, created_at, updated_at
FROM wallets
WHERE user_id = $1`
	args := []any{params.UserID}
	if strings.TrimSpace(params.Country) != "" {
		query += fmt.Sprintf(" AND country = $%d", len(args)+1)
		args = append(args, normalizeCountry(params.Country))
	}
	if strings.TrimSpace(params.Currency) != "" {
		query += fmt.Sprintf(" AND currency = $%d", len(args)+1)
		args = append(args, normalizeCurrency(params.Currency))
	}
	query += " ORDER BY country ASC, currency ASC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list wallets: %w", err)
	}
	defer rows.Close()

	items := make([]Wallet, 0)
	for rows.Next() {
		item, err := scanWallet(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *Repository) CreditPaymentCapture(ctx context.Context, params CreditPaymentCaptureParams) error {
	if params.Amount <= 0 {
		return fmt.Errorf("wallet credit amount must be positive")
	}

	metadata, err := encodeMetadata(params.Metadata)
	if err != nil {
		return err
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin wallet credit: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	walletRecord, err := r.getOrCreateForUpdate(ctx, tx, params.UserID, params.Country, params.Currency)
	if err != nil {
		return err
	}

	balanceAfter := walletRecord.AvailableBalance + params.Amount
	const insertLedger = `
INSERT INTO wallet_ledger_entries (
	wallet_id, user_id, payment_id, transaction_id, direction, balance_type, reason, country, currency, amount, balance_after, metadata
)
VALUES ($1, $2, $3, $4, 'credit', 'available', 'payment_captured', $5, $6, $7, $8, $9)
ON CONFLICT (wallet_id, transaction_id, reason) WHERE transaction_id IS NOT NULL DO NOTHING
RETURNING id`

	var ledgerID uuid.UUID
	err = tx.QueryRow(ctx, insertLedger, walletRecord.ID, params.UserID, params.PaymentID, params.TransactionID, normalizeCountry(params.Country), normalizeCurrency(params.Currency), params.Amount, balanceAfter, metadata).Scan(&ledgerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return tx.Commit(ctx)
	}
	if err != nil {
		return fmt.Errorf("insert wallet ledger entry: %w", err)
	}

	const updateWallet = `
UPDATE wallets
SET available_balance = available_balance + $3
WHERE id = $1 AND user_id = $2`
	if _, err := tx.Exec(ctx, updateWallet, walletRecord.ID, params.UserID, params.Amount); err != nil {
		return fmt.Errorf("update wallet balance: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit wallet credit: %w", err)
	}
	return nil
}

func (r *Repository) ListLedger(ctx context.Context, params ListLedgerParams) ([]LedgerEntry, error) {
	if params.Limit <= 0 || params.Limit > 100 {
		params.Limit = 100
	}
	if params.Offset < 0 {
		params.Offset = 0
	}

	query := `
SELECT le.id, le.wallet_id, le.user_id, le.payment_id, le.transaction_id, le.direction, le.balance_type,
	le.reason, le.country, le.currency, le.amount, le.balance_after, le.metadata, le.created_at
FROM wallet_ledger_entries le
JOIN wallets w ON w.id = le.wallet_id
WHERE le.user_id = $1`
	args := []any{params.UserID}
	if strings.TrimSpace(params.Country) != "" {
		query += fmt.Sprintf(" AND le.country = $%d", len(args)+1)
		args = append(args, normalizeCountry(params.Country))
	}
	if strings.TrimSpace(params.Currency) != "" {
		query += fmt.Sprintf(" AND le.currency = $%d", len(args)+1)
		args = append(args, normalizeCurrency(params.Currency))
	}
	query += fmt.Sprintf(" ORDER BY le.created_at DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, params.Limit, params.Offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list wallet ledger: %w", err)
	}
	defer rows.Close()

	items := make([]LedgerEntry, 0)
	for rows.Next() {
		item, err := scanLedgerEntry(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *Repository) getOrCreateForUpdate(ctx context.Context, tx pgx.Tx, userID uuid.UUID, country, currency string) (*Wallet, error) {
	country = normalizeCountry(country)
	currency = normalizeCurrency(currency)
	const insertWallet = `
INSERT INTO wallets (user_id, country, currency)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, country, currency) DO NOTHING`
	if _, err := tx.Exec(ctx, insertWallet, userID, country, currency); err != nil {
		return nil, fmt.Errorf("ensure wallet: %w", err)
	}

	const selectWallet = `
SELECT id, user_id, country, currency, pending_balance, available_balance, created_at, updated_at
FROM wallets
WHERE user_id = $1 AND country = $2 AND currency = $3
FOR UPDATE`
	walletRecord, err := scanWallet(tx.QueryRow(ctx, selectWallet, userID, country, currency))
	if err != nil {
		return nil, fmt.Errorf("lock wallet: %w", err)
	}
	return walletRecord, nil
}

func scanWallet(row pgx.Row) (*Wallet, error) {
	var item Wallet
	if err := row.Scan(&item.ID, &item.UserID, &item.Country, &item.Currency, &item.PendingBalance, &item.AvailableBalance, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, err
	}
	return &item, nil
}

func scanLedgerEntry(row pgx.Row) (*LedgerEntry, error) {
	var item LedgerEntry
	var metadata []byte
	if err := row.Scan(&item.ID, &item.WalletID, &item.UserID, &item.PaymentID, &item.TransactionID, &item.Direction, &item.BalanceType, &item.Reason, &item.Country, &item.Currency, &item.Amount, &item.BalanceAfter, &metadata, &item.CreatedAt); err != nil {
		return nil, err
	}
	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &item.Metadata); err != nil {
			return nil, err
		}
	}
	if item.Metadata == nil {
		item.Metadata = map[string]any{}
	}
	return &item, nil
}

func encodeMetadata(value map[string]any) ([]byte, error) {
	if value == nil {
		value = map[string]any{}
	}
	return json.Marshal(value)
}

func normalizeCountry(country string) string {
	return strings.ToUpper(strings.TrimSpace(country))
}

func normalizeCurrency(currency string) string {
	return strings.ToUpper(strings.TrimSpace(currency))
}
