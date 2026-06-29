package checkout

import "fmt"

type Status string

const (
	Open      Status = "open"
	Completed Status = "completed"
	Expired   Status = "expired"
	Canceled  Status = "canceled"
)

var transitions = map[Status]map[Status]struct{}{
	Open:      allow(Completed, Expired, Canceled),
	Completed: allow(),
	Expired:   allow(),
	Canceled:  allow(),
}

func CanTransition(from, to Status) bool {
	if from == to {
		return true
	}
	allowed, ok := transitions[from]
	if !ok {
		return false
	}
	_, ok = allowed[to]
	return ok
}

func ValidateTransition(from, to Status) error {
	if CanTransition(from, to) {
		return nil
	}
	return fmt.Errorf("invalid checkout transition from %s to %s", from, to)
}

func Terminal(status Status) bool {
	return status == Completed || status == Expired || status == Canceled
}

func allow(statuses ...Status) map[Status]struct{} {
	m := make(map[Status]struct{}, len(statuses))
	for _, status := range statuses {
		m[status] = struct{}{}
	}
	return m
}
