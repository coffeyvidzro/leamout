package payment

import "testing"

func TestCanTransition(t *testing.T) {
	tests := []struct {
		from, to Status
		want     bool
	}{{Pending, Authorized, true}, {Pending, Captured, true}, {Authorized, Captured, true}, {Captured, Refunded, true}, {Captured, Failed, false}, {Failed, Captured, false}, {Pending, Pending, true}, {Status("unknown"), Captured, false}}
	for _, tt := range tests {
		if got := CanTransition(tt.from, tt.to); got != tt.want {
			t.Fatalf("CanTransition(%q,%q)=%v want %v", tt.from, tt.to, got, tt.want)
		}
	}
}

func TestValidateTransition(t *testing.T) {
	if err := ValidateTransition(Pending, Captured); err != nil {
		t.Fatalf("expected allowed transition: %v", err)
	}
	if err := ValidateTransition(Captured, Failed); err == nil {
		t.Fatal("expected invalid transition error")
	}
}
