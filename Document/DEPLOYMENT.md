# Deployment Guide

This document explains how to deploy the Water Supply Reporting System with different worker configurations.

## Deployment Patterns

### Pattern 1: Single Process (Monolith)

Run API and workers in the same process. Simplest deployment, good for small-to-medium scale.

**Configuration:**
```bash
ENABLE_WORKERS=true  # Default
```

**Run:**
```bash
make api
# or
go run cmd/api/main.go
```

**Use Case:**
- Development environment
- Small production deployments
- PaaS with limited service slots
- When traffic is predictable and moderate

**Pros:**
- Simple deployment (one service)
- Lower resource overhead
- Easier to manage

**Cons:**
- Cannot scale API and workers independently
- Worker load affects API response times
- Single point of failure

---

### Pattern 2: Separate Processes (Microservices)

Run API and workers as separate services. Better for production at scale.

**API Service Configuration:**
```bash
ENABLE_WORKERS=false  # Disable workers in API
```

**Worker Service Configuration:**
```bash
# Worker runs with default settings (no HTTP server)
```

**Run:**
```bash
# Terminal 1: API Server (no workers)
ENABLE_WORKERS=false make api

# Terminal 2: Worker Process
make worker
```

**Use Case:**
- Production deployments
- High traffic or job-heavy workloads
- Need to scale API and workers independently
- Critical systems requiring fault isolation

**Pros:**
- Independent scaling (e.g., 3 API instances, 10 worker instances)
- Workers don't affect API latency
- Can restart workers without API downtime
- Better resource allocation

**Cons:**
- More complex deployment
- Higher operational overhead
- Need to manage multiple services

---

## PaaS Deployment Examples

### Railway / Render / Fly.io

#### Single Process Deployment

**Service: API**
```yaml
Build Command: go build -o bin/api cmd/api/main.go
Start Command: ./bin/api
Environment:
  ENABLE_WORKERS=true
  DATABASE_URL=$DATABASE_URL
  JWT_SECRET=$JWT_SECRET
  PORT=8080
```

#### Separate Process Deployment

**Service 1: API**
```yaml
Build Command: go build -o bin/api cmd/api/main.go
Start Command: ./bin/api
Environment:
  ENABLE_WORKERS=false  # Important!
  DATABASE_URL=$DATABASE_URL
  JWT_SECRET=$JWT_SECRET
  PORT=8080
Instances: 2  # Scale horizontally
```

**Service 2: Worker**
```yaml
Build Command: go build -o bin/worker cmd/worker/main.go
Start Command: ./bin/worker
Environment:
  DATABASE_URL=$DATABASE_URL
Instances: 5  # Scale based on job volume
```

---

## Docker Deployment

### Single Process

```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o api cmd/api/main.go

FROM alpine:latest
COPY --from=builder /app/api /api
ENV ENABLE_WORKERS=true
CMD ["/api"]
```

### Separate Processes

**API Dockerfile:**
```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o api cmd/api/main.go

FROM alpine:latest
COPY --from=builder /app/api /api
ENV ENABLE_WORKERS=false
CMD ["/api"]
```

**Worker Dockerfile:**
```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o worker cmd/worker/main.go

FROM alpine:latest
COPY --from=builder /app/worker /worker
CMD ["/worker"]
```

**docker-compose.yml:**
```yaml
version: '3.8'

services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: wtf_backend
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  api:
    build:
      context: .
      dockerfile: Dockerfile.api
    ports:
      - "8080:8080"
    environment:
      DATABASE_URL: postgres://postgres:postgres@postgres:5432/wtf_backend?sslmode=disable
      JWT_SECRET: dev-secret-change-in-production
      ENABLE_WORKERS: "false"
      FRONTEND_ORIGIN: http://localhost:3000
    depends_on:
      - postgres
    deploy:
      replicas: 2

  worker:
    build:
      context: .
      dockerfile: Dockerfile.worker
    environment:
      DATABASE_URL: postgres://postgres:postgres@postgres:5432/wtf_backend?sslmode=disable
    depends_on:
      - postgres
    deploy:
      replicas: 3

volumes:
  postgres_data:
```

---

## Scaling Guidelines

### When to Scale API
- High request rate (> 1000 req/min)
- Response time degradation
- CPU usage consistently > 70%

### When to Scale Workers
- Job queue depth growing
- Jobs taking longer to process
- `river_job` table shows many `available` jobs

### Monitoring Queries

**Check job queue depth:**
```sql
SELECT state, COUNT(*) 
FROM river_job 
GROUP BY state;
```

**Check job processing rate:**
```sql
SELECT 
  kind,
  COUNT(*) FILTER (WHERE state = 'completed') as completed,
  COUNT(*) FILTER (WHERE state = 'running') as running,
  COUNT(*) FILTER (WHERE state = 'available') as pending
FROM river_job
WHERE created_at > NOW() - INTERVAL '1 hour'
GROUP BY kind;
```

---

## Recommendations

**Development:** Use Pattern 1 (single process)
**Staging:** Use Pattern 1 or 2 depending on production plan
**Production (< 10k requests/day):** Use Pattern 1
**Production (> 10k requests/day):** Use Pattern 2

**Crisis Mode:** During water supply emergencies, scale workers aggressively (10-20 instances) to handle notification and metrics spikes.
