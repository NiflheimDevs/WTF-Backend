CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    full_name TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('dispatcher', 'admin')),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE regions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name_fa TEXT NOT NULL,
    name_en TEXT NOT NULL,
    parent_id UUID REFERENCES regions(id) ON DELETE SET NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    display_order INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_regions_active ON regions(is_active, display_order);

CREATE TABLE requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    region_id UUID NOT NULL REFERENCES regions(id),
    need_type TEXT NOT NULL CHECK (need_type IN ('bottled_water', 'tanker')),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    contact_phone TEXT,
    note TEXT CHECK (note IS NULL OR char_length(note) <= 500),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'dispatched', 'fulfilled', 'cancelled')),
    submitted_ip INET,
    submitted_user_agent TEXT,
    dispatched_by UUID REFERENCES users(id) ON DELETE SET NULL,
    dispatched_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_requests_status_created ON requests(status, created_at DESC);
CREATE INDEX idx_requests_region_created ON requests(region_id, created_at DESC);
CREATE INDEX idx_requests_created_at ON requests(created_at DESC);

CREATE TABLE metrics_daily (
    metric_date DATE NOT NULL,
    region_id UUID NOT NULL REFERENCES regions(id),
    need_type TEXT NOT NULL CHECK (need_type IN ('bottled_water', 'tanker')),
    request_count INTEGER NOT NULL DEFAULT 0 CHECK (request_count >= 0),
    total_quantity INTEGER NOT NULL DEFAULT 0 CHECK (total_quantity >= 0),
    pending_count INTEGER NOT NULL DEFAULT 0 CHECK (pending_count >= 0),
    dispatched_count INTEGER NOT NULL DEFAULT 0 CHECK (dispatched_count >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (metric_date, region_id, need_type)
);

CREATE TABLE audit_log (
    id BIGSERIAL PRIMARY KEY,
    event_type TEXT NOT NULL,
    request_id UUID REFERENCES requests(id) ON DELETE SET NULL,
    actor_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_log_request ON audit_log(request_id, created_at DESC);
