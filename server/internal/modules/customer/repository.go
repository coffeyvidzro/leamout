package customer

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

var ErrNotFound = errors.New("customer not found")

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, userID uuid.UUID, req CreateRequest) (*Customer, error) {
	const query = `
INSERT INTO customers (user_id, name, email, phone, external_id, address, metadata)
VALUES ($1, $2, NULLIF($3, ''), $4, NULLIF($5, ''), $6, $7)
RETURNING id, user_id, name, email, phone, external_id, address, metadata, created_at, updated_at`

	address, err := encodeJSON(req.Address)
	if err != nil {
		return nil, err
	}
	metadata, err := encodeJSON(defaultMetadata(req.Metadata))
	if err != nil {
		return nil, err
	}

	customer, err := scanCustomer(r.db.QueryRow(
		ctx,
		query,
		userID,
		strings.TrimSpace(req.Name),
		optionalString(req.Email),
		strings.TrimSpace(req.Phone),
		optionalString(req.ExternalID),
		address,
		metadata,
	))
	if err != nil {
		return nil, fmt.Errorf("create customer: %w", err)
	}

	return customer, nil
}

func (r *Repository) List(ctx context.Context, userID uuid.UUID) ([]Customer, error) {
	const query = `
SELECT id, user_id, name, email, phone, external_id, address, metadata, created_at, updated_at
FROM customers
WHERE user_id = $1
ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list customers: %w", err)
	}
	defer rows.Close()

	customers := make([]Customer, 0)
	for rows.Next() {
		customer, err := scanCustomer(rows)
		if err != nil {
			return nil, fmt.Errorf("scan customer: %w", err)
		}
		customers = append(customers, *customer)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate customers: %w", err)
	}

	return customers, nil
}

func (r *Repository) Get(ctx context.Context, userID, id uuid.UUID) (*Customer, error) {
	const query = `
SELECT id, user_id, name, email, phone, external_id, address, metadata, created_at, updated_at
FROM customers
WHERE user_id = $1 AND id = $2`

	return r.get(ctx, query, userID, id)
}

func (r *Repository) GetByExternalID(ctx context.Context, userID uuid.UUID, externalID string) (*Customer, error) {
	const query = `
SELECT id, user_id, name, email, phone, external_id, address, metadata, created_at, updated_at
FROM customers
WHERE user_id = $1 AND external_id = $2`

	return r.get(ctx, query, userID, strings.TrimSpace(externalID))
}

func (r *Repository) Update(ctx context.Context, userID, id uuid.UUID, req UpdateRequest) (*Customer, error) {
	query, args, err := buildUpdateQuery("user_id = $1 AND id = $2", []any{userID, id}, req)
	if err != nil {
		return nil, err
	}
	if query == "" {
		return r.Get(ctx, userID, id)
	}

	return r.get(ctx, query, args...)
}

func (r *Repository) UpdateByExternalID(ctx context.Context, userID uuid.UUID, externalID string, req UpdateRequest) (*Customer, error) {
	query, args, err := buildUpdateQuery("user_id = $1 AND external_id = $2", []any{userID, strings.TrimSpace(externalID)}, req)
	if err != nil {
		return nil, err
	}
	if query == "" {
		return r.GetByExternalID(ctx, userID, externalID)
	}

	return r.get(ctx, query, args...)
}

func (r *Repository) Delete(ctx context.Context, userID, id uuid.UUID) error {
	const query = `DELETE FROM customers WHERE user_id = $1 AND id = $2`
	return r.delete(ctx, query, userID, id)
}

func (r *Repository) DeleteByExternalID(ctx context.Context, userID uuid.UUID, externalID string) error {
	const query = `DELETE FROM customers WHERE user_id = $1 AND external_id = $2`
	return r.delete(ctx, query, userID, strings.TrimSpace(externalID))
}

func (r *Repository) get(ctx context.Context, query string, args ...any) (*Customer, error) {
	customer, err := scanCustomer(r.db.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get customer: %w", err)
	}

	return customer, nil
}

func (r *Repository) delete(ctx context.Context, query string, args ...any) error {
	result, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete customer: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func buildUpdateQuery(where string, args []any, req UpdateRequest) (string, []any, error) {
	updates := make([]string, 0, 6)

	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(*req.Name))
	}
	if req.Email != nil {
		updates = append(updates, fmt.Sprintf("email = NULLIF($%d, '')", len(args)+1))
		args = append(args, strings.TrimSpace(*req.Email))
	}
	if req.Phone != nil {
		updates = append(updates, fmt.Sprintf("phone = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(*req.Phone))
	}
	if req.ExternalID != nil {
		updates = append(updates, fmt.Sprintf("external_id = NULLIF($%d, '')", len(args)+1))
		args = append(args, strings.TrimSpace(*req.ExternalID))
	}
	if req.Address != nil {
		address, err := encodeJSON(*req.Address)
		if err != nil {
			return "", nil, err
		}
		updates = append(updates, fmt.Sprintf("address = $%d", len(args)+1))
		args = append(args, address)
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
UPDATE customers
SET %s
WHERE %s
RETURNING id, user_id, name, email, phone, external_id, address, metadata, created_at, updated_at`, strings.Join(updates, ", "), where)

	return query, args, nil
}

func scanCustomer(row pgx.Row) (*Customer, error) {
	var customer Customer
	var addressBytes []byte
	var metadataBytes []byte

	if err := row.Scan(
		&customer.ID,
		&customer.UserID,
		&customer.Name,
		&customer.Email,
		&customer.Phone,
		&customer.ExternalID,
		&addressBytes,
		&metadataBytes,
		&customer.CreatedAt,
		&customer.UpdatedAt,
	); err != nil {
		return nil, err
	}

	if len(addressBytes) > 0 {
		if err := json.Unmarshal(addressBytes, &customer.Address); err != nil {
			return nil, fmt.Errorf("decode customer address: %w", err)
		}
	}
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &customer.Metadata); err != nil {
			return nil, fmt.Errorf("decode customer metadata: %w", err)
		}
	}
	if customer.Metadata == nil {
		customer.Metadata = map[string]any{}
	}

	return &customer, nil
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
