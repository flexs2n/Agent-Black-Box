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

type PostgresStore struct {
	db *sqlx.DB
}

func NewPostgresStore(databaseURL string) (*PostgresStore, error) {
	db, err := sqlx.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) Close() error { return s.db.Close() }

func (s *PostgresStore) ProjectCreate(ctx context.Context, p model.ProjectCreate) (model.Project, error) {
	project := model.Project{
		ID:        uuid.New().String(),
		Name:      p.Name,
		Slug:      p.Slug,
		CreatedAt: model.Time{time.Now()},
	}
	if p.Settings != nil {
		b, _ := json.Marshal(p.Settings)
		project.Settings = string(b)
	}
	_, err := s.db.NamedExecContext(ctx, `INSERT INTO projects (id, name, slug, settings) VALUES (:id, :name, :slug, :settings)`, project)
	return project, err
}

func (s *PostgresStore) ProjectGet(ctx context.Context, projectID string) (model.Project, error) {
	var p model.Project
	err := s.db.GetContext(ctx, &p, `SELECT id, name, slug, created_at, settings FROM projects WHERE id = $1`, projectID)
	return p, err
}

func (s *PostgresStore) ProjectList(ctx context.Context) ([]model.Project, error) {
	var projects []model.Project
	err := s.db.SelectContext(ctx, &projects, `SELECT id, name, slug, created_at, settings FROM projects ORDER BY created_at DESC`)
	return projects, err
}

func (s *PostgresStore) ProjectUpdate(ctx context.Context, projectID string, p model.ProjectUpdate) (model.Project, error) {
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

func (s *PostgresStore) ProjectDelete(ctx context.Context, projectID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM projects WHERE id = $1`, projectID)
	return err
}

func (s *PostgresStore) APIKeyCreate(ctx context.Context, projectID, label, keyHash, keyPrefix string) (model.APIKey, error) {
	key := model.APIKey{
		ID:        uuid.New().String(),
		ProjectID: projectID,
		Label:     label,
		KeyHash:   keyHash,
		KeyPrefix: keyPrefix,
		CreatedAt: model.Time{time.Now()},
	}
	_, err := s.db.NamedExecContext(ctx, `INSERT INTO api_keys (id, project_id, label, key_hash, key_prefix) VALUES (:id, :project_id, :label, :key_hash, :key_prefix)`, key)
	return key, err
}

func (s *PostgresStore) APIKeyList(ctx context.Context, projectID string) ([]model.APIKey, error) {
	var keys []model.APIKey
	err := s.db.SelectContext(ctx, &keys, `SELECT id, project_id, label, key_hash, key_prefix, last_used_at, created_at FROM api_keys WHERE project_id = $1 ORDER BY created_at DESC`, projectID)
	return keys, err
}

func (s *PostgresStore) APIKeyGetByPrefix(ctx context.Context, prefix string) (model.APIKey, error) {
	var key model.APIKey
	err := s.db.GetContext(ctx, &key, `SELECT id, project_id, label, key_hash, key_prefix, last_used_at, created_at FROM api_keys WHERE key_prefix = $1 LIMIT 1`, prefix)
	return key, err
}

func (s *PostgresStore) APIKeyDelete(ctx context.Context, projectID, keyID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM api_keys WHERE id = $1 AND project_id = $2`, keyID, projectID)
	return err
}

func (s *PostgresStore) APIKeyMarkUsed(ctx context.Context, keyID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE api_keys SET last_used_at = $1 WHERE id = $2`, model.Time{time.Now().UTC()}, keyID)
	return err
}

func (s *PostgresStore) TraceCreate(ctx context.Context, t model.Trace) error {
	_, err := s.db.NamedExecContext(ctx, `INSERT INTO traces (id, project_id, run_id, agent_name, status, thread_id, user_id, environment, input, output, error, started_at, ended_at, duration_ms) VALUES (:id, :project_id, :run_id, :agent_name, :status, :thread_id, :user_id, :environment, :input, :output, :error, :started_at, :ended_at, :duration_ms)`, &t)
	return err
}

func (s *PostgresStore) TraceGet(ctx context.Context, traceID string) (model.Trace, error) {
	var t model.Trace
	err := s.db.GetContext(ctx, &t, `SELECT id, project_id, run_id, agent_name, status, thread_id, user_id, environment, input, output, error, started_at, ended_at, duration_ms, created_at FROM traces WHERE id = $1`, traceID)
	return t, err
}

func (s *PostgresStore) TraceGetByIDs(ctx context.Context, ids []string) ([]model.Trace, error) {
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

func (s *PostgresStore) TraceList(ctx context.Context, projectID string, filters TraceFilters) ([]model.Trace, error) {
	base := `SELECT id, project_id, run_id, agent_name, status, thread_id, user_id, environment, input, output, error, started_at, ended_at, duration_ms, created_at FROM traces WHERE project_id = $1`
	args := []any{projectID}
	order := filters.Sort
	if order == "" {
		order = "created_at DESC"
	}
	const pageSize = 50
	offset := (filters.Page - 1) * pageSize
	query := fmt.Sprintf("%s ORDER BY %s LIMIT %d OFFSET %d", base, order, pageSize, offset)
	var traces []model.Trace
	err := s.db.SelectContext(ctx, &traces, query, args...)
	return traces, err
}

func (s *PostgresStore) TraceDelete(ctx context.Context, traceID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM traces WHERE id = $1`, traceID)
	return err
}

