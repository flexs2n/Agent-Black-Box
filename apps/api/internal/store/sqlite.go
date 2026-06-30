package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/blackbox-agentdiff/api/internal/model"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type SQLiteStore struct {
	db *sqlx.DB
}

func NewSQLiteStore(databaseURL string) (*SQLiteStore, error) {
	dsn := databaseURL
	if !strings.Contains(dsn, "_time_format=") {
		if strings.Contains(dsn, "?") {
			dsn += "&_time_format=sqlite"
		} else {
			dsn += "?_time_format=sqlite"
		}
	}
	db, err := sqlx.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Close() error { return s.db.Close() }

func (s *SQLiteStore) ProjectCreate(ctx context.Context, p model.ProjectCreate) (model.Project, error) {
	project := model.Project{
		ID:        uuid.New().String(),
		Name:      p.Name,
		Slug:      p.Slug,
		CreatedAt: time.Now(),
	}
	if p.Settings != nil {
		b, _ := json.Marshal(p.Settings)
		project.Settings = string(b)
	}
	_, err := s.db.NamedExecContext(ctx, `INSERT INTO projects (id, name, slug, settings) VALUES (:id, :name, :slug, :settings)`, project)
	return project, err
}

func (s *SQLiteStore) ProjectGet(ctx context.Context, projectID string) (model.Project, error) {
	var p model.Project
	err := s.db.GetContext(ctx, &p, `SELECT id, name, slug, created_at, settings FROM projects WHERE id = ?`, projectID)
	return p, err
}

func (s *SQLiteStore) ProjectList(ctx context.Context) ([]model.Project, error) {
	var projects []model.Project
	err := s.db.SelectContext(ctx, &projects, `SELECT id, name, slug, created_at, settings FROM projects ORDER BY created_at DESC`)
	return projects, err
}

func (s *SQLiteStore) ProjectUpdate(ctx context.Context, projectID string, p model.ProjectUpdate) (model.Project, error) {
	existing, err := s.ProjectGet(ctx, projectID)
	if err != nil {
		return existing, err
	}
	if p.Name != nil {
		existing.Name = *p.Name
	}
	if p.Settings != nil {
		b, _ := json.Marshal(p.Settings)
		existing.Settings = string(b)
	}
	_, err = s.db.NamedExecContext(ctx, `UPDATE projects SET name = :name, settings = :settings WHERE id = :id`, existing)
	return existing, err
}

func (s *SQLiteStore) ProjectDelete(ctx context.Context, projectID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM projects WHERE id = ?`, projectID)
	return err
}

func (s *SQLiteStore) APIKeyCreate(ctx context.Context, projectID, label, keyHash, keyPrefix string) (model.APIKey, error) {
	key := model.APIKey{
		ID:        uuid.New().String(),
		ProjectID: projectID,
		Label:     label,
		KeyHash:   keyHash,
		KeyPrefix: keyPrefix,
		CreatedAt: time.Now(),
	}
	_, err := s.db.NamedExecContext(ctx, `INSERT INTO api_keys (id, project_id, label, key_hash, key_prefix) VALUES (:id, :project_id, :label, :key_hash, :key_prefix)`, key)
	return key, err
}

func (s *SQLiteStore) APIKeyList(ctx context.Context, projectID string) ([]model.APIKey, error) {
	var keys []model.APIKey
	err := s.db.SelectContext(ctx, &keys, `SELECT id, project_id, label, key_hash, key_prefix, last_used_at, created_at FROM api_keys WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	return keys, err
}

func (s *SQLiteStore) APIKeyGetByPrefix(ctx context.Context, prefix string) (model.APIKey, error) {
	var key model.APIKey
	err := s.db.GetContext(ctx, &key, `SELECT id, project_id, label, key_hash, key_prefix, last_used_at, created_at FROM api_keys WHERE key_prefix = ? LIMIT 1`, prefix)
	return key, err
}

func (s *SQLiteStore) APIKeyDelete(ctx context.Context, projectID, keyID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM api_keys WHERE id = ? AND project_id = ?`, keyID, projectID)
	return err
}

func (s *SQLiteStore) APIKeyMarkUsed(ctx context.Context, keyID string) error {
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `UPDATE api_keys SET last_used_at = ? WHERE id = ?`, now, keyID)
	return err
}

func (s *SQLiteStore) TraceCreate(ctx context.Context, t model.Trace) error {
	_, err := s.db.NamedExecContext(ctx, `INSERT INTO traces (id, project_id, run_id, agent_name, status, thread_id, user_id, environment, input, output, error, started_at, ended_at, duration_ms) VALUES (:id, :project_id, :run_id, :agent_name, :status, :thread_id, :user_id, :environment, :input, :output, :error, :started_at, :ended_at, :duration_ms)`, &t)
	return err
}

func (s *SQLiteStore) TraceGet(ctx context.Context, traceID string) (model.Trace, error) {
	var t model.Trace
	err := s.db.GetContext(ctx, &t, `SELECT id, project_id, run_id, agent_name, status, thread_id, user_id, environment, input, output, error, started_at, ended_at, duration_ms, created_at FROM traces WHERE id = ?`, traceID)
	return t, err
}

func (s *SQLiteStore) TraceGetByIDs(ctx context.Context, ids []string) ([]model.Trace, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	query, args, err := sqlx.In(`SELECT id, project_id, run_id, agent_name, status, thread_id, user_id, environment, input, output, error, started_at, ended_at, duration_ms, created_at FROM traces WHERE id IN (?)`, ids)
	if err != nil {
		return nil, err
	}
	query = s.db.Rebind(query)
	var traces []model.Trace
	err = s.db.SelectContext(ctx, &traces, query, args...)
	return traces, err
}

func (s *SQLiteStore) TraceList(ctx context.Context, projectID string, filters TraceFilters) ([]model.Trace, error) {
	base := `SELECT id, project_id, run_id, agent_name, status, thread_id, user_id, environment, input, output, error, started_at, ended_at, duration_ms, created_at FROM traces WHERE project_id = ?`
	args := []any{projectID}
	order := filters.Sort
	if order == "" {
		order = "created_at DESC"
	}
	const pageSize = 50
	offset := (filters.Page - 1) * pageSize
	query := fmt.Sprintf("%s ORDER BY %s LIMIT ? OFFSET ?", base, order)
	args = append(args, pageSize, offset)
	var traces []model.Trace
	err := s.db.SelectContext(ctx, &traces, query, args...)
	return traces, err
}

func (s *SQLiteStore) TraceDelete(ctx context.Context, traceID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM traces WHERE id = ?`, traceID)
	return err
}

