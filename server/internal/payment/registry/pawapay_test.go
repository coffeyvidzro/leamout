package registry

import "testing"

func TestPawaPayMVPRulesExcludeTieredMarkets(t *testing.T) {
	for _, rule := range PawaPayMVPRules() {
		switch rule.Country {
		case "KE", "ZM":
			t.Fatalf("tiered market %s should not be in MVP registry", rule.Country)
		}
		switch rule.CountryAlpha3 {
		case "KEN", "ZMB":
			t.Fatalf("tiered market %s should not be in MVP registry", rule.CountryAlpha3)
		}
		switch rule.Currency {
		case "KES", "ZMW":
			t.Fatalf("tiered currency %s should not be in MVP registry", rule.Currency)
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

func TestPawaPayMVPFeeRulesSkipUnconfiguredMarkets(t *testing.T) {
	for _, rule := range PawaPayMVPFeeRules() {
		if rule.Country == "MZ" {
			t.Fatal("Mozambique should be present as a market but skipped from fee rules until fee is confirmed")
		}
	}

	mz, ok := FindPawaPayMVPRule("MOZ", "MZN", "vodacom")
	if !ok {
		t.Fatal("expected Mozambique market rule")
	}
	if mz.FeeConfigured {
		t.Fatal("Mozambique fee should not be configured until confirmed")
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