func (s *PostgresStore) TraceDeleteBulk(ctx context.Context, traceIDs []string) error {
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

func (s *PostgresStore) TraceSearch(ctx context.Context, projectID string, filters TraceSearchFilters) ([]model.Trace, error) {
	base := `SELECT id, project_id, run_id, agent_name, status, thread_id, user_id, environment, input, output, error, started_at, ended_at, duration_ms, created_at FROM traces WHERE project_id = $1`
	args := []any{projectID}
	argIdx := 2

	if filters.AgentName != "" {
		base += fmt.Sprintf(` AND agent_name = $%d`, argIdx)
		args = append(args, filters.AgentName)
		argIdx++
	}
	if filters.Status != "" {
		base += fmt.Sprintf(` AND status = $%d`, argIdx)
		args = append(args, filters.Status)
		argIdx++
	}
	if filters.Environment != "" {
		base += fmt.Sprintf(` AND environment = $%d`, argIdx)
		args = append(args, filters.Environment)
		argIdx++
	}
	if filters.UserID != "" {
		base += fmt.Sprintf(` AND user_id = $%d`, argIdx)
		args = append(args, filters.UserID)
		argIdx++
	}
	if filters.ThreadID != "" {
		base += fmt.Sprintf(` AND thread_id = $%d`, argIdx)
		args = append(args, filters.ThreadID)
		argIdx++
	}

	base += ` ORDER BY created_at DESC LIMIT 100`
	var traces []model.Trace
	err := s.db.SelectContext(ctx, &traces, base, args...)
	return traces, err
}

func (s *PostgresStore) SpanPutBatch(ctx context.Context, spans []model.Span) error {
	if len(spans) == 0 {
		return nil
	}
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO spans (trace_id, span_id, parent_span_id, project_id, name, span_kind, status, started_at, ended_at, duration_ms, attributes) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`)
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

func (s *PostgresStore) SpanList(ctx context.Context, traceID string) ([]model.Span, error) {
	var spans []model.Span
	err := s.db.SelectContext(ctx, &spans, `SELECT trace_id, span_id, parent_span_id, project_id, name, span_kind, status, started_at, ended_at, duration_ms, attributes FROM spans WHERE trace_id = $1 ORDER BY started_at`, traceID)
	return spans, err
}

func (s *PostgresStore) TraceStatsPut(ctx context.Context, stats model.TraceStats) error {
	_, err := s.db.NamedExecContext(ctx, `INSERT INTO trace_stats (trace_id, project_id, total_spans, llm_call_count, tool_call_count, total_input_tokens, total_output_tokens, total_tokens) VALUES (:trace_id, :project_id, :total_spans, :llm_call_count, :tool_call_count, :total_input_tokens, :total_output_tokens, :total_tokens) ON CONFLICT (trace_id) DO UPDATE SET total_spans = EXCLUDED.total_spans, llm_call_count = EXCLUDED.llm_call_count, tool_call_count = EXCLUDED.tool_call_count, total_input_tokens = EXCLUDED.total_input_tokens, total_output_tokens = EXCLUDED.total_output_tokens, total_tokens = EXCLUDED.total_tokens`, stats)
	return err
}

func (s *PostgresStore) TraceStatsGet(ctx context.Context, traceID string) (model.TraceStats, error) {
	var stats model.TraceStats
	err := s.db.GetContext(ctx, &stats, `SELECT trace_id, project_id, total_spans, llm_call_count, tool_call_count, total_input_tokens, total_output_tokens, total_tokens FROM trace_stats WHERE trace_id = $1`, traceID)
	return stats, err
}

func (s *PostgresStore) BaselineCreate(ctx context.Context, b model.BaselineCreate) (model.Baseline, error) {
	baseline := model.Baseline{
		ID:        uuid.New().String(),
		ProjectID: b.ProjectID,
		TraceID:   b.TraceID,
		Label:     b.Label,
		Notes:     b.Notes,
		CreatedAt: model.Time{time.Now()},
	}
	_, err := s.db.NamedExecContext(ctx, `INSERT INTO baselines (id, project_id, trace_id, label, notes) VALUES (:id, :project_id, :trace_id, :label, :notes)`, baseline)
	return baseline, err
}

func (s *PostgresStore) BaselineList(ctx context.Context, projectID string) ([]model.Baseline, error) {
	var baselines []model.Baseline
	err := s.db.SelectContext(ctx, &baselines, `SELECT id, project_id, trace_id, label, notes, created_at FROM baselines WHERE project_id = $1 ORDER BY created_at DESC`, projectID)
	return baselines, err
}

func (s *PostgresStore) BaselineGet(ctx context.Context, projectID, label string) (model.Baseline, error) {
	var b model.Baseline
	err := s.db.GetContext(ctx, &b, `SELECT id, project_id, trace_id, label, notes, created_at FROM baselines WHERE project_id = $1 AND label = $2 LIMIT 1`, projectID, label)
	return b, err
}

func (s *PostgresStore) BaselineDelete(ctx context.Context, baselineID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM baselines WHERE id = $1`, baselineID)
	return err
}

