-- Phase 1: minimal schema
CREATE TABLE IF NOT EXISTS projects (
	id          TEXT PRIMARY KEY,
	name        TEXT NOT NULL,
	slug        TEXT NOT NULL UNIQUE,
	created_at  TEXT NOT NULL DEFAULT (datetime('now')),
	settings    TEXT DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS api_keys (
	id            TEXT PRIMARY KEY,
	project_id    TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
	label         TEXT NOT NULL,
	key_hash      TEXT NOT NULL,
	key_prefix    TEXT NOT NULL,
	last_used_at  TEXT,
	created_at    TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS traces (
	id            TEXT PRIMARY KEY,
	project_id    TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
	run_id        TEXT,
	agent_name    TEXT,
	status        TEXT NOT NULL,
	thread_id     TEXT,
	user_id       TEXT,
	environment   TEXT NOT NULL DEFAULT 'production',
	input         TEXT,
	output        TEXT,
	error         TEXT,
	started_at    TEXT,
	ended_at      TEXT,
	duration_ms   INTEGER,
	created_at    TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS spans (
	trace_id      TEXT NOT NULL,
	span_id       TEXT NOT NULL,
	parent_span_id TEXT,
	project_id    TEXT NOT NULL,
	name          TEXT NOT NULL,
	span_kind     TEXT NOT NULL,
	status        TEXT NOT NULL,
	started_at    TEXT NOT NULL,
	ended_at      TEXT NOT NULL,
	duration_ms   INTEGER NOT NULL,
	attributes    TEXT
);

CREATE TABLE IF NOT EXISTS trace_stats (
	trace_id          TEXT PRIMARY KEY,
	project_id        TEXT NOT NULL,
	total_spans       INTEGER NOT NULL,
	llm_call_count    INTEGER NOT NULL,
	tool_call_count   INTEGER NOT NULL,
	total_input_tokens  INTEGER NOT NULL,
	total_output_tokens INTEGER NOT NULL,
	total_tokens        INTEGER NOT NULL,
	created_at       TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS baselines (
	id            TEXT PRIMARY KEY,
	project_id    TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
	trace_id      TEXT NOT NULL,
	label         TEXT NOT NULL,
	notes         TEXT,
	created_at    TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS diffs (
	id              TEXT PRIMARY KEY,
	project_id      TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
	trace_a_id      TEXT NOT NULL,
	trace_b_id      TEXT NOT NULL,
	similarity_score REAL,
	diff_result     TEXT,
	created_at      TEXT NOT NULL DEFAULT (datetime('now'))
);