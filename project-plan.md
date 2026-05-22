# Water Supply Reporting System (سامانه گزارش و تامین آب)
## Project Plan & Technical Specification

> **Project Type:** Crisis Management Web Application
> **Stack:** Go (Golang) · PostgreSQL · React.js · Tailwind CSS
> **Deployment Target:** PaaS (Railway / Render / Fly.io / Heroku)
> **Document Version:** 1.0

---

## 1. Project Overview & Objectives

### 1.1 Context
During water-supply crises (drought, infrastructure failure, contamination events, regional shortages), affected residents need a frictionless way to report shortages, and dispatchers need actionable, aggregated intelligence to route relief — bottled water shipments and water tankers — to the highest-need areas first. Long forms, account creation, and complex dashboards all introduce delays that cost lives in a crisis. This system is built around the opposite principle: **report in seconds, dispatch in minutes**.

### 1.2 Core Objectives
The system has four objectives that drive every design decision:

1. **Zero-friction reporting.** A citizen on a slow mobile connection in a power-degraded area must be able to submit a request in under 15 seconds with no login, no email verification, and no account.
2. **Operational clarity for dispatchers.** Authenticated dispatchers must see the freshest aggregated picture — total requests today, top-N highest-need regions, pending vs. dispatched counts — without waiting for slow queries or complex chart rendering.
3. **Resilience under load.** A localized crisis can produce traffic spikes (hundreds of reports per minute from a single neighborhood). The submission API must remain responsive even when downstream notification or aggregation work is slow or temporarily failing.
4. **Demonstrable engineering rigor.** The codebase must showcase clean asynchronous architecture, recognizable design patterns, and complete documentation suitable for academic or technical review.

### 1.3 Out of Scope (Explicit Non-Goals)
To keep the scope minimal and shippable, the following are explicitly excluded from v1: real SMS gateway integration (simulated only), interactive maps, multi-language UI beyond Persian/English labels, role hierarchies beyond Reporter/Dispatcher, payment or donation flows, and predictive analytics or ML.

---

## 2. System Architecture & Data Flow

### 2.1 High-Level Architecture

The system follows a classic **three-tier architecture** augmented with an **asynchronous worker tier**, all hosted on a single PaaS.

```
┌──────────────────────┐        ┌──────────────────────┐
│  Reporter Browser    │        │  Dispatcher Browser  │
│  (React + Tailwind)  │        │  (React + Tailwind)  │
└──────────┬───────────┘        └──────────┬───────────┘
           │ HTTPS / JSON                  │ HTTPS / JSON + JWT
           ▼                               ▼
┌─────────────────────────────────────────────────────────┐
│                    Go API Server                        │
│   (chi/gin router · handlers · services · repos)        │
│                                                         │
│   ┌──────────────┐   ┌──────────────┐  ┌────────────┐   │
│   │  Reporter    │   │  Dispatcher  │  │  Metrics   │   │
│   │  endpoints   │   │  endpoints   │  │  endpoint  │   │
│   └──────┬───────┘   └──────┬───────┘  └─────┬──────┘   │
│          │                  │                │          │
│          ▼                  ▼                ▼          │
│   ┌────────────────────────────────────────────────┐    │
│   │        Service Layer (business logic)          │    │
│   └────────┬───────────────────────┬───────────────┘    │
│            │                       │                    │
│            ▼                       ▼                    │
│   ┌────────────────┐      ┌────────────────────┐        │
│   │  Repository    │      │   Job Enqueuer     │        │
│   │  (pgx)         │      │  (publishes jobs)  │        │
│   └────────┬───────┘      └─────────┬──────────┘        │
└────────────┼────────────────────────┼───────────────────┘
             │                        │
             ▼                        ▼
   ┌────────────────────┐    ┌────────────────────┐
   │    PostgreSQL      │    │   Job Queue        │
   │  (requests,        │◄───│   (Redis/asynq OR  │
   │   users, regions,  │    │   PG-backed: river │
   │   metrics_daily,   │    │   /gue)            │
   │   audit_log)       │    └─────────┬──────────┘
   └────────────────────┘              │
              ▲                        ▼
              │              ┌────────────────────┐
              └──────────────┤   Go Worker        │
                             │   (background      │
                             │    process)        │
                             └────────────────────┘
```