func (s *PostgresStore) DiffPut(ctx context.Context, d model.Diff) error {
	_, err := s.db.NamedExecContext(ctx, `INSERT INTO diffs (id, project_id, trace_a_id, trace_b_id, similarity_score, diff_result) VALUES (:id, :project_id, :trace_a_id, :trace_b_id, :similarity_score, :diff_result)`, &d)
	return err
}

func (s *PostgresStore) DiffGet(ctx context.Context, diffID string) (model.Diff, error) {
	var d model.Diff
	err := s.db.GetContext(ctx, &d, `SELECT id, project_id, trace_a_id, trace_b_id, similarity_score, diff_result, created_at FROM diffs WHERE id = $1`, diffID)
	return d, err
}

func (s *PostgresStore) DiffGetByTraces(ctx context.Context, projectID, traceAID, traceBID string) (model.Diff, error) {
	var d model.Diff
	err := s.db.GetContext(ctx, &d, `SELECT id, project_id, trace_a_id, trace_b_id, similarity_score, diff_result, created_at FROM diffs WHERE project_id = $1 AND ((trace_a_id = $2 AND trace_b_id = $3) OR (trace_a_id = $4 AND trace_b_id = $5)) LIMIT 1`, projectID, traceAID, traceBID, traceAID, traceBID)
	return d, err
}

