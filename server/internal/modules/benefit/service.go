package benefit

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

var ErrInvalidBenefit = errors.New("invalid benefit")

var codePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, req CreateRequest) (*Benefit, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("%w: missing user id", ErrInvalidBenefit)
	}
	if err := validateCreateRequest(&req); err != nil {
		return nil, err
	}

	return s.repo.Create(ctx, userID, req)
}

func (s *Service) List(ctx context.Context, params ListParams) (*ListResponse, error) {
	if params.UserID == uuid.Nil {
		return nil, fmt.Errorf("%w: missing user id", ErrInvalidBenefit)
	}
	if params.Type != "" && !isValidType(params.Type) {
		return nil, fmt.Errorf("%w: invalid benefit type", ErrInvalidBenefit)
	}

	return s.repo.List(ctx, params)
}

func (s *Service) Get(ctx context.Context, userID, id uuid.UUID) (*Benefit, error) {
	if userID == uuid.Nil || id == uuid.Nil {
		return nil, fmt.Errorf("%w: invalid benefit id", ErrInvalidBenefit)
	}

	return s.repo.Get(ctx, userID, id)
}

func (s *Service) Update(ctx context.Context, userID, id uuid.UUID, req UpdateRequest) (*Benefit, error) {
	if userID == uuid.Nil || id == uuid.Nil {
		return nil, fmt.Errorf("%w: invalid benefit id", ErrInvalidBenefit)
	}

	current, err := s.repo.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if err := validateUpdateRequest(current.Type, &req); err != nil {
		return nil, err
	}

	return s.repo.Update(ctx, userID, id, req)
}

func (s *Service) Delete(ctx context.Context, userID, id uuid.UUID) error {
	if userID == uuid.Nil || id == uuid.Nil {
		return fmt.Errorf("%w: invalid benefit id", ErrInvalidBenefit)
	}

	return s.repo.Delete(ctx, userID, id)
}

func (s *Service) ListGrants(ctx context.Context, params ListGrantsParams) (*ListGrantsResponse, error) {
	if params.UserID == uuid.Nil || params.BenefitID == uuid.Nil {
		return nil, fmt.Errorf("%w: invalid benefit id", ErrInvalidBenefit)
	}
	if params.Status != "" && !isValidGrantStatus(params.Status) {
		return nil, fmt.Errorf("%w: invalid grant status", ErrInvalidBenefit)
	}

	return s.repo.ListGrants(ctx, params)
}

func validateCreateRequest(req *CreateRequest) error {
	req.Type = Type(strings.ToLower(strings.TrimSpace(string(req.Type))))
	if !isValidType(req.Type) {
		return fmt.Errorf("%w: invalid benefit type", ErrInvalidBenefit)
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidBenefit)
	}

	req.Code = normalizeCode(req.Code)
	if !codePattern.MatchString(req.Code) {
		return fmt.Errorf("%w: code must contain only lowercase letters, numbers, underscores, or hyphens", ErrInvalidBenefit)
	}

	if req.Properties == nil {
		req.Properties = map[string]any{}
	}
	if req.Metadata == nil {
		req.Metadata = map[string]any{}
	}

	return validateProperties(req.Type, req.Properties)
}

func validateUpdateRequest(currentType Type, req *UpdateRequest) error {
	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		req.Name = &trimmed
		if trimmed == "" {
			return fmt.Errorf("%w: name is required", ErrInvalidBenefit)
		}
	}
	if req.Code != nil {
		normalized := normalizeCode(*req.Code)
		req.Code = &normalized
		if !codePattern.MatchString(normalized) {
			return fmt.Errorf("%w: code must contain only lowercase letters, numbers, underscores, or hyphens", ErrInvalidBenefit)
		}
	}
	if req.Properties != nil {
		if err := validateProperties(currentType, req.Properties); err != nil {
			return err
		}
	}

	return nil
}

func validateProperties(typ Type, properties map[string]any) error {
	switch typ {
	case TypeCustom, TypeFeature:
		return nil
	case TypeMeterCredit:
		meterID, ok := stringProperty(properties, "meter_id")
		if !ok {
			return fmt.Errorf("%w: meter_credit benefits require properties.meter_id", ErrInvalidBenefit)
		}
		if _, err := uuid.Parse(meterID); err != nil {
			return fmt.Errorf("%w: properties.meter_id must be a valid uuid", ErrInvalidBenefit)
		}
		quantity, ok := positiveIntegerProperty(properties, "quantity")
		if !ok || quantity <= 0 {
			return fmt.Errorf("%w: meter_credit benefits require positive properties.quantity", ErrInvalidBenefit)
		}
		return nil
	default:
		return fmt.Errorf("%w: invalid benefit type", ErrInvalidBenefit)
	}
}

func isValidType(typ Type) bool {
	switch typ {
	case TypeCustom, TypeFeature, TypeMeterCredit:
		return true
	default:
		return false
	}
}

func isValidGrantStatus(status GrantStatus) bool {
	switch status {
	case GrantStatusActive, GrantStatusRevoked, GrantStatusExpired:
		return true
	default:
		return false
	}
}

func stringProperty(properties map[string]any, key string) (string, bool) {
	value, ok := properties[key]
	if !ok || value == nil {
		return "", false
	}

	str, ok := value.(string)
	if !ok {
		return "", false
	}

	str = strings.TrimSpace(str)
	return str, str != ""
}

func positiveIntegerProperty(properties map[string]any, key string) (int64, bool) {
	value, ok := properties[key]
	if !ok || value == nil {
		return 0, false
	}

	switch typed := value.(type) {
	case float64:
		if typed <= 0 || typed != float64(int64(typed)) {
			return 0, false
		}
		return int64(typed), true
	case int:
		return int64(typed), typed > 0
	case int64:
		return typed, typed > 0
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		if err != nil || parsed <= 0 {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}
