package entitlement

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestEvaluateAllowsFeatureBenefit(t *testing.T) {
	customerID := uuid.New()
	benefitID := uuid.New()
	grantID := uuid.New()
	expiresAt := time.Now().UTC().Add(24 * time.Hour)

	response := evaluate(CheckParams{
		Code:     "premium_downloads",
		Quantity: 1,
	}, &GrantCandidate{
		CustomerID: customerID,
		GrantID:    grantID,
		BenefitID:  benefitID,
		Type:       BenefitTypeFeature,
		Code:       "premium_downloads",
		EndsAt:     &expiresAt,
	})

	if response == nil {
		t.Fatal("expected response")
	}
	if !response.Allowed {
		t.Fatalf("expected feature benefit to be allowed, got reason %q", response.Reason)
	}
	if response.Reason != ReasonActiveGrant {
		t.Fatalf("expected reason %q, got %q", ReasonActiveGrant, response.Reason)
	}
	if response.Type != BenefitTypeFeature {
		t.Fatalf("expected type %q, got %q", BenefitTypeFeature, response.Type)
	}
	if response.CustomerID == nil || *response.CustomerID != customerID {
		t.Fatalf("expected customer id %s, got %v", customerID, response.CustomerID)
	}
	if response.BenefitID == nil || *response.BenefitID != benefitID {
		t.Fatalf("expected benefit id %s, got %v", benefitID, response.BenefitID)
	}
	if response.GrantID == nil || *response.GrantID != grantID {
		t.Fatalf("expected grant id %s, got %v", grantID, response.GrantID)
	}
	if response.Required != 1 {
		t.Fatalf("expected required 1, got %v", response.Required)
	}
	if response.ExpiresAt == nil || !response.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("expected expires_at %s, got %v", expiresAt, response.ExpiresAt)
	}
}

func TestEvaluateAllowsCustomBenefit(t *testing.T) {
	customerID := uuid.New()
	benefitID := uuid.New()
	grantID := uuid.New()

	response := evaluate(CheckParams{
		Code:     "private_community",
		Quantity: 1,
	}, &GrantCandidate{
		CustomerID: customerID,
		GrantID:    grantID,
		BenefitID:  benefitID,
		Type:       BenefitTypeCustom,
		Code:       "private_community",
	})

	if response == nil {
		t.Fatal("expected response")
	}
	if !response.Allowed {
		t.Fatalf("expected custom benefit to be allowed, got reason %q", response.Reason)
	}
	if response.Reason != ReasonActiveGrant {
		t.Fatalf("expected reason %q, got %q", ReasonActiveGrant, response.Reason)
	}
	if response.Type != BenefitTypeCustom {
		t.Fatalf("expected type %q, got %q", BenefitTypeCustom, response.Type)
	}
	if response.CustomerID == nil || *response.CustomerID != customerID {
		t.Fatalf("expected customer id %s, got %v", customerID, response.CustomerID)
	}
	if response.BenefitID == nil || *response.BenefitID != benefitID {
		t.Fatalf("expected benefit id %s, got %v", benefitID, response.BenefitID)
	}
	if response.GrantID == nil || *response.GrantID != grantID {
		t.Fatalf("expected grant id %s, got %v", grantID, response.GrantID)
	}
}

func TestEvaluateAllowsMeterCreditBenefitWithEnoughBalance(t *testing.T) {
	customerID := uuid.New()
	benefitID := uuid.New()
	grantID := uuid.New()
	meterID := uuid.New()
	customerMeterID := uuid.New()
	balance := 9500.0

	response := evaluate(CheckParams{
		Code:     "api_calls",
		Quantity: 500,
	}, &GrantCandidate{
		CustomerID:      customerID,
		GrantID:         grantID,
		BenefitID:       benefitID,
		Type:            BenefitTypeMeterCredit,
		Code:            "api_calls",
		MeterID:         &meterID,
		CustomerMeterID: &customerMeterID,
		Balance:         &balance,
	})

	if response == nil {
		t.Fatal("expected response")
	}
	if !response.Allowed {
		t.Fatalf("expected meter credit benefit to be allowed, got reason %q", response.Reason)
	}
	if response.Reason != ReasonMeterBalanceAvailable {
		t.Fatalf("expected reason %q, got %q", ReasonMeterBalanceAvailable, response.Reason)
	}
	if response.Type != BenefitTypeMeterCredit {
		t.Fatalf("expected type %q, got %q", BenefitTypeMeterCredit, response.Type)
	}
	if response.MeterID == nil || *response.MeterID != meterID {
		t.Fatalf("expected meter id %s, got %v", meterID, response.MeterID)
	}
	if response.CustomerMeterID == nil || *response.CustomerMeterID != customerMeterID {
		t.Fatalf("expected customer meter id %s, got %v", customerMeterID, response.CustomerMeterID)
	}
	if response.Balance == nil || *response.Balance != balance {
		t.Fatalf("expected balance %v, got %v", balance, response.Balance)
	}
	if response.Required != 500 {
		t.Fatalf("expected required 500, got %v", response.Required)
	}
}

func TestEvaluateDeniesMeterCreditBenefitWithInsufficientBalance(t *testing.T) {
	customerID := uuid.New()
	benefitID := uuid.New()
	grantID := uuid.New()
	meterID := uuid.New()
	customerMeterID := uuid.New()
	balance := 100.0

	response := evaluate(CheckParams{
		Code:     "api_calls",
		Quantity: 500,
	}, &GrantCandidate{
		CustomerID:      customerID,
		GrantID:         grantID,
		BenefitID:       benefitID,
		Type:            BenefitTypeMeterCredit,
		Code:            "api_calls",
		MeterID:         &meterID,
		CustomerMeterID: &customerMeterID,
		Balance:         &balance,
	})

	if response == nil {
		t.Fatal("expected response")
	}
	if response.Allowed {
		t.Fatal("expected meter credit benefit with insufficient balance to be denied")
	}
	if response.Reason != ReasonInsufficientBalance {
		t.Fatalf("expected reason %q, got %q", ReasonInsufficientBalance, response.Reason)
	}
	if response.Balance == nil || *response.Balance != balance {
		t.Fatalf("expected balance %v, got %v", balance, response.Balance)
	}
}

func TestEvaluateDeniesInvalidMeterCreditBenefitWithoutMeter(t *testing.T) {
	customerID := uuid.New()
	benefitID := uuid.New()
	grantID := uuid.New()
	balance := 1000.0

	response := evaluate(CheckParams{
		Code:     "api_calls",
		Quantity: 1,
	}, &GrantCandidate{
		CustomerID: customerID,
		GrantID:    grantID,
		BenefitID:  benefitID,
		Type:       BenefitTypeMeterCredit,
		Code:       "api_calls",
		Balance:    &balance,
	})

	if response == nil {
		t.Fatal("expected response")
	}
	if response.Allowed {
		t.Fatal("expected meter credit benefit without meter to be denied")
	}
	if response.Reason != ReasonInvalidMeterCreditGrant {
		t.Fatalf("expected reason %q, got %q", ReasonInvalidMeterCreditGrant, response.Reason)
	}
}