func (s *PostgresStore) IssueCreate(ctx context.Context, issue model.IssueCreate) (model.Issue, error) {
	now := model.Time{time.Now()}
	issueModel := model.Issue{
		ID:          uuid.New().String(),
		ProjectID:   issue.ProjectID,
		Fingerprint: issue.Fingerprint,
		Title:       issue.Title,
		Evaluator:   issue.Evaluator,
		Severity:    issue.Severity,
		Status:      issue.Status,
		FirstSeenAt: now,
		LastSeenAt:  now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_, err := s.db.NamedExecContext(ctx, `INSERT INTO issues (id, project_id, fingerprint, title, evaluator, severity, status, first_seen_at, last_seen_at, created_at, updated_at) VALUES (:id, :project_id, :fingerprint, :title, :evaluator, :severity, :status, :first_seen_at, :last_seen_at, :created_at, :updated_at) ON CONFLICT (project_id, fingerprint) DO NOTHING`, issueModel)
	if err != nil {
		return issueModel, err
	}
	// Check if insert succeeded (ON CONFLICT DO NOTHING may skip)
	var count int
	s.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM issues WHERE id = $1`, issueModel.ID)
	if count == 0 {
		return s.IssueGetByFingerprint(ctx, issue.ProjectID, issue.Fingerprint)
	}
	return issueModel, nil
}

func (s *PostgresStore) IssueGet(ctx context.Context, issueID string) (model.Issue, error) {
	var issue model.Issue
	err := s.db.GetContext(ctx, &issue, `SELECT id, project_id, fingerprint, title, evaluator, severity, status, first_seen_at, last_seen_at, occurrence_count, root_cause, suggested_fix, created_at, updated_at FROM issues WHERE id = $1`, issueID)
	return issue, err
}

func (s *PostgresStore) IssueList(ctx context.Context, projectID string, status model.IssueStatus) ([]model.Issue, error) {
	var issues []model.Issue
	query := `SELECT id, project_id, fingerprint, title, evaluator, severity, status, first_seen_at, last_seen_at, occurrence_count, root_cause, suggested_fix, created_at, updated_at FROM issues WHERE project_id = $1`
	args := []any{projectID}
	if status != "" {
		query += ` AND status = $2`
		args = append(args, status)
	}
	query += ` ORDER BY last_seen_at DESC`
	err := s.db.SelectContext(ctx, &issues, query, args...)
	return issues, err
}

func (s *PostgresStore) IssueUpdate(ctx context.Context, issueID string, update model.IssueUpdate) (model.Issue, error) {
	existing, err := s.IssueGet(ctx, issueID)
	if err != nil {
		return existing, err
	}
	now := model.Time{time.Now()}
	if update.Status != nil {
		existing.Status = *update.Status
		existing.UpdatedAt = now
	}
	if update.RootCause != nil {
		existing.RootCause = update.RootCause
	}
	if update.SuggestedFix != nil {
		existing.SuggestedFix = update.SuggestedFix
	}
	_, err = s.db.NamedExecContext(ctx, `UPDATE issues SET status = :status, root_cause = :root_cause, suggested_fix = :suggested_fix, updated_at = :updated_at WHERE id = :id`, existing)
	return existing, err
}

func (s *PostgresStore) IssueGetByFingerprint(ctx context.Context, projectID, fingerprint string) (model.Issue, error) {
	var issue model.Issue
	err := s.db.GetContext(ctx, &issue, `SELECT id, project_id, fingerprint, title, evaluator, severity, status, first_seen_at, last_seen_at, occurrence_count, root_cause, suggested_fix, created_at, updated_at FROM issues WHERE project_id = $1 AND fingerprint = $2 LIMIT 1`, projectID, fingerprint)
	return issue, err
}

func (s *PostgresStore) IssueOccurrenceCreate(ctx context.Context, occ model.IssueOccurrenceCreate) (model.IssueOccurrence, error) {
	occurrence := model.IssueOccurrence{
		ID:        uuid.New().String(),
		IssueID:   occ.IssueID,
		TraceID:   occ.TraceID,
		Evidence:  occ.Evidence,
		CreatedAt: model.Time{time.Now()},
	}
	_, err := s.db.NamedExecContext(ctx, `INSERT INTO issue_occurrences (id, issue_id, trace_id, evidence, created_at) VALUES (:id, :issue_id, :trace_id, :evidence, :created_at)`, occurrence)
	if err != nil {
		return occurrence, err
	}
	_, err = s.db.ExecContext(ctx, `UPDATE issues SET occurrence_count = occurrence_count + 1, last_seen_at = $1 WHERE id = $2`, occurrence.CreatedAt, occ.IssueID)
	return occurrence, err
}

func (s *PostgresStore) IssueOccurrenceList(ctx context.Context, issueID string) ([]model.IssueOccurrence, error) {
	var occurrences []model.IssueOccurrence
	err := s.db.SelectContext(ctx, &occurrences, `SELECT id, issue_id, trace_id, evidence, created_at FROM issue_occurrences WHERE issue_id = $1 ORDER BY created_at DESC`, issueID)
	return occurrences, err
}

func (s *PostgresStore) MetricCreate(ctx context.Context, metric model.MetricCreate) (model.Metric, error) {
	m := model.Metric{
		ID:          uuid.New().String(),
		ProjectID:   metric.ProjectID,
		Name:        metric.Name,
		Aggregation: metric.Aggregation,
		FilterJSON:  metric.FilterJSON,
		WindowSecs:  metric.WindowSecs,
		CreatedAt:   model.Time{time.Now()},
		UpdatedAt:   model.Time{time.Now()},
	}
	_, err := s.db.NamedExecContext(ctx, `INSERT INTO metrics (id, project_id, name, aggregation, filter_json, window_secs, created_at, updated_at) VALUES (:id, :project_id, :name, :aggregation, :filter_json, :window_secs, :created_at, :updated_at)`, m)
	return m, err
}

func (s *PostgresStore) MetricGet(ctx context.Context, metricID string) (model.Metric, error) {
	var m model.Metric
	err := s.db.GetContext(ctx, &m, `SELECT id, project_id, name, aggregation, filter_json, window_secs, created_at, updated_at FROM metrics WHERE id = $1`, metricID)
	return m, err
}

func (s *PostgresStore) MetricList(ctx context.Context, projectID string) ([]model.Metric, error) {
	var metrics []model.Metric
	err := s.db.SelectContext(ctx, &metrics, `SELECT id, project_id, name, aggregation, filter_json, window_secs, created_at, updated_at FROM metrics WHERE project_id = $1 ORDER BY created_at DESC`, projectID)
	return metrics, err
}

func (s *PostgresStore) MetricUpdate(ctx context.Context, metricID string, update model.MetricUpdate) (model.Metric, error) {
	existing, err := s.MetricGet(ctx, metricID)
	if err != nil {
		return existing, err
	}
	now := model.Time{time.Now()}
	if update.Name != nil {
		existing.Name = *update.Name
	}
	if update.Aggregation != nil {
		existing.Aggregation = *update.Aggregation
	}
	if update.FilterJSON != nil {
		existing.FilterJSON = *update.FilterJSON
	}
	if update.WindowSecs != nil {
		existing.WindowSecs = *update.WindowSecs
	}
	existing.UpdatedAt = now
	_, err = s.db.NamedExecContext(ctx, `UPDATE metrics SET name = :name, aggregation = :aggregation, filter_json = :filter_json, window_secs = :window_secs, updated_at = :updated_at WHERE id = :id`, existing)
	return existing, err
}

func (s *PostgresStore) MetricDelete(ctx context.Context, metricID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM metrics WHERE id = $1`, metricID)
	return err
}

func (s *PostgresStore) MetricEventsGet(ctx context.Context, metricID string, limit int) ([]model.MetricEvent, error) {
	var events []model.MetricEvent
	err := s.db.SelectContext(ctx, &events, `SELECT id, metric_id, project_id, value, evaluated_at, created_at FROM metric_events WHERE metric_id = $1 ORDER BY evaluated_at DESC LIMIT $2`, metricID, limit)
	return events, err
}

