-- Postgres Schema — Phase 1: Core tables

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS projects (
    id          UUID PRIMARY KEY,
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    settings    JSONB DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS api_keys (
    id            UUID PRIMARY KEY,
    project_id    UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    label         TEXT NOT NULL,
    key_hash      TEXT NOT NULL,
    key_prefix    TEXT NOT NULL,
    last_used_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS traces (
    id            UUID PRIMARY KEY,
    project_id    UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    run_id        TEXT,
    agent_name    TEXT,
    status        TEXT NOT NULL,
    thread_id     TEXT,
    user_id       TEXT,
    environment   TEXT NOT NULL DEFAULT 'production',
    input         TEXT,
    output        TEXT,
    error         TEXT,
    started_at    TIMESTAMPTZ,
    ended_at      TIMESTAMPTZ,
    duration_ms   BIGINT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS spans (
    trace_id       UUID NOT NULL,
    span_id        TEXT NOT NULL,
    parent_span_id TEXT,
    project_id     UUID NOT NULL,
    name           TEXT NOT NULL,
    span_kind      TEXT NOT NULL,
    status         TEXT NOT NULL,
    started_at     TIMESTAMPTZ NOT NULL,
    ended_at       TIMESTAMPTZ NOT NULL,
    duration_ms    BIGINT NOT NULL,
    attributes     TEXT
);

CREATE TABLE IF NOT EXISTS trace_stats (
    trace_id            UUID PRIMARY KEY,
    project_id          UUID NOT NULL,
    total_spans         INTEGER NOT NULL,
    llm_call_count      INTEGER NOT NULL,
    tool_call_count     INTEGER NOT NULL,
    total_input_tokens  INTEGER NOT NULL,
    total_output_tokens INTEGER NOT NULL,
    total_tokens        INTEGER NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS baselines (
    id            UUID PRIMARY KEY,
    project_id    UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    trace_id      UUID NOT NULL,
    label         TEXT NOT NULL,
    notes         TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS diffs (
    id              UUID PRIMARY KEY,
    project_id      UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    trace_a_id      UUID NOT NULL,
    trace_b_id      UUID NOT NULL,
    similarity_score DOUBLE PRECISION,
    diff_result     TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS issues (
    id                  UUID PRIMARY KEY,
    project_id          UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    fingerprint         TEXT NOT NULL,
    title               TEXT NOT NULL,
    evaluator           TEXT NOT NULL,
    severity            TEXT NOT NULL DEFAULT 'medium',
    status              TEXT NOT NULL DEFAULT 'open',
    first_seen_at       TIMESTAMPTZ NOT NULL,
    last_seen_at        TIMESTAMPTZ NOT NULL,
    occurrence_count    INTEGER DEFAULT 1,
    root_cause          TEXT,
    suggested_fix       TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(project_id, fingerprint)
);

CREATE TABLE IF NOT EXISTS issue_occurrences (
    id          UUID PRIMARY KEY,
    issue_id    UUID NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    trace_id    UUID NOT NULL,
    evidence    TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS metrics (
    id              UUID PRIMARY KEY,
    project_id      UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    aggregation     TEXT NOT NULL,
    filter_json     JSONB DEFAULT '{}',
    window_secs     INTEGER NOT NULL DEFAULT 3600,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS monitors (
    id              UUID PRIMARY KEY,
    metric_id       UUID NOT NULL REFERENCES metrics(id) ON DELETE CASCADE,
    project_id      UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    condition       TEXT NOT NULL,
    threshold       DOUBLE PRECISION NOT NULL,
    severity        TEXT NOT NULL DEFAULT 'high',
    status          TEXT NOT NULL DEFAULT 'ok',
    last_fired_at   TIMESTAMPTZ,
    notify_json     JSONB DEFAULT '[]',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS incidents (
    id                      UUID PRIMARY KEY,
    monitor_id              UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    project_id              UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    status                  TEXT NOT NULL DEFAULT 'unresolved',
    root_cause              TEXT,
    affected_trace_count    INTEGER,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at             TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS webhooks (
    id              UUID PRIMARY KEY,
    project_id      UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    url             TEXT NOT NULL,
    secret_hash     TEXT NOT NULL,
    events          JSONB NOT NULL DEFAULT '[]',
    enabled         BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id            UUID PRIMARY KEY,
    webhook_id    UUID NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    event         TEXT NOT NULL,
    payload       TEXT NOT NULL,
    status_code   INTEGER,
    response      TEXT,
    attempt       INTEGER NOT NULL DEFAULT 1,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    delivered_at  TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS metric_events (
    id            UUID PRIMARY KEY,
    metric_id     UUID NOT NULL REFERENCES metrics(id) ON DELETE CASCADE,
    project_id    UUID NOT NULL,
    value         DOUBLE PRECISION NOT NULL,
    evaluated_at  TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS trace_embeddings (
    trace_id    UUID PRIMARY KEY,
    project_id  UUID NOT NULL,
    embedding   TEXT NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_traces_project_id ON traces(project_id);
CREATE INDEX IF NOT EXISTS idx_traces_created_at ON traces(created_at);
CREATE INDEX IF NOT EXISTS idx_spans_trace_id ON spans(trace_id);
CREATE INDEX IF NOT EXISTS idx_trace_stats_project_id ON trace_stats(project_id);
CREATE INDEX IF NOT EXISTS idx_issues_project_id ON issues(project_id);
CREATE INDEX IF NOT EXISTS idx_issues_status ON issues(status);
CREATE INDEX IF NOT EXISTS idx_issues_fingerprint ON issues(fingerprint);
CREATE INDEX IF NOT EXISTS idx_issue_occurrences_issue_id ON issue_occurrences(issue_id);
CREATE INDEX IF NOT EXISTS idx_issue_occurrences_trace_id ON issue_occurrences(trace_id);
CREATE INDEX IF NOT EXISTS idx_metrics_project_id ON metrics(project_id);
CREATE INDEX IF NOT EXISTS idx_monitors_metric_id ON monitors(metric_id);
CREATE INDEX IF NOT EXISTS idx_monitors_project_id ON monitors(project_id);
CREATE INDEX IF NOT EXISTS idx_incidents_monitor_id ON incidents(monitor_id);
CREATE INDEX IF NOT EXISTS idx_incidents_project_id ON incidents(project_id);
CREATE INDEX IF NOT EXISTS idx_webhooks_project_id ON webhooks(project_id);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_webhook_id ON webhook_deliveries(webhook_id);
CREATE INDEX IF NOT EXISTS idx_metric_events_metric_id ON metric_events(metric_id);
CREATE INDEX IF NOT EXISTS idx_trace_embeddings_project ON trace_embeddings(project_id);
