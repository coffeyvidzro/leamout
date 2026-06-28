package benefit

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

var ErrNotFound = errors.New("benefit not found")

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, userID uuid.UUID, req CreateRequest) (*Benefit, error) {
	const query = `
INSERT INTO benefits (user_id, type, name, code, description, properties, metadata)
VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7)
RETURNING id, user_id, type, name, code, description, properties, metadata, archived_at, created_at, updated_at`

	properties, err := encodeJSON(defaultObject(req.Properties))
	if err != nil {
		return nil, err
	}
	metadata, err := encodeJSON(defaultObject(req.Metadata))
	if err != nil {
		return nil, err
	}

	benefit, err := scanBenefit(r.db.QueryRow(
		ctx,
		query,
		userID,
		req.Type,
		strings.TrimSpace(req.Name),
		normalizeCode(req.Code),
		optionalString(req.Description),
		properties,
		metadata,
	))
	if err != nil {
		return nil, fmt.Errorf("create benefit: %w", err)
	}

	return benefit, nil
}

func (r *Repository) List(ctx context.Context, params ListParams) (*ListResponse, error) {
	params = normalizeListParams(params)

	where := []string{"user_id = $1"}
	args := []any{params.UserID}

	if params.Type != "" {
		args = append(args, params.Type)
		where = append(where, fmt.Sprintf("type = $%d", len(args)))
	}
	if !params.IncludeArchived {
		where = append(where, "archived_at IS NULL")
	}

	whereSQL := strings.Join(where, " AND ")

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM benefits WHERE %s`, whereSQL)
	var totalCount int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("count benefits: %w", err)
	}

	offset := (params.Page - 1) * params.Limit
	query := fmt.Sprintf(`
SELECT id, user_id, type, name, code, description, properties, metadata, archived_at, created_at, updated_at
FROM benefits
WHERE %s
ORDER BY created_at DESC
LIMIT $%d OFFSET $%d`, whereSQL, len(args)+1, len(args)+2)
	args = append(args, params.Limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list benefits: %w", err)
	}
	defer rows.Close()

	items := make([]Benefit, 0)
	for rows.Next() {
		benefit, err := scanBenefit(rows)
		if err != nil {
			return nil, fmt.Errorf("scan benefit: %w", err)
		}
		items = append(items, *benefit)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate benefits: %w", err)
	}

	return &ListResponse{
		Items:      items,
		Pagination: buildPagination(totalCount, params.Page, params.Limit),
	}, nil
}

func (r *Repository) Get(ctx context.Context, userID, id uuid.UUID) (*Benefit, error) {
	const query = `
SELECT id, user_id, type, name, code, description, properties, metadata, archived_at, created_at, updated_at
FROM benefits
WHERE user_id = $1 AND id = $2`

	benefit, err := scanBenefit(r.db.QueryRow(ctx, query, userID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get benefit: %w", err)
	}

	return benefit, nil
}

func (r *Repository) Update(ctx context.Context, userID, id uuid.UUID, req UpdateRequest) (*Benefit, error) {
	query, args, err := buildUpdateQuery(userID, id, req)
	if err != nil {
		return nil, err
	}
	if query == "" {
		return r.Get(ctx, userID, id)
	}

	benefit, err := scanBenefit(r.db.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update benefit: %w", err)
	}

	return benefit, nil
}

func (r *Repository) Delete(ctx context.Context, userID, id uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin delete benefit: %w", err)
	}
	defer tx.Rollback(ctx)

	result, err := tx.Exec(ctx, `
UPDATE benefits
SET archived_at = COALESCE(archived_at, NOW())
WHERE user_id = $1 AND id = $2`, userID, id)
	if err != nil {
		return fmt.Errorf("archive benefit: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	_, err = tx.Exec(ctx, `
UPDATE benefit_grants
SET status = 'revoked', revoked_at = COALESCE(revoked_at, NOW())
WHERE user_id = $1
  AND benefit_id = $2
  AND status = 'active'
  AND revoked_at IS NULL`, userID, id)
	if err != nil {
		return fmt.Errorf("revoke benefit grants: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit delete benefit: %w", err)
	}

	return nil
}

func (r *Repository) ListGrants(ctx context.Context, params ListGrantsParams) (*ListGrantsResponse, error) {
	params = normalizeListGrantsParams(params)

	if _, err := r.Get(ctx, params.UserID, params.BenefitID); err != nil {
		return nil, err
	}

	where := []string{"user_id = $1", "benefit_id = $2"}
	args := []any{params.UserID, params.BenefitID}

	if params.CustomerID != nil {
		args = append(args, *params.CustomerID)
		where = append(where, fmt.Sprintf("customer_id = $%d", len(args)))
	}
	if params.Status != "" {
		args = append(args, params.Status)
		where = append(where, fmt.Sprintf("status = $%d", len(args)))
	}

	whereSQL := strings.Join(where, " AND ")

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM benefit_grants WHERE %s`, whereSQL)
	var totalCount int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("count benefit grants: %w", err)
	}

	offset := (params.Page - 1) * params.Limit
	query := fmt.Sprintf(`
SELECT id, user_id, benefit_id, customer_id, product_id, subscription_id, source_type, source_id, status, starts_at, ends_at, granted_at, revoked_at, properties, metadata, created_at, updated_at
FROM benefit_grants
WHERE %s
ORDER BY created_at DESC
LIMIT $%d OFFSET $%d`, whereSQL, len(args)+1, len(args)+2)
	args = append(args, params.Limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list benefit grants: %w", err)
	}
	defer rows.Close()

	items := make([]Grant, 0)
	for rows.Next() {
		grant, err := scanGrant(rows)
		if err != nil {
			return nil, fmt.Errorf("scan benefit grant: %w", err)
		}
		items = append(items, *grant)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate benefit grants: %w", err)
	}

	return &ListGrantsResponse{
		Items:      items,
		Pagination: buildPagination(totalCount, params.Page, params.Limit),
	}, nil
}

