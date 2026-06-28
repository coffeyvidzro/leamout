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

func (r *Repository) GetState(ctx context.Context, userID, id uuid.UUID) (*State, error) {
	customer, err := r.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}

	return r.buildState(ctx, customer)
}

func (r *Repository) GetStateByExternalID(ctx context.Context, userID uuid.UUID, externalID string) (*State, error) {
	customer, err := r.GetByExternalID(ctx, userID, externalID)
	if err != nil {
		return nil, err
	}

	return r.buildState(ctx, customer)
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

func (r *Repository) buildState(ctx context.Context, customer *Customer) (*State, error) {
	state := &State{
		ID:                  customer.ID,
		UserID:              customer.UserID,
		Name:                customer.Name,
		Email:               customer.Email,
		Phone:               customer.Phone,
		ExternalID:          customer.ExternalID,
		Address:             customer.Address,
		Metadata:            customer.Metadata,
		ActiveSubscriptions: []StateSubscription{},
		GrantedBenefits:     []StateBenefitGrant{},
		ActiveMeters:         []StateActiveMeter{},
		CreatedAt:           customer.CreatedAt,
		UpdatedAt:           customer.UpdatedAt,
	}

	var err error
	state.ActiveSubscriptions, err = r.listActiveSubscriptions(ctx, customer.UserID, customer.ID)
	if err != nil {
		return nil, err
	}
	state.GrantedBenefits, err = r.listGrantedBenefits(ctx, customer.UserID, customer.ID)
	if err != nil {
		return nil, err
	}

	return state, nil
}

func (r *Repository) listActiveSubscriptions(ctx context.Context, userID, customerID uuid.UUID) ([]StateSubscription, error) {
	const query = `
SELECT
	s.id,
	p.product_id,
	s.price_id,
	s.status,
	p.unit_amount,
	p.currency,
	s.current_period_start,
	s.current_period_end,
	s.cancel_at_period_end,
	s.canceled_at,
	s.ends_at,
	s.ended_at,
	s.metadata,
	s.created_at,
	s.updated_at
FROM subscriptions s
JOIN prices p
  ON p.user_id = s.user_id
 AND p.id = s.price_id
WHERE s.user_id = $1
  AND s.customer_id = $2
  AND s.status = 'active'
  AND s.current_period_end > NOW()
ORDER BY s.created_at DESC`

	rows, err := r.db.Query(ctx, query, userID, customerID)
	if err != nil {
		return nil, fmt.Errorf("list customer active subscriptions: %w", err)
	}
	defer rows.Close()

	items := make([]StateSubscription, 0)
	for rows.Next() {
		subscription, err := scanStateSubscription(rows)
		if err != nil {
			return nil, fmt.Errorf("scan customer active subscription: %w", err)
		}
		items = append(items, *subscription)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate customer active subscriptions: %w", err)
	}

	return items, nil
}

func (r *Repository) listGrantedBenefits(ctx context.Context, userID, customerID uuid.UUID) ([]StateBenefitGrant, error) {
	const query = `
SELECT
	bg.id,
	bg.benefit_id,
	b.type,
	b.name,
	b.code,
	b.metadata,
	bg.product_id,
	bg.subscription_id,
	bg.source_type,
	bg.source_id,
	bg.status,
	bg.starts_at,
	bg.ends_at,
	bg.granted_at,
	bg.revoked_at,
	bg.properties,
	bg.metadata,
	bg.created_at,
	bg.updated_at
FROM benefit_grants bg
JOIN benefits b
  ON b.user_id = bg.user_id
 AND b.id = bg.benefit_id
WHERE bg.user_id = $1
  AND bg.customer_id = $2
  AND bg.status = 'active'
  AND bg.revoked_at IS NULL
  AND (bg.starts_at IS NULL OR bg.starts_at <= NOW())
  AND (bg.ends_at IS NULL OR bg.ends_at > NOW())
  AND b.archived_at IS NULL
ORDER BY bg.granted_at DESC`

	rows, err := r.db.Query(ctx, query, userID, customerID)
	if err != nil {
		return nil, fmt.Errorf("list customer granted benefits: %w", err)
	}
	defer rows.Close()

	items := make([]StateBenefitGrant, 0)
	for rows.Next() {
		grant, err := scanStateBenefitGrant(rows)
		if err != nil {
			return nil, fmt.Errorf("scan customer granted benefit: %w", err)
		}
		items = append(items, *grant)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate customer granted benefits: %w", err)
	}

	return items, nil
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

func scanStateSubscription(row pgx.Row) (*StateSubscription, error) {
	var subscription StateSubscription
	var metadataBytes []byte

	if err := row.Scan(
		&subscription.ID,
		&subscription.ProductID,
		&subscription.PriceID,
		&subscription.Status,
		&subscription.Amount,
		&subscription.Currency,
		&subscription.CurrentPeriodStart,
		&subscription.CurrentPeriodEnd,
		&subscription.CancelAtPeriodEnd,
		&subscription.CanceledAt,
		&subscription.EndsAt,
		&subscription.EndedAt,
		&metadataBytes,
		&subscription.CreatedAt,
		&subscription.UpdatedAt,
	); err != nil {
		return nil, err
	}

	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &subscription.Metadata); err != nil {
			return nil, fmt.Errorf("decode subscription metadata: %w", err)
		}
	}
	if subscription.Metadata == nil {
		subscription.Metadata = map[string]any{}
	}

	return &subscription, nil
}

func scanStateBenefitGrant(row pgx.Row) (*StateBenefitGrant, error) {
	var grant StateBenefitGrant
	var benefitMetadataBytes []byte
	var propertiesBytes []byte
	var metadataBytes []byte

	if err := row.Scan(
		&grant.ID,
		&grant.BenefitID,
		&grant.BenefitType,
		&grant.BenefitName,
		&grant.BenefitCode,
		&benefitMetadataBytes,
		&grant.ProductID,
		&grant.SubscriptionID,
		&grant.SourceType,
		&grant.SourceID,
		&grant.Status,
		&grant.StartsAt,
		&grant.EndsAt,
		&grant.GrantedAt,
		&grant.RevokedAt,
		&propertiesBytes,
		&metadataBytes,
		&grant.CreatedAt,
		&grant.UpdatedAt,
	); err != nil {
		return nil, err
	}

	if len(benefitMetadataBytes) > 0 {
		if err := json.Unmarshal(benefitMetadataBytes, &grant.BenefitMetadata); err != nil {
			return nil, fmt.Errorf("decode benefit metadata: %w", err)
		}
	}
	if len(propertiesBytes) > 0 {
		if err := json.Unmarshal(propertiesBytes, &grant.Properties); err != nil {
			return nil, fmt.Errorf("decode grant properties: %w", err)
		}
	}
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &grant.Metadata); err != nil {
			return nil, fmt.Errorf("decode grant metadata: %w", err)
		}
	}
	if grant.BenefitMetadata == nil {
		grant.BenefitMetadata = map[string]any{}
	}
	if grant.Properties == nil {
		grant.Properties = map[string]any{}
	}
	if grant.Metadata == nil {
		grant.Metadata = map[string]any{}
	}

	return &grant, nil
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
