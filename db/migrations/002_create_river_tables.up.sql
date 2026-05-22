CREATE TABLE river_job (
    id bigserial PRIMARY KEY,
    args jsonb NOT NULL,
    attempt smallint NOT NULL DEFAULT 0,
    attempted_at timestamptz,
    attempted_by text[],
    created_at timestamptz NOT NULL DEFAULT now(),
    errors jsonb[],
    finalized_at timestamptz,
    kind text NOT NULL,
    max_attempts smallint NOT NULL,x
    metadata jsonb NOT NULL DEFAULT '{}',
    priority smallint NOT NULL DEFAULT 1,
    queue text NOT NULL DEFAULT 'default',
    state text NOT NULL DEFAULT 'available',
    scheduled_at timestamptz NOT NULL DEFAULT now(),
    tags text[] NOT NULL DEFAULT '{}'
);

CREATE INDEX river_job_kind_idx
ON river_job (kind);

CREATE INDEX river_job_state_finalized_idx
ON river_job (state, finalized_at);

CREATE INDEX river_job_fetch_idx
ON river_job (state, queue, priority, scheduled_at, id);

CREATE INDEX river_job_args_idx
ON river_job USING gin (args);

CREATE INDEX river_job_metadata_idx
ON river_job USING gin (metadata);


CREATE TABLE river_queue (
    name text PRIMARY KEY,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    paused_at timestamptz,
    metadata jsonb NOT NULL DEFAULT '{}'
);

CREATE INDEX river_queue_paused_idx
ON river_queue (paused_at);

CREATE TABLE river_leader (
    elected_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    leader_id text NOT NULL,
    name text PRIMARY KEY,
    CONSTRAINT leader_id_length CHECK (
        char_length(leader_id) > 0 AND char_length(leader_id) < 128
    ),
    CONSTRAINT leader_name_length CHECK (
        char_length(name) > 0 AND char_length(name) < 128
    )
);

CREATE TABLE river_migration (
    id bigserial PRIMARY KEY,
    created_at timestamptz NOT NULL DEFAULT now(),
    version bigint NOT NULL,
    CONSTRAINT version CHECK (version >= 1)
);

CREATE UNIQUE INDEX river_migration_version_idx
ON river_migration (version);

INSERT INTO river_queue (name)
VALUES ('default')
ON CONFLICT DO NOTHING;