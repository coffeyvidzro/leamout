package cron

import (
	"context"
	"time"

	"github.com/robfig/cron/v3"
)

type Config struct {
	Enabled  bool
	Timezone string
}

type Scheduler struct {
	cron *cron.Cron
}

func New(cfg Config) (*Scheduler, error) {
	location, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return nil, err
	}

	return &Scheduler{
		cron: cron.New(
			cron.WithLocation(location),
			cron.WithChain(
				cron.Recover(cron.DefaultLogger),
				cron.SkipIfStillRunning(cron.DefaultLogger),
			),
		),
	}, nil
}

// AddJob registers a function to run on a schedule.
// Examples: "@hourly", "@daily", "@every 1m".
func (s *Scheduler) AddJob(schedule string, task func()) (cron.EntryID, error) {
	return s.cron.AddFunc(schedule, task)
}

func (s *Scheduler) Start() {
	s.cron.Start()
}

func (s *Scheduler) Stop() context.Context {
	return s.cron.Stop()
}
