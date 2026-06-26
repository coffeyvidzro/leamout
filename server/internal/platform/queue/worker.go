package queue

import "github.com/riverqueue/river"

// NewWorkerRegistry creates a new River worker collection.
func NewWorkerRegistry() *river.Workers {
	return river.NewWorkers()
}
