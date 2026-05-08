-- River queue tables
-- This migration creates the tables required by River job queue

CREATE TABLE river_job (
    id bigserial PRIMARY KEY,
    args jsonb NOT NULL,
    attempt smallint NOT NULL DEFAULT 0,
    attempted_at timestamptz,
    attempted_by text[],
    created_at timestamptz NOT NULL DEFAULT NOW(),
    errors jsonb[],
    finalized_at timestamptz,
    kind text NOT NULL,
    max_attempts smallint NOT NULL,
    metadata jsonb NOT NULL DEFAULT '{}',
    priority smallint NOT NULL DEFAULT 1,
    queue text NOT NULL DEFAULT 'default',
    state text NOT NULL DEFAULT 'available',
    scheduled_at timestamptz NOT NULL DEFAULT NOW(),
    tags text[] NOT NULL DEFAULT '{}'
);

CREATE INDEX river_job_kind ON river_job (kind);
CREATE INDEX river_job_state_and_finalized_at_index ON river_job (state, finalized_at);
CREATE INDEX river_job_prioritized_fetching_index ON river_job (state, queue, priority, scheduled_at, id);
CREATE INDEX river_job_args_index ON river_job USING gin (args);
CREATE INDEX river_job_metadata_index ON river_job USING gin (metadata);

CREATE TABLE river_leader (
    elected_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    leader_id text NOT NULL,
    name text PRIMARY KEY,
    CONSTRAINT leader_id_length CHECK (char_length(leader_id) > 0 AND char_length(leader_id) < 128),
    CONSTRAINT name_length CHECK (char_length(name) > 0 AND char_length(name) < 128)
);

CREATE TABLE river_migration (
    id bigserial PRIMARY KEY,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    version bigint NOT NULL,
    CONSTRAINT version CHECK (version >= 1)
);

CREATE UNIQUE INDEX river_migration_version_idx ON river_migration (version);
