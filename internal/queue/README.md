# Queue System

This package implements the background job processing system using River (PostgreSQL-backed queue).

## Architecture

The queue system uses River to process background jobs asynchronously. River stores jobs in PostgreSQL tables, providing ACID guarantees and eliminating the need for Redis.

### Deployment Flexibility

The system supports two deployment patterns:

1. **Single Process**: API and workers run together (set `ENABLE_WORKERS=true`)
2. **Separate Processes**: API and workers run independently (set `ENABLE_WORKERS=false` for API, run `cmd/worker` separately)

See `DEPLOYMENT.md` for detailed deployment strategies.

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
- Workers are conditionally started based on `ENABLE_WORKERS` config

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
// Standalone enqueue (used in request service)
err := queueClient.EnqueueNotifyDispatcher(ctx, queue.NotifyDispatcherJobArgs{
    RequestID: requestID,
    RegionID:  regionID,
})

// Transactional enqueue (for future use with DB transactions)
err := queueClient.EnqueueNotifyDispatcherTx(ctx, tx, queue.NotifyDispatcherJobArgs{
    RequestID: requestID,
    RegionID:  regionID,
})
```

### Running Workers

**Option 1: With API (Single Process)**
```bash
ENABLE_WORKERS=true make api
```

**Option 2: Separate Process**
```bash
# Terminal 1: API without workers
ENABLE_WORKERS=false make api

# Terminal 2: Dedicated worker
make worker
```

## Database Tables

River creates the following tables:
- `river_job`: Stores job queue entries
- `river_leader`: Leader election for distributed workers
- `river_migration`: Tracks River schema migrations

See `db/migrations/002_create_river_tables.up.sql` for the schema.

## Configuration

### Environment Variables

- `ENABLE_WORKERS`: Set to `false` to disable worker processing in API (default: `true`)

### Worker Configuration

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
-- Overall queue health
SELECT state, COUNT(*) 
FROM river_job 
GROUP BY state;

-- Job processing by type
SELECT 
  kind,
  COUNT(*) FILTER (WHERE state = 'completed') as completed,
  COUNT(*) FILTER (WHERE state = 'running') as running,
  COUNT(*) FILTER (WHERE state = 'available') as pending,
  COUNT(*) FILTER (WHERE state = 'retryable') as retrying
FROM river_job
WHERE created_at > NOW() - INTERVAL '1 hour'
GROUP BY kind;

-- Failed jobs requiring attention
SELECT id, kind, errors, attempted_at
FROM river_job
WHERE state = 'retryable' AND attempt >= max_attempts
ORDER BY attempted_at DESC
LIMIT 10;
```

## Production Considerations

1. **Horizontal Scaling**: Run multiple worker processes for higher throughput
2. **Independent Scaling**: Scale API and workers separately based on load
3. **Queue Priorities**: Configure different queues for different job priorities
4. **Dead Letter Queue**: Monitor failed jobs and implement alerting
5. **Metrics**: Export River metrics to your monitoring system
6. **Transactional Enqueue**: Use `InsertTx` methods to ensure atomicity with database operations

## Scaling Strategy

- **Low Traffic** (< 10k requests/day): Single process with `ENABLE_WORKERS=true`
- **Medium Traffic** (10k-100k requests/day): 2-3 API instances + 3-5 worker instances
- **High Traffic** (> 100k requests/day): Scale API and workers independently based on metrics
- **Crisis Mode**: During emergencies, scale workers aggressively (10-20 instances)