func (s *SQLiteStore) TraceDeleteBulk(ctx context.Context, traceIDs []string) error {
	if len(traceIDs) == 0 {
		return nil
	}
	query, args, err := sqlx.In(`DELETE FROM traces WHERE id IN (?)`, traceIDs)
	if err != nil {
		return err
	}
	query = s.db.Rebind(query)
	_, err = s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *SQLiteStore) TraceSearch(ctx context.Context, projectID string, filters TraceSearchFilters) ([]model.Trace, error) {
	base := `SELECT id, project_id, run_id, agent_name, status, thread_id, user_id, environment, input, output, error, started_at, ended_at, duration_ms, created_at FROM traces WHERE project_id = ?`
	args := []any{projectID}

	if filters.AgentName != "" {
		base += ` AND agent_name = ?`
		args = append(args, filters.AgentName)
	}
	if filters.Status != "" {
		base += ` AND status = ?`
		args = append(args, filters.Status)
	}
	if filters.Environment != "" {
		base += ` AND environment = ?`
		args = append(args, filters.Environment)
	}
	if filters.UserID != "" {
		base += ` AND user_id = ?`
		args = append(args, filters.UserID)
	}
	if filters.ThreadID != "" {
		base += ` AND thread_id = ?`
		args = append(args, filters.ThreadID)
	}

	base += ` ORDER BY created_at DESC LIMIT 100`
	var traces []model.Trace
	err := s.db.SelectContext(ctx, &traces, base, args...)
	return traces, err
}

func (s *SQLiteStore) SpanPutBatch(ctx context.Context, spans []model.Span) error {
	if len(spans) == 0 {
		return nil
	}
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO spans (trace_id, span_id, parent_span_id, project_id, name, span_kind, status, started_at, ended_at, duration_ms, attributes) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	for _, sp := range spans {
		if _, err := stmt.ExecContext(ctx, sp.TraceID, sp.SpanID, sp.ParentSpanID, sp.ProjectID, sp.Name, sp.SpanKind, sp.Status, sp.StartedAt, sp.EndedAt, sp.DurationMs, sp.Attributes); err != nil {
			stmt.Close()
			tx.Rollback()
			return err
		}
	}
	stmt.Close()
	return tx.Commit()
}

