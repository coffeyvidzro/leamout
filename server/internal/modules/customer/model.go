package customer

import (
	"time"

	"github.com/google/uuid"
)

type Customer struct {
	ID         uuid.UUID      `json:"id"`
	UserID     uuid.UUID      `json:"user_id"`
	Name       string         `json:"name"`
	Email      *string        `json:"email,omitempty"`
	Phone      string         `json:"phone"`
	ExternalID *string        `json:"external_id,omitempty"`
	Address    Address        `json:"address"`
	Metadata   map[string]any `json:"metadata"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

type Address struct {
	Line1      string `json:"line1,omitempty"`
	Line2      string `json:"line2,omitempty"`
	City       string `json:"city,omitempty"`
	State      string `json:"state,omitempty"`
	PostalCode string `json:"postal_code,omitempty"`
	Country    string `json:"country,omitempty"`
}

type CreateRequest struct {
	Name       string         `json:"name" binding:"required,min=1,max=160"`
	Email      *string        `json:"email" binding:"omitempty,email"`
	Phone      string         `json:"phone" binding:"required,min=3,max=40"`
	ExternalID *string        `json:"external_id" binding:"omitempty,max=160"`
	Address    Address        `json:"address"`
	Metadata   map[string]any `json:"metadata"`
}

type UpdateRequest struct {
	Name       *string        `json:"name,omitempty" binding:"omitempty,min=1,max=160"`
	Email      *string        `json:"email,omitempty" binding:"omitempty,email"`
	Phone      *string        `json:"phone,omitempty" binding:"omitempty,min=3,max=40"`
	ExternalID *string        `json:"external_id,omitempty" binding:"omitempty,max=160"`
	Address    *Address       `json:"address,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type State struct {
	ID                  uuid.UUID            `json:"id"`
	UserID              uuid.UUID            `json:"user_id"`
	Name                string               `json:"name"`
	Email               *string              `json:"email,omitempty"`
	Phone               string               `json:"phone"`
	ExternalID          *string              `json:"external_id,omitempty"`
	Address             Address              `json:"address"`
	Metadata            map[string]any       `json:"metadata"`
	ActiveSubscriptions []StateSubscription  `json:"active_subscriptions"`
	GrantedBenefits     []StateBenefitGrant  `json:"granted_benefits"`
	ActiveMeters         []StateActiveMeter   `json:"active_meters"`
	CreatedAt           time.Time            `json:"created_at"`
	UpdatedAt           time.Time            `json:"updated_at"`
}

type StateSubscription struct {
	ID                  uuid.UUID      `json:"id"`
	ProductID           uuid.UUID      `json:"product_id"`
	PriceID             uuid.UUID      `json:"price_id"`
	Status              string         `json:"status"`
	Amount              int64          `json:"amount"`
	Currency            string         `json:"currency"`
	CurrentPeriodStart  time.Time      `json:"current_period_start"`
	CurrentPeriodEnd    time.Time      `json:"current_period_end"`
	CancelAtPeriodEnd   bool           `json:"cancel_at_period_end"`
	CanceledAt          *time.Time     `json:"canceled_at,omitempty"`
	EndsAt              *time.Time     `json:"ends_at,omitempty"`
	EndedAt             *time.Time     `json:"ended_at,omitempty"`
	Metadata            map[string]any `json:"metadata"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
}

type StateBenefitGrant struct {
	ID              uuid.UUID      `json:"id"`
	BenefitID       uuid.UUID      `json:"benefit_id"`
	BenefitType     string         `json:"benefit_type"`
	BenefitName     string         `json:"benefit_name"`
	BenefitCode     string         `json:"benefit_code"`
	BenefitMetadata map[string]any `json:"benefit_metadata"`
	ProductID       *uuid.UUID     `json:"product_id,omitempty"`
	SubscriptionID  *uuid.UUID     `json:"subscription_id,omitempty"`
	SourceType      string         `json:"source_type"`
	SourceID        uuid.UUID      `json:"source_id"`
	Status          string         `json:"status"`
	StartsAt        *time.Time     `json:"starts_at,omitempty"`
	EndsAt          *time.Time     `json:"ends_at,omitempty"`
	GrantedAt       time.Time      `json:"granted_at"`
	RevokedAt       *time.Time     `json:"revoked_at,omitempty"`
	Properties      map[string]any `json:"properties"`
	Metadata        map[string]any `json:"metadata"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type StateActiveMeter struct {
	ID            uuid.UUID `json:"id"`
	MeterID       uuid.UUID `json:"meter_id"`
	ConsumedUnits int64     `json:"consumed_units"`
	CreditedUnits int64     `json:"credited_units"`
	Balance       int64     `json:"balance"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
