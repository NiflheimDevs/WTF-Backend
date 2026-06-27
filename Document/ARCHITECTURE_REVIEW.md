# Architecture Review — WTF Backend

This document reviews the architecture of the Water Task Force backend from a
**design-pattern** perspective. It assesses how the existing patterns are applied,
points out where the architecture could be strengthened, and gives concrete
recommendations for **adding, editing, or removing** patterns. The goal is the same
as the course's: to make the use of design patterns intentional, correct, and
educational.

> A companion file, [`DESIGN_PATTERNS.md`](./DESIGN_PATTERNS.md), documents the
> patterns **already present**. This file is the critical analysis on top of that.

---

## 1. Current Architecture At A Glance

The project follows a **layered architecture** wired together with **dependency
injection** from a single composition root:

```
cmd/api, cmd/worker        → entry points (composition roots)
internal/handler           → HTTP layer (transport)
internal/middleware        → cross-cutting HTTP concerns
internal/service           → business use cases
internal/repository        → persistence interfaces (ports)
internal/repository/postgres → persistence implementations (adapters)
internal/domain            → entities, value objects, invariants
internal/queue             → background jobs (workers)
internal/app/container.go  → dependency wiring
```

Dependency direction is consistently **inward**: handlers → services → repository
interfaces ← postgres adapters. `domain` depends on nothing. This is healthy and is
the strongest aspect of the codebase.

---

## 2. What Is Done Well

- **Clean layering with stable interfaces.** `internal/repository/repository.go`
  defines the ports; `internal/repository/postgres/` provides the adapters.
  Services depend only on the interfaces, so swapping the datastore is realistic.
- **Single composition root.** `internal/app/container.go` is the only place that
  knows concrete types. Handlers, services, and repositories never construct their
  own dependencies.
- **Rich domain model (partial).** `domain.Request.CanTransitionTo`,
  `Request.IsPending`, and `User.IsDispatcher` keep business rules out of services.
  This is the right instinct.
- **Cross-cutting concerns are isolated.** Middleware (`auth`, `cors`, `rate_limit`,
  `request_id`) is composable and reusable, exactly as the Middleware pattern intends.
- **Async work is decoupled.** Notifications and metrics refresh run on a queue
  (`internal/queue`) so the request path stays fast.

---

## 3. Issues And Weak Spots

### 3.1 The "Service" layer is anemic — business logic leaks into handlers

Several services are thin pass-throughs. `RegionService.ListActive`,
`MetricsService.Summary`, and `MetricsService.ByNeedType` simply forward to a
repository with no domain logic. Meanwhile, **validation that belongs in the domain
or service lives in the handler**:

- `validNeedType`, `validDispatcherStatus`, `validRequestStatus` in
  `internal/handler/request_handler.go`.
- Quantity bounds checks and phone/note trimming in the handler.

This inverts the dependency rule: HTTP code now *knows* domain rules. If a second
transport (e.g. a CLI or gRPC) is added, those rules must be duplicated.

### 3.2 Status transitions are only half a State pattern

`Request.CanTransitionTo` encodes the allowed transitions, which is good. But the
*behavior* attached to each state is scattered:

- `RequestService.UpdateStatus` decides `dispatchedBy` inline.
- The handler decides which statuses a dispatcher may set.
- Side effects (SMS, metrics refresh) are hard-coded per transition in the service.

There is no single place that says "when a request enters `dispatched`, do X, Y, Z".
The state machine is implicit.

### 3.3 Side effects are fire-and-forget (`go func()`)

`RequestService.Create` and `UpdateStatus` launch goroutines that call
`s.queue.Enqueue…` with `context.Background()` and swallow errors with `fmt.Printf`.
This is neither safe nor observable:

- No structured logging (the rest of the app uses `slog`).
- Errors are lost; there is no retry or metric.
- The job is enqueued *after* the DB write with no transactional guarantee — the code
  even has the `Enqueue…Tx` variants available but does not use them.

