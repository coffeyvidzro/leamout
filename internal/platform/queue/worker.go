package queue

import "github.com/riverqueue/river"

// NewWorkerRegistry creates a new River worker collection.
func NewWorkerRegistry() *river.Workers {
	workers := river.NewWorkers()

	// Register jobs here later.
	// Example:
	// river.AddWorker(workers, &billing.SendRenewalSMSWorker{})

	return workers
}
