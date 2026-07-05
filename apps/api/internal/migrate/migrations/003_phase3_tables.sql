CREATE TABLE IF NOT EXISTS trace_embeddings (
    trace_id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    embedding TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_trace_embeddings_project ON trace_embeddings(project_id);
