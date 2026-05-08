# Queue System

This package implements the background job processing system using River (PostgreSQL-backed queue).

## Architecture

The queue system uses River to process background jobs asynchronously. River stores jobs in PostgreSQL tables, providing ACID guarantees and eliminating the need for Redis.

## Job Types

### 1. NotifyDispatcherJob
- **Purpose**: Simulates sending SMS notifications to dispatchers when a new request is submitted
- **Arguments**: `request_id`, `region_id`
- **Behavior**: Logs a simulated notification and creates an audit log entry
- **Retry**: 3 attempts with exponential backoff

### 2. RefreshMetricsJob
- **Purpose**: Recalculates daily metrics for a specific region and date
- **Arguments**: `date`, `region_id`
- **Behavior**: Aggregates request counts, quantities, and status breakdowns, then upserts to `metrics_daily` table
- **Retry**: 3 attempts with exponential backoff

## Components

### Client (`client.go`)
- Wraps River client for job enqueueing
- Provides methods for both standalone and transactional job insertion
- Manages worker lifecycle (start/stop)

### Workers (`workers.go`)
- `NotifyDispatcherWorker`: Processes dispatcher notifications
- `RefreshMetricsWorker`: Processes metrics refresh jobs
- Both implement River's `Worker` interface

### Job Definitions (`jobs.go`)
- Defines job argument structs
- Implements `Kind()` method for River job type identification
- Custom JSON marshaling for UUID and time.Time fields

## Usage

### Enqueueing Jobs

```go
// Standalone enqueue
err := queueClient.EnqueueNotifyDispatcher(ctx, queue.NotifyDispatcherJobArgs{
    RequestID: requestID,
    RegionID:  regionID,
})

// Transactional enqueue (within a database transaction)
err := queueClient.EnqueueNotifyDispatcherTx(ctx, tx, queue.NotifyDispatcherJobArgs{
    RequestID: requestID,
    RegionID:  regionID,
})
```

### Running the Worker

```bash
# Run worker process
make worker

# Or directly
go run cmd/worker/main.go
```

## Database Tables

River creates the following tables:
- `river_job`: Stores job queue entries
- `river_leader`: Leader election for distributed workers
- `river_migration`: Tracks River schema migrations

See `db/migrations/002_create_river_tables.up.sql` for the schema.

## Configuration

Workers are configured in the River client:
- **Queue**: `default` queue with 10 max workers
- **Retry Policy**: 3 attempts with exponential backoff (River default)
- **Graceful Shutdown**: 30-second timeout

## Monitoring

River provides built-in observability:
- Job state tracking (available, running, completed, failed)
- Attempt counts and error logs
- Scheduled and finalized timestamps

Query the `river_job` table to monitor job status:

```sql
SELECT kind, state, COUNT(*) 
FROM river_job 
GROUP BY kind, state;
```

## Production Considerations

1. **Horizontal Scaling**: Run multiple worker processes for higher throughput
2. **Queue Priorities**: Configure different queues for different job priorities
3. **Dead Letter Queue**: Monitor failed jobs and implement alerting
4. **Metrics**: Export River metrics to your monitoring system
5. **Transactional Enqueue**: Use `InsertTx` methods to ensure atomicity with database operations
