package checkout

import "testing"

func TestCanTransition(t *testing.T) {
	tests := []struct {
		from, to Status
		want     bool
	}{{Open, Completed, true}, {Open, Expired, true}, {Open, Canceled, true}, {Completed, Open, false}, {Expired, Completed, false}, {Open, Open, true}}
	for _, tt := range tests {
		if got := CanTransition(tt.from, tt.to); got != tt.want {
			t.Fatalf("CanTransition(%q,%q)=%v want %v", tt.from, tt.to, got, tt.want)
		}
	}
}

func TestValidateTransition(t *testing.T) {
	if err := ValidateTransition(Open, Completed); err != nil {
		t.Fatalf("expected allowed transition: %v", err)
	}
	if err := ValidateTransition(Completed, Open); err == nil {
		t.Fatal("expected invalid transition error")
	}
}
