# Design Patterns Used in This Codebase

This document explains the design patterns **already present** in the Water Task
Force backend, with the exact files where each one appears and why it was chosen.
It is the reference companion to [`ARCHITECTURE_REVIEW.md`](./ARCHITECTURE_REVIEW.md),
which critiques these patterns and proposes changes.

---

## 1. Layered Architecture

The whole project is split into layers, each with one responsibility. Dependencies
only point **inward** (toward the domain).

**Where**
- `internal/domain/` — entities and invariants (`request.go`, `user.go`, `region.go`).
- `internal/repository/` — persistence interfaces (ports).
- `internal/repository/postgres/` — PostgreSQL adapters.
- `internal/service/` — use cases (`request_service.go`, `auth_service.go`, ...).
- `internal/handler/` — HTTP transport (`router.go`, `*_handler.go`).
- `cmd/api/`, `cmd/worker/` — entry points.

**Example flow** (create a water request):
`POST /api/v1/requests` → `RequestHandler.Create` → `RequestService.Create` →
`repository.RequestRepository` → `postgres.RequestRepository.Create`.

**Why** — keeps SQL, HTTP, and business rules in separate places so each can change
independently and be tested in isolation.

---

## 2. Repository Pattern

Data access is hidden behind interfaces. Services never see SQL or `pgx`.

**Where**
- Interfaces: `internal/repository/repository.go`
  (`UserRepository`, `RegionRepository`, `RequestRepository`, `MetricsRepository`,
  `AuditLogRepository`).
- Implementations: `internal/repository/postgres/*_repository.go`.

**Example**

```go
// internal/repository/repository.go
type RequestRepository interface {
    Create(ctx context.Context, req *domain.Request) error
    FindByID(ctx context.Context, id uuid.UUID) (*domain.Request, error)
    List(ctx context.Context, filters RequestFilters) ([]*domain.Request, error)
    UpdateStatus(ctx context.Context, id uuid.UUID, status domain.RequestStatus, dispatchedBy *uuid.UUID) error
    Count(ctx context.Context, filters RequestFilters) (int, error)
}
```

```go
// internal/repository/postgres/request_repository.go
func NewRequestRepository(db *DB) repository.RequestRepository {
    return &RequestRepository{db: db}
}
```

**Why** — the storage backend can be replaced or mocked without touching services.

---

## 3. Dependency Injection / Composition Root

Components receive their collaborators through constructors; nothing reaches for
globals. All wiring happens in one place.

**Where**
- Composition root: `internal/app/container.go` (API),
  `cmd/worker/main.go` (worker).
- Constructor injection: every `New*Service`, `New*Handler`, `New*Worker`.

**Example**

```go
// internal/app/container.go
users := postgres.NewUserRepository(db)
...
return &Container{
    Auth:     service.NewAuthService(users, cfg.JWTSecret, cfg.JWTTTL),
    Requests: service.NewRequestService(requests, regions, audit, queueClient),
    Metrics:  service.NewMetricsService(metrics),
}, nil
```

```go
// internal/handler/router.go — handlers are injected via a Dependencies struct
func NewRouter(deps Dependencies, cfg RouterConfig) http.Handler { ... }
```

**Why** — centralizes object creation, removes hidden dependencies, and makes every
component substitutable in tests.

---

## 4. Middleware (Chain of Responsibility / Decorator)

HTTP handlers are wrapped with reusable cross-cutting behavior. Each middleware
implements `func(http.Handler) http.Handler` and calls `next`.

**Where** — `internal/middleware/` and registration in `internal/handler/router.go`.

| Middleware | File | Concern |
| --- | --- | --- |
| `RequestID` | `request_id.go` | Correlation ID per request |
| `CORS` | `cors.go` | Cross-origin headers |
| `Auth` | `auth.go` | JWT verification, claims into context |
| `RateLimiter.Middleware` | `rate_limit.go` | Per-IP token-bucket throttling |
| `Recoverer` | chi built-in | Panic recovery |

**Example**

```go
r.With(rateLimiter.Middleware).Post("/requests", requestHandler.Create)
r.Group(func(r chi.Router) {
    r.Use(middleware.Auth(deps.Auth))
    r.Get("/dispatcher/requests", requestHandler.List)
})
```

**Why** — auth, CORS, rate limiting, and tracing stay out of the business handlers
and can be applied per-route or globally.

---

## 5. Worker / Job Queue (Producer–Consumer)

Slow or side-effecting work (notifying dispatchers, sending SMS, refreshing metrics)
is pushed onto a queue and processed asynchronously by workers.

**Where**
- Job argument types: `internal/queue/jobs.go`
  (`NotifyDispatcherJobArgs`, `RefreshMetricsJobArgs`, `SendRequesterSMSJobArgs`).
- Queue client wrapper: `internal/queue/client.go`.
- Workers: `internal/queue/workers.go`.
- Enqueue sites: `internal/service/request_service.go`.

**Example** — a service enqueues, a worker consumes:

```go
// service enqueues after creating a request
s.queue.EnqueueNotifyDispatcher(ctx, queue.NotifyDispatcherJobArgs{
    RequestID: req.ID, RegionID: req.RegionID,
})
```

```go
// internal/queue/workers.go
type RefreshMetricsWorker struct {
    river.WorkerDefaults[RefreshMetricsJobArgs]
    requestRepo repository.RequestRepository
    metricsRepo repository.MetricsRepository
    logger      *slog.Logger
}
func (w *RefreshMetricsWorker) Work(ctx context.Context, job *river.Job[RefreshMetricsJobArgs]) error { ... }
```

**Why** — keeps API responses fast, lets failed jobs retry independently, and
decouples request handling from notifications/analytics.

---

## 6. Adapter (Ports & Adapters / Hexagonal)

The repository interfaces are **ports**; the `postgres` package are **adapters**.
The transport layer (chi handlers) is also an adapter over the service core.

**Where**
- Port: `internal/repository/repository.go`.
- Adapter: `internal/repository/postgres/`.
- The `queue.Client` adapts the third-party `river` library behind a small surface.

**Why** — the core domain and services stay unaware of PostgreSQL, River, or chi, so
any of them can be swapped without rippling changes inward.

---

## 7. Strategy (partial — state transitions)

`domain.Request.CanTransitionTo` encodes the rules that decide which status a request
may move to. The method is a strategy table keyed by current status.

**Where** — `internal/domain/request.go`.

```go
func (r *Request) CanTransitionTo(newStatus RequestStatus) bool {
    switch r.Status {
    case StatusPending:
        return newStatus == StatusDispatched || newStatus == StatusCancelled
    case StatusDispatched:
        return newStatus == StatusFulfilled || newStatus == StatusCancelled
    case StatusFulfilled, StatusCancelled:
        return false // terminal states
    }
    return false
}
```

`RequestService.UpdateStatus` consults it before persisting. (See the review doc for
how to promote this into a full State/Strategy implementation.)

---

## 8. Value Object (partial — typed enumerations)

Concepts that are "just strings" are given named types with a fixed set of constants,
so the compiler catches invalid values.

**Where** — `internal/domain/`:

| Type | Constants |
| --- | --- |
| `NeedType` | `NeedTypeBottledWater`, `NeedTypeTanker` |
| `RequestStatus` | `StatusPending`, `StatusDispatched`, `StatusFulfilled`, `StatusCancelled` |
| `Role` | `RoleDispatcher`, `RoleAdmin` |
| `EventType` | `EventRequestSubmitted`, `EventRequestStatusChanged`, ... |

**Why** — prevents accidental mixing of, say, a status and a role, and documents the
legal values in one place.

---

## 9. Rich Domain Model (behavior on entities)

Business queries live on the entities themselves rather than in services, so the
rules travel with the data.

**Where**
- `Request.IsPending()`, `Request.IsDispatched()`, `Request.IsFulfilled()`.
- `User.IsDispatcher()`, `User.IsAdmin()` (admin implies dispatcher).

**Why** — keeps domain rules readable and co-located with the entity they describe.

---

## 10. Façade (thin service layer)

Some services (`RegionService`, `MetricsService`) expose a simplified interface over
a repository without adding logic. They act as a façade so the handler layer always
talks to a service, keeping the layering uniform.

**Where** — `internal/service/region_service.go`, `internal/service/metrics_service.go`.

**Why** — preserves a consistent handler→service→repository call shape even where
there is no extra business rule yet.

---

## 11. Error Sentinels & Envelope

Domain/service errors are sentinel values; the handler maps them to a uniform JSON
envelope.

**Where**
- Sentinels: `internal/service/errors.go`
  (`ErrInvalidCredentials`, `ErrInactiveUser`, `ErrNotFound`, `ErrInvalidTransition`).
- Envelope: `internal/handler/response_handler.go`
  (`ErrorEnvelope`, `WriteError`, `WriteJSON`).

**Why** — callers test with `errors.Is`, and clients always receive the same error
shape regardless of which handler failed.

---

## Pattern Map

| Pattern | Primary Location |
| --- | --- |
| Layered Architecture | `internal/{domain,repository,service,handler}` |
| Repository | `internal/repository/`, `internal/repository/postgres/` |
| Dependency Injection / Composition Root | `internal/app/container.go`, `New*` constructors |
| Middleware (Decorator/Chain) | `internal/middleware/`, `internal/handler/router.go` |
| Worker / Job Queue | `internal/queue/`, `internal/service/request_service.go` |
| Adapter (Ports & Adapters) | repository interfaces vs. `postgres` adapters |
| Strategy (state transitions) | `internal/domain/request.go` (`CanTransitionTo`) |
| Value Object | `NeedType`, `RequestStatus`, `Role`, `EventType` |
| Rich Domain Model | `Request.*`, `User.*` methods |
| Façade | `RegionService`, `MetricsService` |
| Error Sentinels & Envelope | `internal/service/errors.go`, `internal/handler/response_handler.go` |
