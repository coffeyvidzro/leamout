package dunning

type AttemptStatus string

const (
	Pending  AttemptStatus = "pending"
	Sent     AttemptStatus = "sent"
	Paid     AttemptStatus = "paid"
	Expired  AttemptStatus = "expired"
	Canceled AttemptStatus = "canceled"
)

var transitions = map[AttemptStatus]map[AttemptStatus]struct{}{
	Pending:  allow(Sent, Paid, Expired, Canceled),
	Sent:     allow(Paid, Expired, Canceled),
	Paid:     allow(),
	Expired:  allow(),
	Canceled: allow(),
}

func CanTransition(from, to AttemptStatus) bool {
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

func Terminal(status AttemptStatus) bool {
	return status == Paid || status == Expired || status == Canceled
}

func allow(statuses ...AttemptStatus) map[AttemptStatus]struct{} {
	m := make(map[AttemptStatus]struct{}, len(statuses))
	for _, status := range statuses {
		m[status] = struct{}{}
	}
	return m
}
