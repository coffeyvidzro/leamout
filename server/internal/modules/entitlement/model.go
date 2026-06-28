package entitlement

import (
	"time"

	"github.com/google/uuid"
)

const (
	BenefitTypeCustom      = "custom"
	BenefitTypeFeature     = "feature"
	BenefitTypeMeterCredit = "meter_credit"

	ReasonActiveGrant            = "active_grant"
	ReasonMeterBalanceAvailable  = "meter_balance_available"
	ReasonNoActiveGrant          = "no_active_grant"
	ReasonInsufficientBalance    = "insufficient_meter_balance"
	ReasonInvalidMeterCreditGrant = "invalid_meter_credit_grant"
)

type CheckRequest struct {
	CustomerID         *uuid.UUID `json:"customer_id,omitempty"`
	ExternalCustomerID *string    `json:"external_customer_id,omitempty" binding:"omitempty,max=160"`
	Code               string     `json:"code" binding:"required,min=2,max=120"`
	Quantity           *float64   `json:"quantity,omitempty" binding:"omitempty,gt=0"`
}

type CheckResponse struct {
	Allowed         bool       `json:"allowed"`
	Code            string     `json:"code"`
	Type            string     `json:"type,omitempty"`
	CustomerID      *uuid.UUID `json:"customer_id,omitempty"`
	BenefitID       *uuid.UUID `json:"benefit_id,omitempty"`
	GrantID         *uuid.UUID `json:"grant_id,omitempty"`
	MeterID         *uuid.UUID `json:"meter_id,omitempty"`
	CustomerMeterID *uuid.UUID `json:"customer_meter_id,omitempty"`
	Balance         *float64   `json:"balance,omitempty"`
	Required        float64    `json:"required"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	Reason          string     `json:"reason"`
}

type CheckParams struct {
	UserID             uuid.UUID
	CustomerID         *uuid.UUID
	ExternalCustomerID string
	Code               string
	Quantity           float64
}

type GrantCandidate struct {
	CustomerID      uuid.UUID
	GrantID         uuid.UUID
	BenefitID       uuid.UUID
	Type            string
	Code            string
	EndsAt          *time.Time
	Properties      map[string]any
	MeterID         *uuid.UUID
	CustomerMeterID *uuid.UUID
	Balance         *float64
}