### 3.4 No interface for the queue — it is a concrete dependency

`RequestService` depends on the concrete `*queue.Client`. Every other collaborator
(user/region/metrics repos) is an interface. This breaks the symmetry and makes
`RequestService` untestable without a real River + Postgres stack.

### 3.5 No interface for the logger

`slog.Default()` is grabbed inside `container.go`, and workers receive a concrete
`*slog.Logger`. There is no `Logger` interface, so swapping or capturing logs in
tests is awkward.

### 3.6 Duplicated helper code across packages

- `clientIP` is implemented **twice**: once in `middleware/rate_limit.go` and once
  in `handler/request_handler.go`, with slightly different return types.
- Error-response writing exists in two places: `handler.WriteError` (with details)
  and `middleware.writeError` (without). The shapes even differ.
- UUID/phone/null handling is repeated in handlers.

### 3.7 Two parallel composition roots drift apart

`cmd/api/main.go` builds the container via `app.NewContainer`, but
`cmd/worker/main.go` re-wires repositories and workers **by hand**, duplicating the
logic in `container.go`. Adding a worker means editing two files, and they can
silently diverge.

### 3.8 Concurrency control inside services is inconsistent

`RequestService` takes no lock and does a read-then-write in `UpdateStatus`
(`FindByID` → `CanTransitionTo` → `UpdateStatus`). Two concurrent dispatch calls can
both pass the guard. There is no optimistic concurrency (e.g. version column) or
database-level row lock.

---

## 4. Recommendations

Recommendations are grouped by **add**, **edit**, and **remove**, in rough priority
order. Each maps to a concrete design pattern.

### 4.1 ADD — Queue port interface (Dependency Inversion)

Define an interface in the service layer and have `queue.Client` satisfy it.

```go
// internal/service/ports.go (or internal/queue/queue.go)
type JobEnqueuer interface {
    EnqueueNotifyDispatcher(ctx context.Context, args queue.NotifyDispatcherJobArgs) error
    EnqueueRefreshMetrics(ctx context.Context, args queue.RefreshMetricsJobArgs) error
    EnqueueSendRequesterSMS(ctx context.Context, args queue.SendRequesterSMSJobArgs) error
}
```

Then `RequestService` depends on `JobEnqueuer`, not `*queue.Client`. Benefits: the
service becomes unit-testable with a fake, and the queue can be swapped for an
in-memory implementation in tests.

### 4.2 ADD — Strategy pattern for request status transitions

Replace the implicit state machine with an explicit set of transition strategies.
Each transition owns its guard, its DB mutation, and its side effects.

```go
type StatusTransition interface {
    CanApply(req *domain.Request) bool
    Apply(ctx context.Context, req *domain.Request, actor uuid.UUID) (*domain.Request, error)
}

type DispatchTransition struct{ requests repository.RequestRepository; queue JobEnqueuer }
type FulfillTransition  struct{ ... }
type CancelTransition   struct{ ... }
```

`RequestService.UpdateStatus` becomes a *lookup*:

```go
transition, ok := s.transitions[status]
if !ok || !transition.CanApply(req) { return nil, ErrInvalidTransition }
return transition.Apply(ctx, req, actorID)
```

This is the canonical Go Strategy pattern and turns the scattered transition logic
into one discoverable registry.

### 4.3 ADD — Result/Notification collector (or at least structured logging)

Replace `fmt.Printf` inside the `go func()` blocks with the injected logger, and
record enqueue failures. Even better, enqueue the jobs **synchronously inside the
request transaction** using the already-existing `Enqueue…Tx` methods so the job and
the row commit atomically (Transactional Outbox). This removes the goroutines
entirely, which is simpler *and* more correct.

### 4.4 ADD — Logger interface

```go
type Logger interface {
    Info(msg string, args ...any)
    Warn(msg string, args ...any)
    Error(msg string, args ...any)
}
```