### 2.2 Component Responsibilities

The **Go API server** is a stateless HTTP service that handles request validation, authentication (JWT for dispatchers), persistence of new reports, and serving read endpoints. It deliberately does no slow work synchronously: when a report arrives, the handler validates, persists the row, enqueues a job, and returns `202 Accepted` (or `201 Created`) immediately.

The **PostgreSQL database** is the single source of truth. It stores users, regions (a seeded reference table of neighborhoods), requests (the main transactional table), a `metrics_daily` rollup table updated by the worker, and an audit log of status transitions.

The **job queue** decouples submission from background work. Two pragmatic choices: (a) Redis-backed `hibiken/asynq` — battle-tested, low operational cost on PaaS that offers Redis add-ons; or (b) PostgreSQL-backed `riverqueue/river` or `vgarvardt/gue` — zero extra infrastructure, transactional enqueue. **Recommendation: river (PG-backed)** for this project, because it eliminates an extra service on the PaaS and gives transactional "enqueue with insert" semantics.

The **Go worker** is a separate process (same binary, different `cmd/worker` entrypoint) that consumes jobs and performs: simulated SMS notification (logged to stdout and persisted to `audit_log`), aggregated metrics refresh, and structured logging. Workers are horizontally scalable.

### 2.3 Synchronous Report Submission Flow

When a reporter submits a request, the lifecycle is:

1. React form posts JSON to `POST /api/v1/requests` with `region_id`, `need_type`, `quantity`, optional contact phone, and a free-text note.
2. Go handler validates payload (region exists, need_type in enum, quantity > 0), inserts the row inside a transaction, and within the same transaction enqueues two jobs: `notify_dispatcher` and `refresh_metrics_for_region`.
3. Transaction commits atomically — either both the request and its jobs are persisted, or neither is.
4. Handler returns `201 Created` with the new request's UUID. Total target latency: under 150ms p95.

### 2.4 Asynchronous Background Flow

Workers poll the queue (or are notified via `LISTEN/NOTIFY` for river). For each job type:

- **`notify_dispatcher`** — fetches the request, formats a simulated SMS message ("New tanker request in district X, quantity Y"), writes it to the `audit_log` table with `event_type='notification_sent'`, and logs structured JSON to stdout for observability.
- **`refresh_metrics_for_region`** — runs an `INSERT ... ON CONFLICT DO UPDATE` against `metrics_daily` to increment counters for `(region_id, date, need_type)`. This pre-aggregates the heavy work so the dispatcher dashboard reads from a small, indexed rollup table instead of doing `GROUP BY` over the full requests table on every load.

If a job fails, the queue retries with exponential backoff (defaults: 3 attempts, 1m / 5m / 25m). Permanently failed jobs land in a dead-letter view inspectable by the dispatcher (out of scope for v1 UI but logged).

---

## 3. Database Schema Design

All tables use UUIDs (`gen_random_uuid()` from `pgcrypto`) as primary keys. All timestamps are `TIMESTAMPTZ` in UTC. The schema is intentionally narrow.

### 3.1 `users`
Stores dispatcher accounts only. Reporters are anonymous.

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | UUID | PK, default `gen_random_uuid()` | |
| `email` | TEXT | UNIQUE, NOT NULL | Used as login identifier |
| `password_hash` | TEXT | NOT NULL | bcrypt, cost 12 |
| `full_name` | TEXT | NOT NULL | |
| `role` | TEXT | NOT NULL, CHECK IN ('dispatcher','admin') | Future-proofs role expansion |
| `is_active` | BOOLEAN | NOT NULL DEFAULT TRUE | Soft-disable account |
| `created_at` | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | |
| `updated_at` | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | |

### 3.2 `regions`
Seeded reference table of neighborhoods/districts. Editable only via migrations or admin tooling.

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | UUID | PK | |
| `name_fa` | TEXT | NOT NULL | Persian display name |
| `name_en` | TEXT | NOT NULL | English/transliterated name |
| `parent_id` | UUID | FK → regions.id, NULLABLE | Allows district → neighborhood hierarchy |
| `is_active` | BOOLEAN | NOT NULL DEFAULT TRUE | |
| `display_order` | INTEGER | NOT NULL DEFAULT 0 | Controls dropdown ordering |

Index: `CREATE INDEX idx_regions_active ON regions(is_active, display_order);`

### 3.3 `requests`
The main transactional table.

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | UUID | PK | |
| `region_id` | UUID | FK → regions.id, NOT NULL | |
| `need_type` | TEXT | NOT NULL, CHECK IN ('bottled_water','tanker') | Extensible enum |
| `quantity` | INTEGER | NOT NULL, CHECK > 0 | Bottles or tankers count |
| `contact_phone` | TEXT | NULLABLE | Optional reporter callback |
| `note` | TEXT | NULLABLE, max 500 chars | Free-text context |
| `status` | TEXT | NOT NULL DEFAULT 'pending', CHECK IN ('pending','dispatched','fulfilled','cancelled') | |
| `submitted_ip` | INET | NULLABLE | For abuse mitigation |
| `submitted_user_agent` | TEXT | NULLABLE | |
| `dispatched_by` | UUID | FK → users.id, NULLABLE | |
| `dispatched_at` | TIMESTAMPTZ | NULLABLE | |
| `created_at` | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | |
| `updated_at` | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | |

Indexes:
```
CREATE INDEX idx_requests_status_created ON requests(status, created_at DESC);
CREATE INDEX idx_requests_region_created ON requests(region_id, created_at DESC);
CREATE INDEX idx_requests_created_at ON requests(created_at DESC);
```

### 3.4 `metrics_daily`
Pre-aggregated rollup updated by the worker. Reading from here is O(rows-in-day) instead of scanning `requests`.

| Column | Type | Constraints |
|---|---|---|
| `metric_date` | DATE | PK part 1 |
| `region_id` | UUID | PK part 2, FK → regions.id |
| `need_type` | TEXT | PK part 3 |
| `request_count` | INTEGER | NOT NULL DEFAULT 0 |
| `total_quantity` | INTEGER | NOT NULL DEFAULT 0 |
| `pending_count` | INTEGER | NOT NULL DEFAULT 0 |
| `dispatched_count` | INTEGER | NOT NULL DEFAULT 0 |
| `updated_at` | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |

Composite PK: `(metric_date, region_id, need_type)`.

### 3.5 `audit_log`
Append-only log of meaningful events.

| Column | Type | Constraints |
|---|---|---|
| `id` | BIGSERIAL | PK |
| `event_type` | TEXT | NOT NULL (e.g. 'request_created', 'status_changed', 'notification_sent') |
| `request_id` | UUID | FK → requests.id, NULLABLE |
| `actor_user_id` | UUID | FK → users.id, NULLABLE (NULL for system/worker) |
| `payload` | JSONB | NOT NULL — flexible event details |
| `created_at` | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |

Index: `CREATE INDEX idx_audit_log_request ON audit_log(request_id, created_at DESC);`

### 3.6 Migration Strategy
Use `golang-migrate/migrate` with timestamped SQL files in `db/migrations/`. Migrations run automatically on app boot in non-production and manually via a one-off PaaS command in production.

---

## 4. API Specification

All endpoints are versioned under `/api/v1`. JSON only. Errors use a consistent envelope: `{ "error": { "code": "string", "message": "string", "details": {} } }`.