func (s *PostgresStore) MonitorCreate(ctx context.Context, m model.MonitorCreate) (model.Monitor, error) {
	monitor := model.Monitor{
		ID:         uuid.New().String(),
		MetricID:   m.MetricID,
		ProjectID:  m.ProjectID,
		Condition:  m.Condition,
		Threshold:  m.Threshold,
		Severity:   m.Severity,
		Status:     model.MonitorStatusOK,
		NotifyJSON: m.NotifyJSON,
		CreatedAt:  model.Time{time.Now()},
		UpdatedAt:  model.Time{time.Now()},
	}
	_, err := s.db.NamedExecContext(ctx, `INSERT INTO monitors (id, metric_id, project_id, condition, threshold, severity, status, notify_json, created_at, updated_at) VALUES (:id, :metric_id, :project_id, :condition, :threshold, :severity, :status, :notify_json, :created_at, :updated_at)`, monitor)
	return monitor, err
}

func (s *PostgresStore) MonitorGet(ctx context.Context, monitorID string) (model.Monitor, error) {
	var monitor model.Monitor
	err := s.db.GetContext(ctx, &monitor, `SELECT id, metric_id, project_id, condition, threshold, severity, status, last_fired_at, notify_json, created_at, updated_at FROM monitors WHERE id = $1`, monitorID)
	return monitor, err
}

func (s *PostgresStore) MonitorList(ctx context.Context, projectID string) ([]model.Monitor, error) {
	var monitors []model.Monitor
	err := s.db.SelectContext(ctx, &monitors, `SELECT id, metric_id, project_id, condition, threshold, severity, status, last_fired_at, notify_json, created_at, updated_at FROM monitors WHERE project_id = $1 ORDER BY created_at DESC`, projectID)
	return monitors, err
}

func (s *PostgresStore) MonitorUpdate(ctx context.Context, monitorID string, update model.MonitorUpdate) (model.Monitor, error) {
	existing, err := s.MonitorGet(ctx, monitorID)
	if err != nil {
		return existing, err
	}
	now := model.Time{time.Now()}
	if update.Condition != nil {
		existing.Condition = *update.Condition
	}
	if update.Threshold != nil {
		existing.Threshold = *update.Threshold
	}
	if update.Severity != nil {
		existing.Severity = *update.Severity
	}
	if update.Status != nil {
		existing.Status = *update.Status
	}
	if update.NotifyJSON != nil {
		existing.NotifyJSON = *update.NotifyJSON
	}
	existing.UpdatedAt = now
	_, err = s.db.NamedExecContext(ctx, `UPDATE monitors SET condition = :condition, threshold = :threshold, severity = :severity, status = :status, notify_json = :notify_json, updated_at = :updated_at WHERE id = :id`, existing)
	return existing, err
}

func (s *PostgresStore) MonitorDelete(ctx context.Context, monitorID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM monitors WHERE id = $1`, monitorID)
	return err
}

func (s *PostgresStore) MonitorSetFired(ctx context.Context, monitorID string, status model.MonitorStatus, firedAt model.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE monitors SET status = $1, last_fired_at = $2 WHERE id = $3`, string(status), firedAt, monitorID)
	return err
}

func (s *PostgresStore) IncidentCreate(ctx context.Context, incident model.Incident) (model.Incident, error) {
	_, err := s.db.NamedExecContext(ctx, `INSERT INTO incidents (id, monitor_id, project_id, status, root_cause, affected_trace_count, created_at, resolved_at) VALUES (:id, :monitor_id, :project_id, :status, :root_cause, :affected_trace_count, :created_at, :resolved_at)`, incident)
	return incident, err
}

func (s *PostgresStore) IncidentList(ctx context.Context, projectID string) ([]model.Incident, error) {
	var incidents []model.Incident
	err := s.db.SelectContext(ctx, &incidents, `SELECT id, monitor_id, project_id, status, root_cause, affected_trace_count, created_at, resolved_at FROM incidents WHERE project_id = $1 ORDER BY created_at DESC`, projectID)
	return incidents, err
}

func (s *PostgresStore) IncidentGet(ctx context.Context, incidentID string) (model.Incident, error) {
	var incident model.Incident
	err := s.db.GetContext(ctx, &incident, `SELECT id, monitor_id, project_id, status, root_cause, affected_trace_count, created_at, resolved_at FROM incidents WHERE id = $1`, incidentID)
	return incident, err
}

func (s *PostgresStore) IncidentUpdate(ctx context.Context, incidentID string, update model.IncidentUpdate) (model.Incident, error) {
	existing, err := s.IncidentGet(ctx, incidentID)
	if err != nil {
		return existing, err
	}
	if update.Status != nil {
		existing.Status = *update.Status
	}
	if update.RootCause != nil {
		existing.RootCause = update.RootCause
	}
	if update.ResolvedAt != nil {
		existing.ResolvedAt = update.ResolvedAt
	}
	_, err = s.db.NamedExecContext(ctx, `UPDATE incidents SET status = :status, root_cause = :root_cause, resolved_at = :resolved_at WHERE id = :id`, existing)
	return existing, err
}