func (s *SQLiteStore) SpanList(ctx context.Context, traceID string) ([]model.Span, error) {
	var spans []model.Span
	err := s.db.SelectContext(ctx, &spans, `SELECT trace_id, span_id, parent_span_id, project_id, name, span_kind, status, started_at, ended_at, duration_ms, attributes FROM spans WHERE trace_id = ? ORDER BY started_at`, traceID)
	return spans, err
}

func (s *SQLiteStore) TraceStatsPut(ctx context.Context, stats model.TraceStats) error {
	_, err := s.db.NamedExecContext(ctx, `INSERT OR REPLACE INTO trace_stats (trace_id, project_id, total_spans, llm_call_count, tool_call_count, total_input_tokens, total_output_tokens, total_tokens) VALUES (:trace_id, :project_id, :total_spans, :llm_call_count, :tool_call_count, :total_input_tokens, :total_output_tokens, :total_tokens)`, stats)
	return err
}

func (s *SQLiteStore) TraceStatsGet(ctx context.Context, traceID string) (model.TraceStats, error) {
	var stats model.TraceStats
	err := s.db.GetContext(ctx, &stats, `SELECT trace_id, project_id, total_spans, llm_call_count, tool_call_count, total_input_tokens, total_output_tokens, total_tokens FROM trace_stats WHERE trace_id = ?`, traceID)
	return stats, err
}

func (s *SQLiteStore) BaselineCreate(ctx context.Context, b model.BaselineCreate) (model.Baseline, error) {
	baseline := model.Baseline{
		ID:        uuid.New().String(),
		ProjectID: b.ProjectID,
		TraceID:   b.TraceID,
		Label:     b.Label,
		Notes:     b.Notes,
		CreatedAt: time.Now(),
	}
	_, err := s.db.NamedExecContext(ctx, `INSERT INTO baselines (id, project_id, trace_id, label, notes) VALUES (:id, :project_id, :trace_id, :label, :notes)`, baseline)
	return baseline, err
}

func (s *SQLiteStore) BaselineList(ctx context.Context, projectID string) ([]model.Baseline, error) {
	var baselines []model.Baseline
	err := s.db.SelectContext(ctx, &baselines, `SELECT id, project_id, trace_id, label, notes, created_at FROM baselines WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	return baselines, err
}

func (s *SQLiteStore) BaselineGet(ctx context.Context, projectID, label string) (model.Baseline, error) {
	var b model.Baseline
	err := s.db.GetContext(ctx, &b, `SELECT id, project_id, trace_id, label, notes, created_at FROM baselines WHERE project_id = ? AND label = ? LIMIT 1`, projectID, label)
	return b, err
}

func (s *SQLiteStore) BaselineDelete(ctx context.Context, baselineID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM baselines WHERE id = ?`, baselineID)
	return err
}

func (s *SQLiteStore) DiffPut(ctx context.Context, d model.Diff) error {
	_, err := s.db.NamedExecContext(ctx, `INSERT INTO diffs (id, project_id, trace_a_id, trace_b_id, similarity_score, diff_result) VALUES (:id, :project_id, :trace_a_id, :trace_b_id, :similarity_score, :diff_result)`, &d)
	return err
}

func (s *SQLiteStore) DiffGet(ctx context.Context, diffID string) (model.Diff, error) {
	var d model.Diff
	err := s.db.GetContext(ctx, &d, `SELECT id, project_id, trace_a_id, trace_b_id, similarity_score, diff_result, created_at FROM diffs WHERE id = ?`, diffID)
	return d, err
}

func (s *SQLiteStore) DiffGetByTraces(ctx context.Context, projectID, traceAID, traceBID string) (model.Diff, error) {
	var d model.Diff
	err := s.db.GetContext(ctx, &d, `SELECT id, project_id, trace_a_id, trace_b_id, similarity_score, diff_result, created_at FROM diffs WHERE project_id = ? AND ((trace_a_id = ? AND trace_b_id = ?) OR (trace_a_id = ? AND trace_b_id = ?)) LIMIT 1`, projectID, traceAID, traceBID, traceAID, traceBID)
	return d, err
}