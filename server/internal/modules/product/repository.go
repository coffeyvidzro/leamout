package product

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/cuffeyvidzro/leamout/internal/modules/benefit"
	"github.com/cuffeyvidzro/leamout/internal/modules/price"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("product not found")

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, userID uuid.UUID, req CreateRequest) (*Product, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin create product: %w", err)
	}
	defer tx.Rollback(ctx)

	metadata, err := encodeJSON(defaultMetadata(req.Metadata))
	if err != nil {
		return nil, err
	}

	active := true
	if req.Active != nil {
		active = *req.Active
	}

	const query = `
INSERT INTO products (user_id, name, description, active, metadata)
VALUES ($1, $2, NULLIF($3, ''), $4, $5)
RETURNING id, user_id, name, description, active, metadata, created_at, updated_at`

	product, err := scanProduct(tx.QueryRow(
		ctx,
		query,
		userID,
		strings.TrimSpace(req.Name),
		optionalString(req.Description),
		active,
		metadata,
	))
	if err != nil {
		return nil, fmt.Errorf("create product: %w", err)
	}

	product.Prices = make([]price.Price, 0, len(req.Prices))
	for _, priceReq := range req.Prices {
		createdPrice, err := createPrice(ctx, tx, userID, product.ID, priceReq)
		if err != nil {
			return nil, fmt.Errorf("create price: %w", err)
		}
		product.Prices = append(product.Prices, *createdPrice)
	}
	product.Benefits = []benefit.Benefit{}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit create product: %w", err)
	}

	return product, nil
}

func (r *Repository) List(ctx context.Context, userID uuid.UUID) ([]Product, error) {
	const query = `
SELECT id, user_id, name, description, active, metadata, created_at, updated_at
FROM products
WHERE user_id = $1
ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()

	products := make([]Product, 0)
	for rows.Next() {
		product, err := scanProduct(rows)
		if err != nil {
			return nil, fmt.Errorf("scan product: %w", err)
		}
		product.Prices, err = r.listPrices(ctx, userID, product.ID)
		if err != nil {
			return nil, err
		}
		product.Benefits, err = r.listBenefits(ctx, userID, product.ID)
		if err != nil {
			return nil, err
		}
		products = append(products, *product)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate products: %w", err)
	}

	return products, nil
}

func (r *Repository) Get(ctx context.Context, userID, id uuid.UUID) (*Product, error) {
	const query = `
SELECT id, user_id, name, description, active, metadata, created_at, updated_at
FROM products
WHERE user_id = $1 AND id = $2`

	product, err := r.get(ctx, query, userID, id)
	if err != nil {
		return nil, err
	}

	if err := r.hydrateProduct(ctx, product); err != nil {
		return nil, err
	}

	return product, nil
}

func (r *Repository) Update(ctx context.Context, userID, id uuid.UUID, req UpdateRequest) (*Product, error) {
	query, args, err := buildUpdateQuery([]any{userID, id}, req)
	if err != nil {
		return nil, err
	}
	if query == "" {
		return r.Get(ctx, userID, id)
	}

	product, err := r.get(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	if err := r.hydrateProduct(ctx, product); err != nil {
		return nil, err
	}

	return product, nil
}

func (r *Repository) Delete(ctx context.Context, userID, id uuid.UUID) error {
	result, err := r.db.Exec(ctx, `DELETE FROM products WHERE user_id = $1 AND id = $2`, userID, id)
	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *Repository) UpdateBenefits(ctx context.Context, userID, productID uuid.UUID, benefitIDs []uuid.UUID) (*Product, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin update product benefits: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := ensureProductExists(ctx, tx, userID, productID); err != nil {
		return nil, err
	}

	benefitIDs = uniqueUUIDs(benefitIDs)
	for _, benefitID := range benefitIDs {
		if err := ensureBenefitExists(ctx, tx, userID, benefitID); err != nil {
			return nil, err
		}
	}

	if _, err := tx.Exec(ctx, `DELETE FROM product_benefits WHERE user_id = $1 AND product_id = $2`, userID, productID); err != nil {
		return nil, fmt.Errorf("clear product benefits: %w", err)
	}

	for _, benefitID := range benefitIDs {
		_, err := tx.Exec(ctx, `
INSERT INTO product_benefits (user_id, product_id, benefit_id)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, product_id, benefit_id) DO NOTHING`, userID, productID, benefitID)
		if err != nil {
			return nil, fmt.Errorf("attach product benefit: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit update product benefits: %w", err)
	}

	return r.Get(ctx, userID, productID)
}

func (r *Repository) get(ctx context.Context, query string, args ...any) (*Product, error) {
	product, err := scanProduct(r.db.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get product: %w", err)
	}

	return product, nil
}

func (r *Repository) hydrateProduct(ctx context.Context, product *Product) error {
	var err error
	product.Prices, err = r.listPrices(ctx, product.UserID, product.ID)
	if err != nil {
		return err
	}
	product.Benefits, err = r.listBenefits(ctx, product.UserID, product.ID)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) listPrices(ctx context.Context, userID, productID uuid.UUID) ([]price.Price, error) {
	const query = `
