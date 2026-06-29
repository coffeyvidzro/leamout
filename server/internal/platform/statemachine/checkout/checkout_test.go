package checkout

import "testing"

func TestCanTransition(t *testing.T) {
	tests := []struct {
		from, to Status
		want     bool
	}{{Open, Completed, true}, {Open, Expired, true}, {Completed, Open, false}, {Canceled, Completed, false}, {Open, Open, true}}
	for _, tt := range tests {
		if got := CanTransition(tt.from, tt.to); got != tt.want {
			t.Fatalf("CanTransition(%q,%q)=%v want %v", tt.from, tt.to, got, tt.want)
		}
	}
}
