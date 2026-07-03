-- Phase 2: Observability Layer tables
-- Issues
CREATE TABLE IF NOT EXISTS issues (
    id                  TEXT PRIMARY KEY,
    project_id          TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    fingerprint         TEXT NOT NULL,
    title               TEXT NOT NULL,
    evaluator           TEXT NOT NULL,
    severity            TEXT NOT NULL DEFAULT 'medium',
    status              TEXT NOT NULL DEFAULT 'open',
    first_seen_at       TEXT NOT NULL,
    last_seen_at        TEXT NOT NULL,
    occurrence_count    INTEGER DEFAULT 1,
    root_cause          TEXT,
    suggested_fix       TEXT,
    created_at          TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at          TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(project_id, fingerprint)
);

-- Issue Occurrences (trace <-> issue link)
CREATE TABLE IF NOT EXISTS issue_occurrences (
    id          TEXT PRIMARY KEY,
    issue_id    TEXT NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    trace_id    TEXT NOT NULL,
    evidence    TEXT,
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Metrics
CREATE TABLE IF NOT EXISTS metrics (
    id              TEXT PRIMARY KEY,
    project_id      TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    aggregation     TEXT NOT NULL,
    filter_json     TEXT DEFAULT '{}',
    window_secs     INTEGER NOT NULL DEFAULT 3600,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Monitors
CREATE TABLE IF NOT EXISTS monitors (
    id              TEXT PRIMARY KEY,
    metric_id       TEXT NOT NULL REFERENCES metrics(id) ON DELETE CASCADE,
    project_id      TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    condition       TEXT NOT NULL,
    threshold       REAL NOT NULL,
    severity        TEXT NOT NULL DEFAULT 'high',
    status          TEXT NOT NULL DEFAULT 'ok',
    last_fired_at   TEXT,
    notify_json     TEXT DEFAULT '[]',
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Incidents (monitor-triggered)
CREATE TABLE IF NOT EXISTS incidents (
    id                      TEXT PRIMARY KEY,
    monitor_id              TEXT NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    project_id              TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    status                  TEXT NOT NULL DEFAULT 'unresolved',
    root_cause              TEXT,
    affected_trace_count    INTEGER,
    created_at              TEXT NOT NULL DEFAULT (datetime('now')),
    resolved_at             TEXT
);

-- Webhooks
CREATE TABLE IF NOT EXISTS webhooks (
    id              TEXT PRIMARY KEY,
    project_id      TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    url             TEXT NOT NULL,
    secret_hash     TEXT NOT NULL,
    events          TEXT NOT NULL,
    enabled         INTEGER NOT NULL DEFAULT 1,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Webhook deliveries log
CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id            TEXT PRIMARY KEY,
    webhook_id    TEXT NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    event         TEXT NOT NULL,
    payload       TEXT NOT NULL,
    status_code   INTEGER,
    response      TEXT,
    attempt       INTEGER NOT NULL DEFAULT 1,
    created_at    TEXT NOT NULL DEFAULT (datetime('now')),
    delivered_at  TEXT
);

-- Metric events
CREATE TABLE IF NOT EXISTS metric_events (
    id            TEXT PRIMARY KEY,
    metric_id     TEXT NOT NULL REFERENCES metrics(id) ON DELETE CASCADE,
    project_id    TEXT NOT NULL,
    value         REAL NOT NULL,
    evaluated_at  TEXT NOT NULL,
    created_at    TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Indexes for performance
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