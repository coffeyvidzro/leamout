package dunning

import "testing"

func TestCanTransition(t *testing.T) {
	tests := []struct {
		from, to AttemptStatus
		want     bool
	}{{Pending, Sent, true}, {Pending, Paid, true}, {Sent, Paid, true}, {Sent, Expired, true}, {Paid, Sent, false}, {Canceled, Paid, false}, {Pending, Pending, true}}
	for _, tt := range tests {
		if got := CanTransition(tt.from, tt.to); got != tt.want {
			t.Fatalf("CanTransition(%q,%q)=%v want %v", tt.from, tt.to, got, tt.want)
		}
	}
}

func TestValidateTransition(t *testing.T) {
	if err := ValidateTransition(Sent, Paid); err != nil {
		t.Fatalf("expected allowed transition: %v", err)
	}
	if err := ValidateTransition(Paid, Sent); err == nil {
		t.Fatal("expected invalid transition error")
	}
}
