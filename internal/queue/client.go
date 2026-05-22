package queue

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

// Client wraps the River client for job enqueueing
type Client struct {
	river *river.Client[pgx.Tx]
}

// NewClient creates a new queue client
func NewClient(pool *pgxpool.Pool, workers *river.Workers) (*Client, error) {
	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 10},
		},
		Workers: workers,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create river client: %w", err)
	}

	return &Client{river: riverClient}, nil
}

// Start starts the River client to begin processing jobs
func (c *Client) Start(ctx context.Context) error {
	return c.river.Start(ctx)
}

// Stop gracefully stops the River client
func (c *Client) Stop(ctx context.Context) error {
	return c.river.Stop(ctx)
}

// EnqueueNotifyDispatcher enqueues a dispatcher notification job
func (c *Client) EnqueueNotifyDispatcher(ctx context.Context, args NotifyDispatcherJobArgs) error {
	_, err := c.river.Insert(ctx, args, nil)
	if err != nil {
		return fmt.Errorf("failed to enqueue notify dispatcher job: %w", err)
	}
	return nil
}

// EnqueueNotifyDispatcherTx enqueues a dispatcher notification job within a transaction
func (c *Client) EnqueueNotifyDispatcherTx(ctx context.Context, tx pgx.Tx, args NotifyDispatcherJobArgs) error {
	_, err := c.river.InsertTx(ctx, tx, args, nil)
	if err != nil {
		return fmt.Errorf("failed to enqueue notify dispatcher job in tx: %w", err)
	}
	return nil
}

// EnqueueRefreshMetrics enqueues a metrics refresh job
func (c *Client) EnqueueRefreshMetrics(ctx context.Context, args RefreshMetricsJobArgs) error {
	_, err := c.river.Insert(ctx, args, nil)
	if err != nil {
		return fmt.Errorf("failed to enqueue refresh metrics job: %w", err)
	}
	return nil
}

// EnqueueRefreshMetricsTx enqueues a metrics refresh job within a transaction
func (c *Client) EnqueueRefreshMetricsTx(ctx context.Context, tx pgx.Tx, args RefreshMetricsJobArgs) error {
	_, err := c.river.InsertTx(ctx, tx, args, nil)
	if err != nil {
		return fmt.Errorf("failed to enqueue refresh metrics job in tx: %w", err)
	}
	return nil
}