func (s *PostgresStore) IncidentListByMonitor(ctx context.Context, monitorID string) ([]model.Incident, error) {
	var incidents []model.Incident
	err := s.db.SelectContext(ctx, &incidents, `SELECT id, monitor_id, project_id, status, root_cause, affected_trace_count, created_at, resolved_at FROM incidents WHERE monitor_id = $1 ORDER BY created_at DESC`, monitorID)
	return incidents, err
}

func (s *PostgresStore) WebhookCreate(ctx context.Context, webhook model.WebhookCreate) (model.Webhook, error) {
	w := model.Webhook{
		ID:         uuid.New().String(),
		ProjectID:  webhook.ProjectID,
		URL:        webhook.URL,
		SecretHash: webhook.Secret,
		Events:     "[]",
		Enabled:    webhook.Enabled,
		CreatedAt:  model.Time{time.Now()},
		UpdatedAt:  model.Time{time.Now()},
	}
	eventsJSON, _ := json.Marshal(webhook.Events)
	w.Events = string(eventsJSON)
	_, err := s.db.NamedExecContext(ctx, `INSERT INTO webhooks (id, project_id, url, secret_hash, events, enabled, created_at, updated_at) VALUES (:id, :project_id, :url, :secret_hash, :events, :enabled, :created_at, :updated_at)`, w)
	return w, err
}

func (s *PostgresStore) WebhookList(ctx context.Context, projectID string) ([]model.Webhook, error) {
	var webhooks []model.Webhook
	err := s.db.SelectContext(ctx, &webhooks, `SELECT id, project_id, url, secret_hash, events, enabled, created_at, updated_at FROM webhooks WHERE project_id = $1 ORDER BY created_at DESC`, projectID)
	return webhooks, err
}

func (s *PostgresStore) WebhookGet(ctx context.Context, webhookID string) (model.Webhook, error) {
	var webhook model.Webhook
	err := s.db.GetContext(ctx, &webhook, `SELECT id, project_id, url, secret_hash, events, enabled, created_at, updated_at FROM webhooks WHERE id = $1`, webhookID)
	return webhook, err
}

func (s *PostgresStore) WebhookUpdate(ctx context.Context, webhookID string, update model.WebhookUpdate) (model.Webhook, error) {
	existing, err := s.WebhookGet(ctx, webhookID)
	if err != nil {
		return existing, err
	}
	if update.URL != nil {
		existing.URL = *update.URL
	}
	if update.Secret != nil {
		existing.SecretHash = *update.Secret
	}
	if update.Events != nil {
		eventsJSON, _ := json.Marshal(*update.Events)
		existing.Events = string(eventsJSON)
	}
	if update.Enabled != nil {
		existing.Enabled = *update.Enabled
	}
	existing.UpdatedAt = model.Time{time.Now()}
	_, err = s.db.NamedExecContext(ctx, `UPDATE webhooks SET url = :url, secret_hash = :secret_hash, events = :events, enabled = :enabled, updated_at = :updated_at WHERE id = :id`, existing)
	return existing, err
}

func (s *PostgresStore) WebhookDelete(ctx context.Context, webhookID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM webhooks WHERE id = $1`, webhookID)
	return err
}

func (s *PostgresStore) WebhookDeliveryLog(ctx context.Context, webhookID string, limit int) ([]model.WebhookDelivery, error) {
	var deliveries []model.WebhookDelivery
	err := s.db.SelectContext(ctx, &deliveries, `SELECT id, webhook_id, event, payload, status_code, response, attempt, created_at, delivered_at FROM webhook_deliveries WHERE webhook_id = $1 ORDER BY created_at DESC LIMIT $2`, webhookID, limit)
	return deliveries, err
}

func (s *PostgresStore) DashboardStats(ctx context.Context, projectID string) (model.DashboardStats, error) {
	var stats model.DashboardStats
	var total int
	var successCount int
	err := s.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM traces WHERE project_id = $1`, projectID)
	if err != nil {
		return stats, err
	}
	stats.TotalTraces = total
	if total > 0 {
		err = s.db.GetContext(ctx, &successCount, `SELECT COUNT(*) FROM traces WHERE project_id = $1 AND status = 'success'`, projectID)
		if err == nil {
			stats.SuccessRate = float64(successCount) / float64(total) * 100
		}
	}
	var result struct {
		TotalDuration int64 `db:"total"`
		TotalLLM      int   `db:"llm"`
		TotalTool     int   `db:"tool"`
		TotalIn       int   `db:"inp"`
		TotalOut      int   `db:"out"`
	}
	err = s.db.GetContext(ctx, &result, `SELECT 
		COALESCE(SUM(duration_ms), 0) as total,
		COALESCE(SUM(llm_call_count), 0) as llm,
		COALESCE(SUM(tool_call_count), 0) as tool,
		COALESCE(SUM(total_input_tokens), 0) as inp,
		COALESCE(SUM(total_output_tokens), 0) as out
		FROM trace_stats WHERE project_id = $1`, projectID)
	if err == nil {
		stats.P95LatencyMs = result.TotalDuration
		stats.TotalLLMCalls = result.TotalLLM
		stats.TotalToolCalls = result.TotalTool
		stats.TotalInputTokens = result.TotalIn
		stats.TotalOutputTokens = result.TotalOut
	}
	var openIssues int
	err = s.db.GetContext(ctx, &openIssues, `SELECT COUNT(*) FROM issues WHERE project_id = $1 AND status = 'open'`, projectID)
	if err == nil {
		stats.OpenIssues = openIssues
	}
	var activeIncidents int
	err = s.db.GetContext(ctx, &activeIncidents, `SELECT COUNT(*) FROM incidents WHERE project_id = $1 AND status IN ('unresolved', 'analyzed')`, projectID)
	if err == nil {
		stats.ActiveIncidents = activeIncidents
	}
	return stats, nil
}

