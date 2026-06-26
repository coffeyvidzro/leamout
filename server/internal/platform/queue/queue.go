package queue

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

const DefaultQueue = river.QueueDefault

type Config struct {
	Enabled    bool
	MaxWorkers int
}

// Client wraps the River client used by Leamout.
type Client struct {
	River *river.Client[pgx.Tx]
}

// NewClient initializes River using the existing PostgreSQL pool.
func NewClient(pool *pgxpool.Pool, workers *river.Workers, cfg Config) (*Client, error) {
	if workers == nil {
		workers = NewWorkerRegistry()
	}

	maxWorkers := cfg.MaxWorkers
	if maxWorkers <= 0 {
		maxWorkers = 10
	}

	riverConfig := &river.Config{
		Workers: workers,
	}

	// Queues are only needed when this process should execute jobs.
	// For API/scheduler processes that only insert jobs, Queues can stay empty.
	if cfg.Enabled {
		riverConfig.Queues = map[string]river.QueueConfig{
			DefaultQueue: {
				MaxWorkers: maxWorkers,
			},
		}
	}

	client, err := river.NewClient(
		riverpgxv5.New(pool),
		riverConfig,
	)
	if err != nil {
		return nil, err
	}

	return &Client{
		River: client,
	}, nil
}

func (c *Client) Start(ctx context.Context) error {
	return c.River.Start(ctx)
}

func (c *Client) Stop(ctx context.Context) error {
	return c.River.Stop(ctx)
}

func (c *Client) Insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) error {
	_, err := c.River.Insert(ctx, args, opts)
	return err
}
