package meter

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

var ErrInvalidMeter = errors.New("invalid meter")

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, req CreateRequest) (*Meter, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("%w: missing user id", ErrInvalidMeter)
	}
	if err := validateCreateRequest(&req); err != nil {
		return nil, err
	}

	return s.repo.Create(ctx, userID, req)
}

func (s *Service) List(ctx context.Context, params ListParams) (*ListResponse, error) {
	if params.UserID == uuid.Nil {
		return nil, fmt.Errorf("%w: missing user id", ErrInvalidMeter)
	}

	return s.repo.List(ctx, params)
}

func (s *Service) Get(ctx context.Context, userID, id uuid.UUID) (*Meter, error) {
	if userID == uuid.Nil || id == uuid.Nil {
		return nil, fmt.Errorf("%w: invalid meter id", ErrInvalidMeter)
	}

	return s.repo.Get(ctx, userID, id)
}

func (s *Service) Update(ctx context.Context, userID, id uuid.UUID, req UpdateRequest) (*Meter, error) {
	if userID == uuid.Nil || id == uuid.Nil {
		return nil, fmt.Errorf("%w: invalid meter id", ErrInvalidMeter)
	}
	if err := validateUpdateRequest(&req); err != nil {
		return nil, err
	}

	return s.repo.Update(ctx, userID, id, req)
}

func (s *Service) GetQuantities(ctx context.Context, params QuantityParams) (*QuantityResponse, error) {
	if params.UserID == uuid.Nil || params.MeterID == uuid.Nil {
		return nil, fmt.Errorf("%w: invalid meter id", ErrInvalidMeter)
	}
	if params.StartTimestamp != nil && params.EndTimestamp != nil && params.StartTimestamp.After(*params.EndTimestamp) {
		return nil, fmt.Errorf("%w: start_timestamp must be before end_timestamp", ErrInvalidMeter)
	}

	return s.repo.GetQuantities(ctx, params)
}

func validateCreateRequest(req *CreateRequest) error {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidMeter)
	}
	req.Unit = normalizeUnit(req.Unit)
	if err := validateUnit(req.Unit, req.CustomLabel, req.CustomMultiplier); err != nil {
		return err
	}
	if err := validateFilter(req.EventFilter); err != nil {
		return err
	}
	if err := validateAggregation(req.Aggregation); err != nil {
		return err
	}
	if req.Metadata == nil {
		req.Metadata = map[string]any{}
	}

	return nil
}

func validateUpdateRequest(req *UpdateRequest) error {
	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		req.Name = &trimmed
		if trimmed == "" {
			return fmt.Errorf("%w: name is required", ErrInvalidMeter)
		}
	}
	if req.Unit != nil {
		unit := normalizeUnit(*req.Unit)
		req.Unit = &unit
		if err := validateUnit(unit, req.CustomLabel, req.CustomMultiplier); err != nil {
			return err
		}
	}
	if req.EventFilter != nil {
		if err := validateFilter(*req.EventFilter); err != nil {
			return err
		}
	}
	if req.Aggregation != nil {
		if err := validateAggregation(*req.Aggregation); err != nil {
			return err
		}
	}

	return nil
}

func validateUnit(unit Unit, label *string, multiplier *int) error {
	switch unit {
	case UnitScalar, UnitToken:
		if label != nil || multiplier != nil {
			return fmt.Errorf("%w: custom unit fields are only allowed when unit is custom", ErrInvalidMeter)
		}
		return nil
	case UnitCustom:
		if label == nil || strings.TrimSpace(*label) == "" {
			return fmt.Errorf("%w: custom_label is required for custom unit", ErrInvalidMeter)
		}
		if multiplier == nil || *multiplier <= 0 {
			return fmt.Errorf("%w: custom_multiplier must be greater than zero for custom unit", ErrInvalidMeter)
		}
		return nil
	default:
		return fmt.Errorf("%w: invalid unit", ErrInvalidMeter)
	}
}

func validateFilter(filter EventFilter) error {
	filter.Conjunction = strings.ToLower(strings.TrimSpace(filter.Conjunction))
	if filter.Conjunction != "and" && filter.Conjunction != "or" {
		return fmt.Errorf("%w: filter conjunction must be and or", ErrInvalidMeter)
	}
	if filter.Clauses == nil {
		return fmt.Errorf("%w: filter clauses are required", ErrInvalidMeter)
	}
	for _, clause := range filter.Clauses {
		if strings.TrimSpace(clause.Property) == "" {
			return fmt.Errorf("%w: filter property is required", ErrInvalidMeter)
		}
		if _, _, _, err := filterClauseSQL(clause); err != nil {
			return err
		}
	}

	return nil
}

func validateAggregation(aggregation Aggregation) error {
	switch aggregation.Func {
	case AggregationCount:
		if strings.TrimSpace(aggregation.Property) != "" {
			return fmt.Errorf("%w: count aggregation does not accept property", ErrInvalidMeter)
		}
	case AggregationSum, AggregationMax, AggregationMin, AggregationAvg, AggregationUnique:
		if strings.TrimSpace(aggregation.Property) == "" {
			return fmt.Errorf("%w: aggregation property is required", ErrInvalidMeter)
		}
		if _, err := eventPropertyExpression(aggregation.Property); err != nil {
			return err
		}
	default:
		return fmt.Errorf("%w: invalid aggregation function", ErrInvalidMeter)
	}

	return nil
}