`*slog.Logger` already satisfies this. Inject it into services and workers instead of
calling `slog.Default()` ad hoc.

### 4.5 ADD — Specification / validator for `CreateRequestInput`

Move all validation out of the handler into a `RequestValidator` (Specification
pattern) or into value-object constructors on the domain:

```go
domain.NewNeedType(s string) (NeedType, error)        // rejects invalid values
domain.NewQuantity(n int) (int, error)                // bounds check in one place
```

Then the handler just decodes JSON and calls the service. A second transport reuses
the same rules for free.

### 4.6 ADD — Shared HTTP helpers (DRY)

Consolidate the two `clientIP` implementations and the two error writers into one
package (e.g. `internal/httpx`). This removes duplication and guarantees a single
error-envelope shape across middleware and handlers.

### 4.7 ADD — Unit of Work / transaction boundary

Introduce a `UnitOfWork` (or `TxManager`) so `RequestService.Create` can run the
INSERT, the audit-log INSERT, and the job enqueue inside one transaction. This makes
the audit log actually reliable (today its error is ignored with `_ =`) and enables
the transactional outbox from §4.3.

### 4.8 EDIT — Consolidate the two composition roots

Move worker registration into `internal/app` (e.g. `app.NewWorkerContainer`) and have
`cmd/worker/main.go` call it, mirroring `cmd/api/main.go`. One source of truth for
which workers exist.

### 4.9 EDIT — Push anemic service logic down, or accept it deliberately

For `RegionService` and `MetricsService`, either:
- add the real business rules (caching, authorization, aggregation) so the service
  earns its existence, or
- document that they are intentional façades kept for layering symmetry.

Right now they read as "we needed a service because the architecture says so", which
is a common but worth-explaining trade-off for a course on design patterns.

### 4.10 EDIT — Make concurrency explicit

Add either:
- an optimistic `version` column on `requests` and a `WHERE version = $X` in
  `UpdateStatus`, or
- `SELECT … FOR UPDATE` inside the transaction from §4.7.

Otherwise the State/Strategy work in §4.2 is undermined by races.

### 4.11 REMOVE — Inline `go func()` goroutines in the service

Once §4.3 (transactional enqueue) is in place, the goroutines should be deleted.
They add nondeterminism and hide errors; the queue already provides retries.

### 4.12 REMOVE — Duplicate `clientIP` and duplicate error writers

Delete one copy of each after §4.6 lands.

---

## 5. Suggested Target Layering (After Refactor)

```
handler  ── depends on ──►  service interfaces
service  ── depends on ──►  repository + JobEnqueuer + Logger + UnitOfWork
domain   ── owns ──►        entities, value objects, Specification validators
queue    ── implements ──►  JobEnqueuer (adapter)
postgres ── implements ──►  repository interfaces (adapter)
app      ── wires ──►       everything (single composition root per binary)
```

Every arrow points toward an interface owned by the *consumer*, which is the defining
property of the Dependency Inversion Principle and the pattern that makes all the
others (Strategy, Repository, Unit of Work) clean.

---

## 6. Priority Summary

| Priority | Action | Pattern |
| --- | --- | --- |
| High | Add `JobEnqueuer` interface | Dependency Inversion |
| High | Remove `go func()` fire-and-forget; use transactional enqueue | Transactional Outbox / Unit of Work |
| High | Add status-transition strategies | Strategy / State |
| High | Move validation out of handlers | Specification / Value Object |
| Medium | Add `Logger` interface | Dependency Inversion |
| Medium | Consolidate composition roots | Composition Root |
| Medium | De-duplicate HTTP helpers | DRY / Facade |
| Medium | Add optimistic locking | Concurrency control |
| Low | Justify or enrich anemic services | Facade / Service Layer |

Applying the High items alone would make this a textbook example for the course:
each layer would depend on interfaces it owns, every business rule would live in
exactly one place, and the system would be fully unit-testable without a database.
