# Design Patterns Used in This Project

This backend uses several design patterns to keep the codebase modular, testable, and easier to extend. The most visible patterns are Repository, Dependency Injection, Layered Architecture, Middleware, and Worker/Job Queue.

## 1. Repository Pattern

The Repository Pattern separates business logic from database access. Instead of services writing SQL directly, they depend on repository interfaces.

### Where it appears

- Interfaces are defined in `internal/repository/repository.go`.
- PostgreSQL implementations live in `internal/repository/postgres/`.
- Services consume repository interfaces from `internal/service/`.

### Example

`internal/repository/repository.go` defines contracts such as:

```go
type RequestRepository interface {
    Create(ctx context.Context, req *domain.Request) error
    FindByID(ctx context.Context, id uuid.UUID) (*domain.Request, error)
    List(ctx context.Context, filters RequestFilters) ([]*domain.Request, error)
    UpdateStatus(ctx context.Context, id uuid.UUID, status domain.RequestStatus, dispatchedBy *uuid.UUID) error
    Count(ctx context.Context, filters RequestFilters) (int, error)
}
```

`internal/repository/postgres/request_repository.go` provides the PostgreSQL implementation:

```go
type RequestRepository struct {
    db *DB
}

func NewRequestRepository(db *DB) repository.RequestRepository {
    return &RequestRepository{db: db}
}
```

### Why it is useful

- Services do not need to know how data is stored.
- Database logic is isolated in one package.
- It becomes easier to replace PostgreSQL repositories with mocks or another storage backend.
- Tests can target business logic without requiring direct SQL setup.

## 2. Dependency Injection Pattern

Dependency Injection provides components with the dependencies they need instead of letting them create those dependencies internally.

### Where it appears

- `internal/app/container.go` wires the application together.
- Service constructors receive repositories and configuration values.
- Router creation receives dependencies through `handler.Dependencies`.

### Example

`internal/app/container.go` creates repositories and injects them into services:

```go
users := postgres.NewUserRepository(db)
regions := postgres.NewRegionRepository(db)
requests := postgres.NewRequestRepository(db)
audit := postgres.NewAuditLogRepository(db)
metrics := postgres.NewMetricsRepository(db)

return &Container{
    Auth:     service.NewAuthService(users, cfg.JWTSecret, cfg.JWTTTL),
    Regions:  service.NewRegionService(regions),
    Requests: service.NewRequestService(requests, regions, audit, queueClient),
    Metrics:  service.NewMetricsService(metrics),
}, nil
```

`internal/service/request_service.go` receives its dependencies through a constructor:

```go
func NewRequestService(
    requests repository.RequestRepository,
    regions repository.RegionRepository,
    audit repository.AuditLogRepository,
    queue *queue.Client,
) *RequestService {
    return &RequestService{
        requests: requests,
        regions:  regions,
        audit:    audit,
        queue:    queue,
    }
}
```

### Why it is useful

- Object creation is centralized in the application container.
- Services are easier to test because dependencies can be substituted.
- The code avoids hidden global dependencies.
- Configuration and infrastructure are kept separate from business logic.

## 3. Layered Architecture Pattern

The project is organized into layers, where each layer has a clear responsibility and depends only on lower-level abstractions.

### Main layers

- `internal/domain/`: core business entities and domain rules.
- `internal/repository/`: data-access interfaces.
- `internal/repository/postgres/`: PostgreSQL persistence implementations.
- `internal/service/`: business use cases and workflows.
- `internal/handler/`: HTTP request/response handling.
- `internal/middleware/`: reusable HTTP middleware.
- `cmd/api/` and `cmd/worker/`: application entry points.

### Example flow

A request creation flow generally moves through these layers:

1. HTTP route is registered in `internal/handler/router.go`.
2. Handler parses and validates the HTTP request.
3. Service in `internal/service/request_service.go` applies business rules.
4. Repository interface from `internal/repository/repository.go` is used for persistence.
5. PostgreSQL implementation in `internal/repository/postgres/request_repository.go` executes SQL.

### Why it is useful

- Each layer has a focused responsibility.
- Business logic is not mixed with HTTP or SQL code.
- New delivery mechanisms, such as CLI or background jobs, can reuse services.
- The codebase is easier to navigate and maintain.

## 4. Middleware Pattern

The Middleware Pattern wraps HTTP handlers with reusable request-processing behavior.

### Where it appears

Middleware functions are located in `internal/middleware/` and are registered in `internal/handler/router.go`.

### Example

`internal/handler/router.go` applies middleware globally:

```go
r.Use(chimiddleware.Recoverer)
r.Use(middleware.RequestID)
r.Use(middleware.CORS(cfg.AllowedOrigin))
```

It also applies authentication middleware only to dispatcher routes:

```go
r.Group(func(r chi.Router) {
    r.Use(middleware.Auth(deps.Auth))
    r.Get("/dispatcher/requests", requestHandler.List)
    r.Get("/dispatcher/metrics/summary", metricsHandler.Summary)
})
```

### Why it is useful

- Cross-cutting concerns stay out of handlers.
- Authentication, CORS, request IDs, recovery, and rate limiting are reusable.
- Middleware can be applied globally or only to selected route groups.

## 5. Worker/Job Queue Pattern

The project uses background jobs to process work asynchronously after an API request completes.

### Where it appears

- Job argument types are defined in `internal/queue/jobs.go`.
- Workers are implemented in `internal/queue/workers.go`.
- Queue client wrapper is implemented in `internal/queue/client.go`.
- Jobs are enqueued from services such as `internal/service/request_service.go`.

### Example

`internal/service/request_service.go` enqueues background jobs after a request is created:

```go
go func() {
    bgCtx := context.Background()

    _ = s.queue.EnqueueNotifyDispatcher(bgCtx, queue.NotifyDispatcherJobArgs{
        RequestID: req.ID,
        RegionID:  req.RegionID,
    })

    _ = s.queue.EnqueueRefreshMetrics(bgCtx, queue.RefreshMetricsJobArgs{
        Date:     now,
        RegionID: req.RegionID,
    })
}()
```

`internal/queue/workers.go` contains workers that process those jobs:

```go
type RefreshMetricsWorker struct {
    river.WorkerDefaults[RefreshMetricsJobArgs]
    requestRepo repository.RequestRepository
    metricsRepo repository.MetricsRepository
    logger      *slog.Logger
}
```

### Why it is useful

- Slow or non-critical work does not block API responses.
- Notifications and metrics refreshes can be retried independently.
- Background processing is separated from request handling.
- Workers can reuse repositories and services through injected dependencies.

## Summary

| Pattern | Main Location | Purpose |
| --- | --- | --- |
| Repository | `internal/repository/`, `internal/repository/postgres/` | Isolates database access behind interfaces |
| Dependency Injection | `internal/app/container.go`, service constructors | Provides dependencies explicitly and improves testability |
| Layered Architecture | `internal/domain/`, `internal/service/`, `internal/handler/` | Separates domain, business, HTTP, and persistence concerns |
| Middleware | `internal/middleware/`, `internal/handler/router.go` | Reuses HTTP request processing behavior |
| Worker/Job Queue | `internal/queue/`, `internal/service/request_service.go` | Runs asynchronous background work |