### 4.1 Public Endpoints (No Auth)

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/api/v1/regions` | List active regions for the dropdown. Returns `[{id, name_fa, name_en}]`, cached 5 minutes. |
| `POST` | `/api/v1/requests` | Submit a new water request. Body: `{region_id, need_type, quantity, contact_phone?, note?}`. Returns `201` with `{id, status, created_at}`. Rate-limited per IP. |
| `GET` | `/api/v1/health` | Liveness probe for PaaS. |
| `GET` | `/api/v1/health/ready` | Readiness probe — checks DB connectivity. |

### 4.2 Dispatcher Endpoints (JWT Required)

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/api/v1/auth/login` | Body: `{email, password}`. Returns `{access_token, expires_at, user}`. JWT TTL 8h. |
| `POST` | `/api/v1/auth/refresh` | Optional: refresh token rotation. |
| `GET` | `/api/v1/dispatcher/requests` | Paginated list. Query params: `status`, `region_id`, `from`, `to`, `page`, `page_size`. Sort: newest first. |
| `GET` | `/api/v1/dispatcher/requests/{id}` | Detail view including audit log entries. |
| `PATCH` | `/api/v1/dispatcher/requests/{id}/status` | Body: `{status: "dispatched"\|"fulfilled"\|"cancelled"}`. Records `dispatched_by` and emits an audit event. |
| `GET` | `/api/v1/dispatcher/metrics/summary` | Top-level KPIs: `{requests_today, pending_count, dispatched_today, avg_response_minutes}`. |
| `GET` | `/api/v1/dispatcher/metrics/by-region` | `[{region_id, name_fa, request_count, pending_count}, ...]` sorted by need, top 10. Reads from `metrics_daily`. |
| `GET` | `/api/v1/dispatcher/metrics/by-need-type` | `[{need_type, count, total_quantity}]` for today. |

### 4.3 Validation & Security
- Request body validation via `go-playground/validator`.
- Rate limiting on `POST /requests` — 10 req/min per IP using a token-bucket middleware backed by an in-memory store (sufficient for v1; swap to Redis later).
- CORS: allow only the deployed frontend origin.
- All responses include `X-Request-ID` for traceability.
- Secrets (DB URL, JWT signing key) come from environment variables only; never committed.

---

## 5. Step-by-Step Implementation Plan

The plan is organized into seven phases, sequenced so each phase produces a demoable artifact and unblocks the next. Estimated total effort for one engineer: 3–4 weeks.

### Phase 1 — Project Setup & Foundations (Days 1–2)

Initialize a monorepo with `apps/api` (Go) and `apps/web` (React/Vite). Configure Go modules with the layout: `cmd/api`, `cmd/worker`, `internal/handler`, `internal/service`, `internal/repository`, `internal/queue`, `internal/domain`, `internal/middleware`, `db/migrations`. Set up `golangci-lint`, `gofumpt`, and `air` for live reload. On the frontend, scaffold Vite + React + TypeScript + Tailwind, with ESLint and Prettier. Create a root `Makefile` with targets `make api`, `make worker`, `make web`, `make migrate`, `make test`. Initialize Git with a clean `.gitignore` (ignore `.env`, `dist/`, `bin/`, `node_modules/`, `coverage/`) and a `.env.example` documenting all required variables. Branching model: `main` (protected, deployable) and short-lived `feat/*` branches merged via PR.

### Phase 2 — Database & Migrations (Days 3–4)

Author migrations for all six tables in order: `users`, `regions`, `requests`, `metrics_daily`, `audit_log`, plus a seed migration inserting initial dispatcher users (with bcrypt-hashed passwords) and ~20 representative regions. Add the `pgcrypto` extension migration first. Verify migrations are reversible by running `migrate down` then `migrate up` cleanly. Write a small `scripts/seed_dev_requests.go` that generates 200 fake requests across regions and dates so the dashboard has data to display from day one.

### Phase 3 — Backend API (Days 5–10)