func buildUpdateQuery(userID, id uuid.UUID, req UpdateRequest) (string, []any, error) {
	args := []any{userID, id}
	updates := make([]string, 0, 7)

	add := func(expr string, value any) {
		args = append(args, value)
		updates = append(updates, fmt.Sprintf(expr, len(args)))
	}

	if req.Name != nil {
		add("name = $%d", strings.TrimSpace(*req.Name))
	}
	if req.Code != nil {
		add("code = $%d", normalizeCode(*req.Code))
	}
	if req.Description != nil {
		add("description = NULLIF($%d, '')", strings.TrimSpace(*req.Description))
	}
	if req.Properties != nil {
		properties, err := encodeJSON(req.Properties)
		if err != nil {
			return "", nil, err
		}
		add("properties = $%d", properties)
	}
	if req.Metadata != nil {
		metadata, err := encodeJSON(req.Metadata)
		if err != nil {
			return "", nil, err
		}
		add("metadata = $%d", metadata)
	}
	if req.Archived != nil {
		if *req.Archived {
			updates = append(updates, "archived_at = COALESCE(archived_at, NOW())")
		} else {
			updates = append(updates, "archived_at = NULL")
		}
	}

	if len(updates) == 0 {
		return "", args, nil
	}

	query := fmt.Sprintf(`
UPDATE benefits
SET %s
WHERE user_id = $1 AND id = $2
RETURNING id, user_id, type, name, code, description, properties, metadata, archived_at, created_at, updated_at`, strings.Join(updates, ", "))

	return query, args, nil
}

func scanBenefit(row pgx.Row) (*Benefit, error) {
	var benefit Benefit
	var typ string
	var properties []byte
	var metadata []byte

	if err := row.Scan(
		&benefit.ID,
		&benefit.UserID,
		&typ,
		&benefit.Name,
		&benefit.Code,
		&benefit.Description,
		&properties,
		&metadata,
		&benefit.ArchivedAt,
		&benefit.CreatedAt,
		&benefit.UpdatedAt,
	); err != nil {
		return nil, err
	}

	benefit.Type = Type(typ)
	if len(properties) > 0 {
		if err := json.Unmarshal(properties, &benefit.Properties); err != nil {
			return nil, fmt.Errorf("decode benefit properties: %w", err)
		}
	}
	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &benefit.Metadata); err != nil {
			return nil, fmt.Errorf("decode benefit metadata: %w", err)
		}
	}
	if benefit.Properties == nil {
		benefit.Properties = map[string]any{}
	}
	if benefit.Metadata == nil {
		benefit.Metadata = map[string]any{}
	}

	return &benefit, nil
}

func scanGrant(row pgx.Row) (*Grant, error) {
	var grant Grant
	var sourceType string
	var status string
	var properties []byte
	var metadata []byte

	if err := row.Scan(
		&grant.ID,
		&grant.UserID,
		&grant.BenefitID,
		&grant.CustomerID,
		&grant.ProductID,
		&grant.SubscriptionID,
		&sourceType,
		&grant.SourceID,
		&status,
		&grant.StartsAt,
		&grant.EndsAt,
		&grant.GrantedAt,
		&grant.RevokedAt,
		&properties,
		&metadata,
		&grant.CreatedAt,
		&grant.UpdatedAt,
	); err != nil {
		return nil, err
	}

	grant.SourceType = GrantSourceType(sourceType)
	grant.Status = GrantStatus(status)
	if len(properties) > 0 {
		if err := json.Unmarshal(properties, &grant.Properties); err != nil {
			return nil, fmt.Errorf("decode grant properties: %w", err)
		}
	}
	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &grant.Metadata); err != nil {
			return nil, fmt.Errorf("decode grant metadata: %w", err)
		}
	}
	if grant.Properties == nil {
		grant.Properties = map[string]any{}
	}
	if grant.Metadata == nil {
		grant.Metadata = map[string]any{}
	}

	return &grant, nil
}

func normalizeListParams(params ListParams) ListParams {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.Limit <= 0 {
		params.Limit = 10
	}
	if params.Limit > 100 {
		params.Limit = 100
	}

	return params
}

func normalizeListGrantsParams(params ListGrantsParams) ListGrantsParams {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.Limit <= 0 {
		params.Limit = 10
	}
	if params.Limit > 100 {
		params.Limit = 100
	}

	return params
}

func buildPagination(totalCount, page, limit int) Pagination {
	maxPage := 0
	if totalCount > 0 {
		maxPage = (totalCount + limit - 1) / limit
	}

	return Pagination{
		TotalCount: totalCount,
		Page:       page,
		Limit:      limit,
		MaxPage:    maxPage,
	}
}

func encodeJSON(value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode json: %w", err)
	}

	return data, nil
}

func defaultObject(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}

	return value
}

func optionalString(value *string) string {
	if value == nil {
		return ""
	}

	return strings.TrimSpace(*value)
}

func normalizeCode(code string) string {
	code = strings.ToLower(strings.TrimSpace(code))
	code = strings.ReplaceAll(code, " ", "_")

	return code
}
