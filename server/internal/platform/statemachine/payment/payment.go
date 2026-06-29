package payment

type Status string

const (
	Pending    Status = "pending"
	Authorized Status = "authorized"
	Captured   Status = "captured"
	Failed     Status = "failed"
	Refunded   Status = "refunded"
	Voided     Status = "voided"
)

var transitions = map[Status]map[Status]struct{}{
	Pending:    allow(Authorized, Captured, Failed, Voided),
	Authorized: allow(Captured, Failed, Voided),
	Captured:   allow(Refunded),
	Failed:     allow(),
	Refunded:   allow(),
	Voided:     allow(),
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

func Terminal(status Status) bool {
	return status == Failed || status == Refunded || status == Voided
}

func allow(statuses ...Status) map[Status]struct{} {
	m := make(map[Status]struct{}, len(statuses))
	for _, status := range statuses {
		m[status] = struct{}{}
	}
	return m
}