Build the API server in layers, bottom-up. Start with the **Repository layer** using `jackc/pgx/v5` directly (no ORM — keeps SQL explicit and fast). Implement `RequestRepository`, `UserRepository`, `RegionRepository`, `MetricsRepository`, each with an interface and a concrete `pgxRequestRepository` implementation. Then the **Service layer** containing business logic: `RequestService.Submit()` orchestrates DB insert + job enqueue inside one transaction; `AuthService.Login()` verifies bcrypt and issues JWT; `MetricsService.Summary()` reads the rollup table. Then **Handlers** (thin — they unmarshal, call service, marshal response) using `go-chi/chi` for routing. Add **middleware**: structured logging (`log/slog`), recovery, CORS, request ID, JWT auth (only on dispatcher routes), rate limiter on submission. Write unit tests for services using mock repositories and integration tests for handlers using `testcontainers-go` to spin up real Postgres.

### Phase 4 — Async Workers & Queue (Days 11–13)

Add `riverqueue/river` (PG-backed) as the queue. Define two job types in `internal/queue/jobs/`: `NotifyDispatcherJob{RequestID uuid.UUID}` and `RefreshMetricsJob{RegionID uuid.UUID, Date time.Time, NeedType string}`. Each job type has a `Worker` struct implementing `river.Worker[T]` with a `Work(ctx, job) error` method. The notification worker formats a message and writes an `audit_log` entry with `event_type='notification_sent'` (real SMS gateway out of scope). The metrics worker performs an `INSERT ... ON CONFLICT (metric_date, region_id, need_type) DO UPDATE SET ...` upsert. Wire the enqueue calls into `RequestService.Submit()` using river's `client.InsertTx()` so the job is enqueued atomically with the request insert. Build a separate `cmd/worker/main.go` entrypoint that boots a river `Client` configured to consume from the queues. Verify retries by intentionally panicking in a worker and observing exponential backoff.

### Phase 5 — Frontend (Days 14–19)

Build the React app as two distinct route trees. The **Reporter tree** has a single page: a centered card with region dropdown (populated from `GET /regions`), a two-button toggle for need type (bottled water / tanker), a numeric quantity input with sensible defaults, optional phone and note fields, and a large submit button. Tailwind handles styling — clean typography, generous touch targets (≥44px), full RTL support via `dir="rtl"` and `tailwindcss-rtl` plugin or logical properties. On submit, show a success toast with the request ID and reset the form. The **Dispatcher tree** lives behind `/dispatcher/*`: a login page; a layout with a top nav showing the logged-in user; a dashboard page with four KPI cards (Requests Today, Pending, Dispatched Today, Avg Response Time) and two simple ranked lists (Top Regions by Pending Need, Breakdown by Need Type) — all rendered as plain HTML/Tailwind, no chart library; and a Requests page with a filterable, sortable table where each row has a status dropdown that calls `PATCH /requests/{id}/status`. Use `@tanstack/react-query` for data fetching/caching, `react-router` v6, and `zustand` (or React context) for auth state. Store the JWT in `httpOnly` cookie if possible; otherwise `localStorage` with explicit XSS hardening.

### Phase 6 — Architecture Documentation (Days 20–21)

Create `ARCHITECTURE.md` at the repo root. Sections: System Context, Component Diagram (ASCII), Sequence Diagrams (two — see below), Design Patterns Used, Data Model, Failure Modes & Retries, Security Notes, Deployment Topology. The two sequence diagrams use Mermaid syntax so they render natively on GitHub:

**Diagram 1 — Synchronous Report Submission:** `Reporter Browser → API Handler → RequestService → DB (BEGIN) → DB (INSERT request) → Queue (enqueue jobs in same tx) → DB (COMMIT) → API Handler → Reporter Browser (201)`.

**Diagram 2 — Asynchronous Background Processing:** `Worker → Queue (poll/notify) → Worker.Work() → DB (read request) → simulated SMS log → DB (INSERT audit_log) → DB (UPSERT metrics_daily) → Queue (ack)`. Include retry/dead-letter branches.