func (s *PostgresStore) TracesByHour(ctx context.Context, projectID string, hours int) ([]model.TraceByHour, error) {
	var results []model.TraceByHour
	query := `SELECT to_char(created_at, 'YYYY-MM-DD HH24:00') as hour, COUNT(*) as count, 
		SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as success_count,
		SUM(CASE WHEN status = 'error' THEN 1 ELSE 0 END) as error_count,
		AVG(duration_ms) as avg_duration 
		FROM traces WHERE project_id = $1 AND created_at >= NOW() - make_interval(hours => $2) 
		GROUP BY hour ORDER BY hour DESC LIMIT $2`
	err := s.db.SelectContext(ctx, &results, query, projectID, hours)
	return results, err
}

func (s *PostgresStore) ThreadList(ctx context.Context, projectID string) ([]model.ThreadSummary, error) {
	var threads []model.ThreadSummary
	err := s.db.SelectContext(ctx, &threads, `
		SELECT
			thread_id,
			COUNT(*) AS trace_count,
			MIN(created_at) AS first_seen_at,
			MAX(created_at) AS last_seen_at,
			(SELECT agent_name FROM traces t2 WHERE t2.project_id = $1 AND t2.thread_id = t1.thread_id ORDER BY t2.created_at DESC LIMIT 1) AS last_agent_name,
			(SELECT status FROM traces t3 WHERE t3.project_id = $1 AND t3.thread_id = t1.thread_id ORDER BY t3.created_at DESC LIMIT 1) AS last_status
		FROM traces t1
		WHERE t1.project_id = $1 AND t1.thread_id IS NOT NULL
		GROUP BY t1.thread_id
		ORDER BY last_seen_at DESC
	`, projectID)
	return threads, err
}

func (s *PostgresStore) PresetMetrics(ctx context.Context, projectID string, windowSecs int) ([]model.PresetMetric, error) {
	var metrics []model.PresetMetric

	var totalTraces, successTraces, errorTraces int64
	var totalInput, totalOutput, totalTokens float64
	var traceCount int64

	s.db.GetContext(ctx, &totalTraces, `SELECT COUNT(*) FROM traces WHERE project_id = $1 AND created_at >= NOW() - make_interval(secs => $2)`, projectID, windowSecs)
	s.db.GetContext(ctx, &successTraces, `SELECT COUNT(*) FROM traces WHERE project_id = $1 AND status = 'success' AND created_at >= NOW() - make_interval(secs => $2)`, projectID, windowSecs)
	s.db.GetContext(ctx, &errorTraces, `SELECT COUNT(*) FROM traces WHERE project_id = $1 AND status = 'error' AND created_at >= NOW() - make_interval(secs => $2)`, projectID, windowSecs)

	if totalTraces > 0 {
		metrics = append(metrics, model.PresetMetric{Slug: "trace_success_rate", Name: "Trace Success Rate", Value: float64(successTraces) / float64(totalTraces) * 100, Format: "percentage"})
		metrics = append(metrics, model.PresetMetric{Slug: "error_rate", Name: "Error Rate", Value: float64(errorTraces) / float64(totalTraces) * 100, Format: "percentage"})
	}

	var totalToolSpans, okToolSpans int64
	s.db.GetContext(ctx, &totalToolSpans, `SELECT COUNT(*) FROM spans WHERE project_id = $1 AND span_kind = 'tool'`, projectID)
	s.db.GetContext(ctx, &okToolSpans, `SELECT COUNT(*) FROM spans WHERE project_id = $1 AND span_kind = 'tool' AND status = 'ok'`, projectID)
	if totalToolSpans > 0 {
		metrics = append(metrics, model.PresetMetric{Slug: "tool_call_success_rate", Name: "Tool Call Success Rate", Value: float64(okToolSpans) / float64(totalToolSpans) * 100, Format: "percentage"})
	}

	s.db.GetContext(ctx, &totalInput, `SELECT COALESCE(AVG(total_input_tokens), 0) FROM trace_stats WHERE project_id = $1 AND created_at >= NOW() - make_interval(secs => $2)`, projectID, windowSecs)
	s.db.GetContext(ctx, &totalOutput, `SELECT COALESCE(AVG(total_output_tokens), 0) FROM trace_stats WHERE project_id = $1 AND created_at >= NOW() - make_interval(secs => $2)`, projectID, windowSecs)
	s.db.GetContext(ctx, &totalTokens, `SELECT COALESCE(AVG(total_tokens), 0) FROM trace_stats WHERE project_id = $1 AND created_at >= NOW() - make_interval(secs => $2)`, projectID, windowSecs)
	metrics = append(metrics, model.PresetMetric{Slug: "avg_input_tokens", Name: "Avg Input Tokens", Value: totalInput, Format: "count"})
	metrics = append(metrics, model.PresetMetric{Slug: "avg_output_tokens", Name: "Avg Output Tokens", Value: totalOutput, Format: "count"})
	metrics = append(metrics, model.PresetMetric{Slug: "avg_total_tokens", Name: "Avg Total Tokens", Value: totalTokens, Format: "count"})

	var durations []int64
	s.db.SelectContext(ctx, &durations, `SELECT duration_ms FROM traces WHERE project_id = $1 AND duration_ms IS NOT NULL AND created_at >= NOW() - make_interval(secs => $2) ORDER BY duration_ms`, projectID, windowSecs)
	if len(durations) > 0 {
		p50 := durations[len(durations)*50/100]
		p95 := durations[len(durations)*95/100]
		p99 := durations[len(durations)*99/100]
		metrics = append(metrics, model.PresetMetric{Slug: "p50_latency", Name: "P50 Latency", Value: float64(p50), Format: "ms"})
		metrics = append(metrics, model.PresetMetric{Slug: "p95_latency", Name: "P95 Latency", Value: float64(p95), Format: "ms"})
		metrics = append(metrics, model.PresetMetric{Slug: "p99_latency", Name: "P99 Latency", Value: float64(p99), Format: "ms"})
	}

	s.db.GetContext(ctx, &traceCount, `SELECT COUNT(*) FROM traces WHERE project_id = $1 AND created_at >= NOW() - make_interval(secs => $2)`, projectID, windowSecs)
	minutes := float64(windowSecs) / 60
	if minutes > 0 {
		metrics = append(metrics, model.PresetMetric{Slug: "traces_per_minute", Name: "Traces Per Minute", Value: float64(traceCount) / minutes, Format: "rate"})
	}

	return metrics, nil
}

