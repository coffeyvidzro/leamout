package entitlement

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
)

var ErrInvalidCheck = errors.New("invalid entitlement check")

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Check(ctx context.Context, userID uuid.UUID, req CheckRequest) (*CheckResponse, error) {
	params, err := normalizeCheckRequest(userID, req)
	if err != nil {
		return nil, err
	}

	candidate, err := s.repo.Check(ctx, params)
	if errors.Is(err, ErrNoActiveGrant) {
		return negativeResponse(params, "", ReasonNoActiveGrant), nil
	}
	if err != nil {
		return nil, err
	}

	return evaluate(params, candidate), nil
}

func normalizeCheckRequest(userID uuid.UUID, req CheckRequest) (CheckParams, error) {
	code := strings.ToLower(strings.TrimSpace(req.Code))
	externalID := ""
	if req.ExternalCustomerID != nil {
		externalID = strings.TrimSpace(*req.ExternalCustomerID)
	}

	hasCustomerID := req.CustomerID != nil && *req.CustomerID != uuid.Nil
	hasExternalID := externalID != ""
	if userID == uuid.Nil || code == "" || hasCustomerID == hasExternalID {
		return CheckParams{}, ErrInvalidCheck
	}

	quantity := 1.0
	if req.Quantity != nil {
		quantity = *req.Quantity
	}
	if quantity <= 0 {
		return CheckParams{}, ErrInvalidCheck
	}

	return CheckParams{
		UserID:             userID,
		CustomerID:         req.CustomerID,
		ExternalCustomerID: externalID,
		Code:               code,
		Quantity:           quantity,
	}, nil
}

func evaluate(params CheckParams, candidate *GrantCandidate) *CheckResponse {
	response := &CheckResponse{
		Allowed:    false,
		Code:       candidate.Code,
		Type:       candidate.Type,
		CustomerID: uuidPtr(candidate.CustomerID),
		BenefitID:  uuidPtr(candidate.BenefitID),
		GrantID:    uuidPtr(candidate.GrantID),
		Required:   params.Quantity,
		ExpiresAt:  candidate.EndsAt,
	}

	switch candidate.Type {
	case BenefitTypeCustom, BenefitTypeFeature:
		response.Allowed = true
		response.Reason = ReasonActiveGrant
		return response
	case BenefitTypeMeterCredit:
		response.MeterID = candidate.MeterID
		response.CustomerMeterID = candidate.CustomerMeterID
		response.Balance = candidate.Balance

		if candidate.MeterID == nil {
			response.Reason = ReasonInvalidMeterCreditGrant
			return response
		}
		if candidate.Balance == nil || *candidate.Balance < params.Quantity {
			response.Reason = ReasonInsufficientBalance
			return response
		}

		response.Allowed = true
		response.Reason = ReasonMeterBalanceAvailable
		return response
	default:
		response.Reason = ReasonNoActiveGrant
		return response
	}
}

func negativeResponse(params CheckParams, benefitType, reason string) *CheckResponse {
	return &CheckResponse{
		Allowed:  false,
		Code:     params.Code,
		Type:     benefitType,
		Required: params.Quantity,
		Reason:   reason,
	}
}

func uuidPtr(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}