### Phase 7 — Deployment to PaaS (Days 22–23)

Recommended target: **Railway** or **Render** (both offer Postgres add-ons and support background workers as separate services natively). Create two services from the same repo: `api` (build: `go build -o bin/api ./cmd/api`, start: `./bin/api`) and `worker` (`go build -o bin/worker ./cmd/worker`, start: `./bin/worker`). Provision a managed PostgreSQL add-on; inject `DATABASE_URL` as an env var into both services. Run migrations as a one-off pre-deploy step. Deploy the React app as a static site (Render Static Site or Railway static service), with `VITE_API_BASE_URL` pointing to the API's public URL. Configure custom domain (optional) and TLS (automatic on these PaaS). Final step: smoke-test the live URL — submit a report as a reporter, log in as a dispatcher, dispatch the request, verify the metric increments.

---

## 6. Design Patterns Strategy

The codebase implements four recognizable patterns. The DoD requires two; including all four costs little and meaningfully strengthens the architecture.

### 6.1 Repository Pattern (Backend, mandatory)

The Repository pattern abstracts data persistence behind an interface, decoupling business logic from PostgreSQL specifics. Each domain entity (`Request`, `User`, `Region`, `Metrics`) has a Go interface in `internal/repository/` declaring methods like `FindByID`, `Insert`, `UpdateStatus`. A concrete `pgxRequestRepository` implements the interface using `pgx`. Services depend on the interface, not the concrete type, which means: tests substitute in-memory mocks trivially; swapping Postgres for another store would touch only one file; and SQL stays localized and auditable. This is the spine of the backend's testability.

### 6.2 Strategy Pattern (Backend, for job processing)

Different job types (`NotifyDispatcherJob`, `RefreshMetricsJob`, future `SendEmailJob`, `ExportReportJob`) share a common contract — `Work(ctx, job) error` — but execute very different logic. The river queue dispatches each incoming job to its registered worker strategy at runtime. Adding a new background task type means writing a new strategy struct and registering it; no existing code changes. This is textbook Strategy: a family of interchangeable algorithms behind a uniform interface, selected at runtime by job type.

### 6.3 Factory Pattern (Backend, for service construction)

A `app.NewContainer(cfg)` factory function constructs the full dependency graph: opens the DB pool, instantiates all repositories, wires services with their repository dependencies, builds the river client, and returns a struct exposing the assembled components. Both `cmd/api` and `cmd/worker` call the same factory at boot, guaranteeing identical wiring. This eliminates duplicate setup code and makes the dependency graph inspectable in one place.

### 6.4 Container/Presentational Pattern (Frontend)

React components split into two categories. **Container components** (e.g. `DashboardContainer`, `RequestsTableContainer`) own data fetching via React Query, manage state, and pass data + callbacks down as props. **Presentational components** (e.g. `KpiCard`, `RegionRankList`, `StatusBadge`) are pure functions of props with no fetching, no state beyond UI concerns, and no side effects. This makes presentational components trivially reusable and unit-testable in Storybook or with React Testing Library, while containers stay thin and focused.

### 6.5 Pattern Documentation

Each pattern gets a dedicated subsection in `ARCHITECTURE.md` answering: *What problem does it solve here? Where in the code is it implemented? What would be harder without it?* This satisfies the DoD requirement that patterns be both implemented and documented.

---

## 7. Definition of Done (DoD) Checklist

A feature/release is considered shipped only when **every** item below is verifiably true.

### Functional Completeness
- [ ] Reporter form is reachable at the public URL with no authentication.
- [ ] Reporter form submits successfully and returns a request ID within 500ms p95.
- [ ] Dispatcher login works with seeded credentials and rejects invalid passwords.
- [ ] Dispatcher dashboard displays at least four live KPIs sourced from real DB data.
- [ ] Dispatcher can view a paginated, filterable table of requests.
- [ ] Dispatcher can change a request's status to "Dispatched" and the change persists.
- [ ] Status changes are visible in the audit log table.

