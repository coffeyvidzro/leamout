package registry

import "testing"

func TestPawaPayMVPRulesExcludeDeferredMarkets(t *testing.T) {
	for _, rule := range PawaPayMVPRules() {
		switch rule.Country {
		case "KE", "MZ", "ZM":
			t.Fatalf("deferred market %s should not be in MVP registry", rule.Country)
		}
		switch rule.CountryAlpha3 {
		case "KEN", "MOZ", "ZMB":
			t.Fatalf("deferred market %s should not be in MVP registry", rule.CountryAlpha3)
		}
		switch rule.Currency {
		case "KES", "MZN", "ZMW":
			t.Fatalf("deferred currency %s should not be in MVP registry", rule.Currency)
		}
	}
}

func TestFindPawaPayMVPRuleNormalizesCountryAndOperator(t *testing.T) {
	rule, ok := FindPawaPayMVPRule("GHA", "ghs", "MTN MoMo")
	if !ok {
		t.Fatal("expected Ghana MTN rule")
	}

	if rule.ProviderCode != "MTN_MOMO_GHA" {
		t.Fatalf("provider code = %q, want MTN_MOMO_GHA", rule.ProviderCode)
	}
	if rule.CollectionFeeBps != 200 {
		t.Fatalf("collection fee bps = %d, want 200", rule.CollectionFeeBps)
	}
	if rule.MinAmountMinor != 100 || rule.MaxAmountMinor != 1500000 {
		t.Fatalf("amount range = %d..%d, want 100..1500000", rule.MinAmountMinor, rule.MaxAmountMinor)
	}
}

func TestFindPawaPayMVPRuleByProviderCode(t *testing.T) {
	rule, ok := FindPawaPayMVPRuleByProviderCode("vodafone_gha")
	if !ok {
		t.Fatal("expected Telecel rule by PawaPay provider code")
	}

	if rule.Country != "GH" || rule.Operator != "telecel" {
		t.Fatalf("rule = %s/%s, want GH/telecel", rule.Country, rule.Operator)
	}
}

func TestPawaPayMVPFeeRulesAreConfigured(t *testing.T) {
	marketRules := PawaPayMVPRules()
	feeRules := PawaPayMVPFeeRules()

	if len(feeRules) != len(marketRules) {
		t.Fatalf("fee rule count = %d, want %d", len(feeRules), len(marketRules))
	}

	for _, rule := range marketRules {
		if !rule.FeeConfigured {
			t.Fatalf("market %s/%s/%s should have a configured MVP fee", rule.Country, rule.Currency, rule.Operator)
		}
	}
}

func TestPawaPayMarketRuleValidateAmount(t *testing.T) {
	rule, ok := FindPawaPayMVPRule("GH", "GHS", "at")
	if !ok {
		t.Fatal("expected Ghana AT rule")
	}

	if !rule.ValidateAmount(rule.MinAmountMinor) {
		t.Fatal("expected min amount to be valid")
	}
	if !rule.ValidateAmount(rule.MaxAmountMinor) {
		t.Fatal("expected max amount to be valid")
	}
	if rule.ValidateAmount(rule.MinAmountMinor - 1) {
		t.Fatal("expected amount below min to be invalid")
	}
	if rule.ValidateAmount(rule.MaxAmountMinor + 1) {
		t.Fatal("expected amount above max to be invalid")
	}
}
