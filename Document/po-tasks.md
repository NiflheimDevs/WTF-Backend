
- Add `.env.example` with required vars (DB_URL, JWT_SECRET, PORT, etc.)
- Create `Makefile` with targets: `run`, `migrate-up`, `migrate-down`, `test`, `lint`, `seed`

### Task 1.2: Dependencies
- Add chi/gin router
- Add `pgx` or `database/sql` + `pq`
- Add migration tool (`golang-migrate` or `goose`)
- Add JWT library (`golang-jwt/jwt`)
- Add `river` for PG-backed job queue
- Add `bcrypt` for password hashing
- Add rate limiter (`golang.org/x/time/rate`)

---

## **Phase 2: Database Layer**

### Task 2.1: Migrations
Create migrations for:
- `users` table (id, email, password_hash, full_name, role, is_active, timestamps)
- `regions` table (id, name_fa, name_en, parent_id, is_active, display_order)
- `requests` table (all fields per spec, indexes on status/created_at, region/created_at)
- `metrics_daily` table (composite PK: metric_date, region_id, need_type)
- `audit_log` table (append-only event log)
- Enable `pgcrypto` extension

### Task 2.2: Seed data
- Seed 2-3 dispatcher users with hashed passwords
- Seed ~20 regions (districts/neighborhoods) with Persian/English names
- Create dev seed script for 50-100 sample requests

---

## **Phase 3: Domain & Repository Layer**

### Task 3.1: Domain models (`internal/domain/`)
Define structs:
- `User`
- `Region`
- `Request` (with `NeedType` and `Status` enums)
- `MetricsDaily`
- `AuditLog`

### Task 3.2: Repository interfaces & implementations (`internal/repository/`)
- `UserRepository`: `FindByEmail()`, `FindByID()`
- `RegionRepository`: `ListActive()`
- `RequestRepository`: `Create()`, `FindByID()`, `List()` (with filters), `UpdateStatus()`
- `MetricsRepository`: `GetSummary()`, `GetByRegion()`, `GetByNeedType()`, `UpsertDaily()`
- `AuditLogRepository`: `Insert()`

Use **Repository pattern** with interfaces for testability.

---

## **Phase 4: Service Layer**

### Task 4.1: Auth service (`internal/service/auth.go`)
- `Login(email, password)` → validate, return JWT + refresh token
- `RefreshToken(refreshToken)` → issue new access token
- `ValidateToken(token)` → parse and verify JWT

### Task 4.2: Request service (`internal/service/request.go`)
- `Submit(req)` → validate, insert into DB, enqueue `NotifyDispatcherJob`, return request ID
- `GetByID(id)` → fetch request + audit log
- `List(filters)` → paginated list with status/region/date filters
- `UpdateStatus(id, newStatus, actorID)` → transactional update + audit log insert

### Task 4.3: Metrics service (`internal/service/metrics.go`)
- `GetSummary()` → total/pending/dispatched/fulfilled counts
- `GetByRegion()` → top regions by request count
- `GetByNeedType()` → breakdown by bottled_water vs tanker
- `RefreshDailyMetrics(date, regionID)` → recalculate and upsert `metrics_daily`

Use **Factory pattern** for service instantiation (dependency injection container).

---

## **Phase 5: Queue & Worker**

### Task 5.1: Queue setup (`internal/queue/`)
- Initialize `river` with PostgreSQL connection
- Define job structs:
  - `NotifyDispatcherJob` (request_id, region_id)
  - `RefreshMetricsJob` (date, region_id)
- Implement `Enqueue()` helper

### Task 5.2: Worker implementation (`cmd/worker/main.go`)
- Register job handlers using **Strategy pattern**:
  - `NotifyDispatcherHandler`: log simulated notification, insert audit event
  - `RefreshMetricsHandler`: call `MetricsService.RefreshDailyMetrics()`
- Add retry logic (3 attempts, exponential backoff)
- Graceful shutdown on SIGTERM

### Task 5.3: Transactional enqueue
- In `RequestService.Submit()`, use DB transaction to:
  1. Insert request
  2. Enqueue `NotifyDispatcherJob`
  3. Commit atomically

---

## **Phase 6: Middleware**

### Task 6.1: Core middleware (`internal/middleware/`)
- `Logger`: log method, path, status, duration
- `Recovery`: catch panics, return 500
- `RequestID`: generate and inject `X-Request-ID`
- `CORS`: restrict to frontend origin

### Task 6.2: Auth middleware
- `RequireAuth`: validate JWT, extract user ID, inject into context
- Return 401 if missing/invalid token

### Task 6.3: Rate limiter
- IP-based rate limiter: 10 req/min per IP on `POST /api/v1/requests`
- Return 429 if exceeded

---

## **Phase 7: HTTP Handlers**

### Task 7.1: Public handlers (`internal/handler/public.go`)
- `GET /api/v1/regions` → list active regions
- `POST /api/v1/requests` → validate, call `RequestService.Submit()`, return 201
- `GET /api/v1/health` → return 200
- `GET /api/v1/health/ready` → check DB connection, return 200 or 503

### Task 7.2: Auth handlers (`internal/handler/auth.go`)
- `POST /api/v1/auth/login` → call `AuthService.Login()`, return tokens
- `POST /api/v1/auth/refresh` → call `AuthService.RefreshToken()`

### Task 7.3: Dispatcher handlers (`internal/handler/dispatcher.go`)
- `GET /api/v1/dispatcher/requests` → call `RequestService.List()` with filters
- `GET /api/v1/dispatcher/requests/{id}` → call `RequestService.GetByID()`
- `PATCH /api/v1/dispatcher/requests/{id}/status` → call `RequestService.UpdateStatus()`
- `GET /api/v1/dispatcher/metrics/summary` → call `MetricsService.GetSummary()`
- `GET /api/v1/dispatcher/metrics/by-region` → call `MetricsService.GetByRegion()`
- `GET /api/v1/dispatcher/metrics/by-need-type` → call `MetricsService.GetByNeedType()`

All dispatcher routes protected by `RequireAuth` middleware.

---

## **Phase 8: API Server**

### Task 8.1: Router setup (`cmd/api/main.go`)
- Initialize chi/gin router
- Mount middleware: Logger → Recovery → RequestID → CORS
- Mount public routes
- Mount auth routes
- Mount dispatcher routes (with `RequireAuth`)
- Start HTTP server on configured port

### Task 8.2: Error handling
- Standardize error responses:
```json
  {
"error": {
"code": "VALIDATION_ERROR",
"message": "Invalid region_id",
"details": {}
}
  }
  