### Asynchronous Architecture
- [ ] Submission API returns before background work completes (verified by inspecting timing logs).
- [ ] At least two distinct job types are processed by the worker.
- [ ] Worker runs as a separate process (separate PaaS service), not inside the API.
- [ ] A failing job retries with backoff and is observable in logs.
- [ ] Job enqueue is transactional with the originating DB write.

### Metrics Implementation
- [ ] At least one metric is computed via PostgreSQL `GROUP BY` (the worker's upsert into `metrics_daily` is sourced from grouped queries during backfills).
- [ ] Dispatcher metrics endpoints return data in under 100ms p95.
- [ ] Metrics reflect newly submitted requests within 60 seconds.
- [ ] Frontend renders metrics as plain numeric cards/lists — no chart libraries imported.

### Design Patterns
- [ ] At least two design patterns are implemented in the codebase.
- [ ] Each pattern has a dedicated section in `ARCHITECTURE.md` with file references.
- [ ] Repository interfaces are mocked in at least one unit test.

### Documentation
- [ ] `README.md` covers setup, env vars, running locally, and deployment.
- [ ] `ARCHITECTURE.md` exists at repo root.
- [ ] `ARCHITECTURE.md` contains a synchronous-flow Mermaid sequence diagram.
- [ ] `ARCHITECTURE.md` contains an asynchronous-flow Mermaid sequence diagram.
- [ ] `.env.example` documents every required environment variable with comments.
- [ ] API endpoints are documented (inline OpenAPI comments or a simple `API.md`).

### Code Quality & Version Control
- [ ] No `.env`, secrets, or credentials committed (verified via `git log -p` or `gitleaks`).
- [ ] All work merged to `main` via pull requests with descriptive titles.
- [ ] Commit messages follow Conventional Commits (`feat:`, `fix:`, `docs:`, `chore:`).
- [ ] `golangci-lint run` passes with zero warnings.
- [ ] Frontend builds with zero TypeScript errors and zero ESLint warnings.
- [ ] Unit test coverage on backend service layer ≥ 60%.

### Deployment
- [ ] Application is deployed to a PaaS with HTTPS.
- [ ] Live URL is documented in `README.md`.
- [ ] Database migrations run successfully against the production database.
- [ ] API and worker are running as independent PaaS services.
- [ ] `/health` and `/health/ready` endpoints respond 200 in production.
- [ ] An end-to-end smoke test (submit → dispatch → metric increment) passes against the live URL.

---

## Appendix A — Recommended Library Choices

**Backend:** `go-chi/chi` (router), `jackc/pgx/v5` (DB driver), `riverqueue/river` (queue), `golang-migrate/migrate` (migrations), `go-playground/validator` (validation), `golang-jwt/jwt/v5` (auth), `log/slog` (logging), `golang.org/x/crypto/bcrypt` (hashing), `testcontainers-go` (integration tests).

**Frontend:** `react`, `react-router-dom`, `@tanstack/react-query`, `tailwindcss`, `axios` or native `fetch`, `react-hook-form` + `zod` (form validation), `clsx` (class composition).

**Tooling:** `golangci-lint`, `gofumpt`, `air` (live reload), `vite`, `eslint`, `prettier`, `gitleaks` (pre-commit secret scan).

## Appendix B — Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Submission spike overwhelms DB | API errors during crisis | Rate limit per IP; PaaS auto-scaling; queue absorbs downstream load |
| Worker falls behind queue | Stale dashboards | Horizontally scale worker; alert on queue depth |
| Region dropdown becomes stale | Reporters can't find their area | Admin-only migration for region edits; cache TTL kept low (5m) |
| Lost JWT secret on redeploy | All sessions invalidated | Store in PaaS secret manager; document recovery procedure |
| Test data leaks to production | Confused metrics | Separate seed migrations gated by `APP_ENV != production` |
