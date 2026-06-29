package subscription

import "testing"

func TestCanTransition(t *testing.T) {
	tests := []struct {
		from, to Status
		want     bool
	}{{Trialing, Active, true}, {Active, PastDue, true}, {PastDue, Active, true}, {Paused, Active, true}, {Canceled, Active, false}, {Active, Incomplete, false}}
	for _, tt := range tests {
		if got := CanTransition(tt.from, tt.to); got != tt.want {
			t.Fatalf("CanTransition(%q,%q)=%v want %v", tt.from, tt.to, got, tt.want)
		}
	}
}

func TestValidateTransition(t *testing.T) {
	if err := ValidateTransition(Active, PastDue); err != nil {
		t.Fatalf("expected allowed transition: %v", err)
	}
	if err := ValidateTransition(Canceled, Active); err == nil {
		t.Fatal("expected invalid transition error")
	}
}
