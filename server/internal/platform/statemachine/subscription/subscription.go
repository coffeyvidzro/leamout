package subscription

import "fmt"

type Status string

const (
	Active     Status = "active"
	Canceled   Status = "canceled"
	PastDue    Status = "past_due"
	Trialing   Status = "trialing"
	Incomplete Status = "incomplete"
	Paused     Status = "paused"
)

var transitions = map[Status]map[Status]struct{}{
	Incomplete: allow(Active, Canceled),
	Trialing:   allow(Active, Canceled),
	Active:     allow(PastDue, Paused, Canceled),
	PastDue:    allow(Active, Canceled),
	Paused:     allow(Active, Canceled),
	Canceled:   allow(),
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
	return fmt.Errorf("invalid subscription transition from %s to %s", from, to)
}

func Terminal(status Status) bool { return status == Canceled }

func allow(statuses ...Status) map[Status]struct{} {
	m := make(map[Status]struct{}, len(statuses))
	for _, status := range statuses {
		m[status] = struct{}{}
	}
	return m
}