func (s *PostgresStore) WebhookListByEvent(ctx context.Context, projectID string, eventType string) ([]model.Webhook, error) {
	var webhooks []model.Webhook
	err := s.db.SelectContext(ctx, &webhooks, `SELECT id, project_id, url, secret_hash, events, enabled, created_at, updated_at FROM webhooks WHERE project_id = $1 AND enabled = true AND events LIKE $2`, projectID, `%"`+eventType+`%"`)
	return webhooks, err
}

func (s *PostgresStore) WebhookDeliveryCreate(ctx context.Context, d model.WebhookDelivery) error {
	_, err := s.db.NamedExecContext(ctx, `INSERT INTO webhook_deliveries (id, webhook_id, event, payload, status_code, response, attempt, created_at) VALUES (:id, :webhook_id, :event, :payload, :status_code, :response, :attempt, :created_at)`, d)
	return err
}

func (s *PostgresStore) EmbeddingPut(ctx context.Context, traceID, projectID string, embedding []float32) error {
	data, err := json.Marshal(embedding)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO trace_embeddings (trace_id, project_id, embedding, created_at) VALUES ($1, $2, $3, NOW()) ON CONFLICT (trace_id) DO UPDATE SET embedding = $3, created_at = NOW()`,
		traceID, projectID, string(data))
	return err
}

func (s *PostgresStore) EmbeddingGet(ctx context.Context, traceID string) ([]float32, error) {
	var raw string
	err := s.db.GetContext(ctx, &raw, `SELECT embedding FROM trace_embeddings WHERE trace_id = $1`, traceID)
	if err != nil {
		return nil, err
	}
	var vec []float32
	if err := json.Unmarshal([]byte(raw), &vec); err != nil {
		return nil, err
	}
	return vec, nil
}

func (s *PostgresStore) EmbeddingListByProject(ctx context.Context, projectID string) ([]model.TraceEmbedding, error) {
	var rows []struct {
		TraceID   string    `db:"trace_id"`
		ProjectID string    `db:"project_id"`
		Embedding string    `db:"embedding"`
		CreatedAt model.Time `db:"created_at"`
	}
	err := s.db.SelectContext(ctx, &rows, `SELECT trace_id, project_id, embedding, created_at FROM trace_embeddings WHERE project_id = $1`, projectID)
	if err != nil {
		return nil, err
	}
	result := make([]model.TraceEmbedding, len(rows))
	for i, r := range rows {
		result[i] = model.TraceEmbedding{
			TraceID:   r.TraceID,
			ProjectID: r.ProjectID,
			Embedding: r.Embedding,
			CreatedAt: r.CreatedAt,
		}
		json.Unmarshal([]byte(r.Embedding), &result[i].Vector)
	}
	return result, nil
}