SELECT id, user_id, product_id, nickname, type, lookup_key, unit_amount, currency, interval, metadata, created_at, updated_at
FROM prices
WHERE user_id = $1 AND product_id = $2
ORDER BY created_at ASC`

	rows, err := r.db.Query(ctx, query, userID, productID)
	if err != nil {
		return nil, fmt.Errorf("list prices: %w", err)
	}
	defer rows.Close()

	prices := make([]price.Price, 0)
	for rows.Next() {
		p, err := scanPrice(rows)
		if err != nil {
			return nil, fmt.Errorf("scan price: %w", err)
		}
		prices = append(prices, *p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate prices: %w", err)
	}

	return prices, nil
}

func (r *Repository) listBenefits(ctx context.Context, userID, productID uuid.UUID) ([]benefit.Benefit, error) {
	const query = `
SELECT b.id, b.user_id, b.type, b.name, b.code, b.description, b.properties, b.metadata, b.archived_at, b.created_at, b.updated_at
FROM product_benefits pb
JOIN benefits b ON b.user_id = pb.user_id AND b.id = pb.benefit_id
WHERE pb.user_id = $1 AND pb.product_id = $2
ORDER BY pb.created_at ASC`

	rows, err := r.db.Query(ctx, query, userID, productID)
	if err != nil {
		return nil, fmt.Errorf("list benefits: %w", err)
	}
	defer rows.Close()

	benefits := make([]benefit.Benefit, 0)
	for rows.Next() {
		b, err := scanBenefit(rows)
		if err != nil {
			return nil, fmt.Errorf("scan benefit: %w", err)
		}
		benefits = append(benefits, *b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate benefits: %w", err)
	}

	return benefits, nil
}

func buildUpdateQuery(args []any, req UpdateRequest) (string, []any, error) {
	updates := make([]string, 0, 4)

	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(*req.Name))
	}
	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = NULLIF($%d, '')", len(args)+1))
		args = append(args, strings.TrimSpace(*req.Description))
	}
	if req.Active != nil {
		updates = append(updates, fmt.Sprintf("active = $%d", len(args)+1))
		args = append(args, *req.Active)
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
UPDATE products
SET %s
WHERE user_id = $1 AND id = $2
RETURNING id, user_id, name, description, active, metadata, created_at, updated_at`, strings.Join(updates, ", "))

	return query, args, nil
}

func createPrice(ctx context.Context, tx pgx.Tx, userID, productID uuid.UUID, req price.CreateRequest) (*price.Price, error) {
	metadata, err := encodeJSON(defaultMetadata(req.Metadata))
	if err != nil {
		return nil, err
	}

	const query = `
INSERT INTO prices (user_id, product_id, nickname, type, lookup_key, unit_amount, currency, interval, metadata)
VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7, NULLIF($8, ''), $9)
RETURNING id, user_id, product_id, nickname, type, lookup_key, unit_amount, currency, interval, metadata, created_at, updated_at`

	return scanPrice(tx.QueryRow(
		ctx,
		query,
		userID,
		productID,
		strings.TrimSpace(req.Nickname),
		strings.TrimSpace(req.Type),
		optionalString(req.LookupKey),
		req.UnitAmount,
		strings.ToUpper(strings.TrimSpace(req.Currency)),
		optionalString(req.Interval),
		metadata,
	))
}

