package pricing

import "testing"

func TestQuoteGrossesUpPassThroughFee(t *testing.T) {
	svc := NewDefaultService()

	quote, err := svc.Quote(Request{BaseAmount: 10000, Country: "GH", Currency: "GHS", Method: MethodMobileMoney, Operator: "mtn"})
	if err != nil {
		t.Fatalf("Quote() error = %v", err)
	}

	if quote.BaseAmount != 10000 {
		t.Fatalf("BaseAmount = %d, want 10000", quote.BaseAmount)
	}
	if quote.ProcessingFee != 205 {
		t.Fatalf("ProcessingFee = %d, want 205", quote.ProcessingFee)
	}
	if quote.PayableAmount != 10205 {
		t.Fatalf("PayableAmount = %d, want 10205", quote.PayableAmount)
	}
	if quote.FeeRateBps != 200 {
		t.Fatalf("FeeRateBps = %d, want 200", quote.FeeRateBps)
	}
	if quote.FeeMode != FeeModePassThrough {
		t.Fatalf("FeeMode = %q, want %q", quote.FeeMode, FeeModePassThrough)
	}
}

func TestQuoteRejectsUnsupportedOperator(t *testing.T) {
	svc := NewDefaultService()

	_, err := svc.Quote(Request{BaseAmount: 10000, Country: "GH", Currency: "GHS", Method: MethodMobileMoney, Operator: "orange"})
	if err == nil {
		t.Fatal("Quote() error = nil, want error")
	}
}
