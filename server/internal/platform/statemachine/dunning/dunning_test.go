package dunning

import "testing"

func TestCanTransition(t *testing.T) {
	tests := []struct {
		from, to AttemptStatus
		want     bool
	}{{Pending, Sent, true}, {Sent, Paid, true}, {Sent, Expired, true}, {Paid, Sent, false}, {Canceled, Paid, false}, {Pending, Pending, true}}
	for _, tt := range tests {
		if got := CanTransition(tt.from, tt.to); got != tt.want {
			t.Fatalf("CanTransition(%q,%q)=%v want %v", tt.from, tt.to, got, tt.want)
		}
	}
}