func ensureProductExists(ctx context.Context, tx pgx.Tx, userID, productID uuid.UUID) error {
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM products WHERE user_id = $1 AND id = $2)`, userID, productID).Scan(&exists); err != nil {
		return fmt.Errorf("check product exists: %w", err)
	}
	if !exists {
		return ErrNotFound
	}

	return nil
}

func ensureBenefitExists(ctx context.Context, tx pgx.Tx, userID, benefitID uuid.UUID) error {
	var exists bool
	if err := tx.QueryRow(ctx, `
SELECT EXISTS(
    SELECT 1
    FROM benefits
    WHERE user_id = $1
      AND id = $2
      AND archived_at IS NULL
)`, userID, benefitID).Scan(&exists); err != nil {
		return fmt.Errorf("check benefit exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("%w: benefit %s not found", ErrInvalidProduct, benefitID)
	}

	return nil
}

func scanProduct(row pgx.Row) (*Product, error) {
	var product Product
	var metadataBytes []byte

	if err := row.Scan(
		&product.ID,
		&product.UserID,
		&product.Name,
		&product.Description,
		&product.Active,
		&metadataBytes,
		&product.CreatedAt,
		&product.UpdatedAt,
	); err != nil {
		return nil, err
	}

	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &product.Metadata); err != nil {
			return nil, fmt.Errorf("decode product metadata: %w", err)
		}
	}
	if product.Metadata == nil {
		product.Metadata = map[string]any{}
	}
	if product.Prices == nil {
		product.Prices = []price.Price{}
	}
	if product.Benefits == nil {
		product.Benefits = []benefit.Benefit{}
	}

	return &product, nil
}

func scanPrice(row pgx.Row) (*price.Price, error) {
	var p price.Price
	var metadataBytes []byte

	if err := row.Scan(
		&p.ID,
		&p.UserID,
		&p.ProductID,
		&p.Nickname,
		&p.Type,
		&p.LookupKey,
		&p.UnitAmount,
		&p.Currency,
		&p.Interval,
		&metadataBytes,
		&p.CreatedAt,
		&p.UpdatedAt,
	); err != nil {
		return nil, err
	}

	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &p.Metadata); err != nil {
			return nil, fmt.Errorf("decode price metadata: %w", err)
		}
	}
	if p.Metadata == nil {
		p.Metadata = map[string]any{}
	}

	return &p, nil
}

func scanBenefit(row pgx.Row) (*benefit.Benefit, error) {
	var b benefit.Benefit
	var typ string
	var properties []byte
	var metadata []byte

	if err := row.Scan(
		&b.ID,
		&b.UserID,
		&typ,
		&b.Name,
		&b.Code,
		&b.Description,
		&properties,
		&metadata,
		&b.ArchivedAt,
		&b.CreatedAt,
		&b.UpdatedAt,
	); err != nil {
		return nil, err
	}

	b.Type = benefit.Type(typ)
	if len(properties) > 0 {
		if err := json.Unmarshal(properties, &b.Properties); err != nil {
			return nil, fmt.Errorf("decode benefit properties: %w", err)
		}
	}
	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &b.Metadata); err != nil {
			return nil, fmt.Errorf("decode benefit metadata: %w", err)
		}
	}
	if b.Properties == nil {
		b.Properties = map[string]any{}
	}
	if b.Metadata == nil {
		b.Metadata = map[string]any{}
	}

	return &b, nil
}

func uniqueUUIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	unique := make([]uuid.UUID, 0, len(ids))

	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}

	return unique
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
