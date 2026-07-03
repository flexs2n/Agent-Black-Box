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
		CreatedAt: model.Time{time.Now()},
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
		CreatedAt: model.Time{time.Now()},
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
	now := model.Time{time.Now().UTC()}
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
		CreatedAt: model.Time{time.Now()},
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

// Issues
func (s *SQLiteStore) IssueCreate(ctx context.Context, issue model.IssueCreate) (model.Issue, error) {
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
	_, err := s.db.NamedExecContext(ctx, `INSERT INTO issues (id, project_id, fingerprint, title, evaluator, severity, status, first_seen_at, last_seen_at, created_at, updated_at) VALUES (:id, :project_id, :fingerprint, :title, :evaluator, :severity, :status, :first_seen_at, :last_seen_at, :created_at, :updated_at)`, issueModel)
	if err != nil {
		return issueModel, err
	}
	return issueModel, nil
}

func (s *SQLiteStore) IssueGet(ctx context.Context, issueID string) (model.Issue, error) {
	var issue model.Issue
	err := s.db.GetContext(ctx, &issue, `SELECT id, project_id, fingerprint, title, evaluator, severity, status, first_seen_at, last_seen_at, occurrence_count, root_cause, suggested_fix, created_at, updated_at FROM issues WHERE id = ?`, issueID)
	return issue, err
}

func (s *SQLiteStore) IssueList(ctx context.Context, projectID string, status model.IssueStatus) ([]model.Issue, error) {
	var issues []model.Issue
	query := `SELECT id, project_id, fingerprint, title, evaluator, severity, status, first_seen_at, last_seen_at, occurrence_count, root_cause, suggested_fix, created_at, updated_at FROM issues WHERE project_id = ?`
	args := []any{projectID}
	if status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY last_seen_at DESC`
	err := s.db.SelectContext(ctx, &issues, query, args...)
	return issues, err
}

func (s *SQLiteStore) IssueUpdate(ctx context.Context, issueID string, update model.IssueUpdate) (model.Issue, error) {
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

func (s *SQLiteStore) IssueGetByFingerprint(ctx context.Context, projectID, fingerprint string) (model.Issue, error) {
	var issue model.Issue
	err := s.db.GetContext(ctx, &issue, `SELECT id, project_id, fingerprint, title, evaluator, severity, status, first_seen_at, last_seen_at, occurrence_count, root_cause, suggested_fix, created_at, updated_at FROM issues WHERE project_id = ? AND fingerprint = ? LIMIT 1`, projectID, fingerprint)
	return issue, err
}

func (s *SQLiteStore) IssueOccurrenceCreate(ctx context.Context, occ model.IssueOccurrenceCreate) (model.IssueOccurrence, error) {
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
	_, err = s.db.ExecContext(ctx, `UPDATE issues SET occurrence_count = occurrence_count + 1, last_seen_at = ? WHERE id = ?`, occurrence.CreatedAt, occ.IssueID)
	return occurrence, err
}

func (s *SQLiteStore) IssueOccurrenceList(ctx context.Context, issueID string) ([]model.IssueOccurrence, error) {
	var occurrences []model.IssueOccurrence
	err := s.db.SelectContext(ctx, &occurrences, `SELECT id, issue_id, trace_id, evidence, created_at FROM issue_occurrences WHERE issue_id = ? ORDER BY created_at DESC`, issueID)
	return occurrences, err
}

// Metrics
func (s *SQLiteStore) MetricCreate(ctx context.Context, metric model.MetricCreate) (model.Metric, error) {
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

func (s *SQLiteStore) MetricGet(ctx context.Context, metricID string) (model.Metric, error) {
	var m model.Metric
	err := s.db.GetContext(ctx, &m, `SELECT id, project_id, name, aggregation, filter_json, window_secs, created_at, updated_at FROM metrics WHERE id = ?`, metricID)
	return m, err
}

func (s *SQLiteStore) MetricList(ctx context.Context, projectID string) ([]model.Metric, error) {
	var metrics []model.Metric
	err := s.db.SelectContext(ctx, &metrics, `SELECT id, project_id, name, aggregation, filter_json, window_secs, created_at, updated_at FROM metrics WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	return metrics, err
}

func (s *SQLiteStore) MetricUpdate(ctx context.Context, metricID string, update model.MetricUpdate) (model.Metric, error) {
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

func (s *SQLiteStore) MetricDelete(ctx context.Context, metricID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM metrics WHERE id = ?`, metricID)
	return err
}

func (s *SQLiteStore) MetricEventsGet(ctx context.Context, metricID string, limit int) ([]model.MetricEvent, error) {
	var events []model.MetricEvent
	err := s.db.SelectContext(ctx, &events, `SELECT id, metric_id, project_id, value, evaluated_at, created_at FROM metric_events WHERE metric_id = ? ORDER BY evaluated_at DESC LIMIT ?`, metricID, limit)
	return events, err
}

// Monitors
func (s *SQLiteStore) MonitorCreate(ctx context.Context, m model.MonitorCreate) (model.Monitor, error) {
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

func (s *SQLiteStore) MonitorGet(ctx context.Context, monitorID string) (model.Monitor, error) {
	var monitor model.Monitor
	err := s.db.GetContext(ctx, &monitor, `SELECT id, metric_id, project_id, condition, threshold, severity, status, last_fired_at, notify_json, created_at, updated_at FROM monitors WHERE id = ?`, monitorID)
	return monitor, err
}

func (s *SQLiteStore) MonitorList(ctx context.Context, projectID string) ([]model.Monitor, error) {
	var monitors []model.Monitor
	err := s.db.SelectContext(ctx, &monitors, `SELECT id, metric_id, project_id, condition, threshold, severity, status, last_fired_at, notify_json, created_at, updated_at FROM monitors WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	return monitors, err
}

func (s *SQLiteStore) MonitorUpdate(ctx context.Context, monitorID string, update model.MonitorUpdate) (model.Monitor, error) {
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

func (s *SQLiteStore) MonitorDelete(ctx context.Context, monitorID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM monitors WHERE id = ?`, monitorID)
	return err
}

// Incidents
func (s *SQLiteStore) IncidentCreate(ctx context.Context, incident model.Incident) (model.Incident, error) {
	_, err := s.db.NamedExecContext(ctx, `INSERT INTO incidents (id, monitor_id, project_id, status, root_cause, affected_trace_count, created_at, resolved_at) VALUES (:id, :monitor_id, :project_id, :status, :root_cause, :affected_trace_count, :created_at, :resolved_at)`, incident)
	return incident, err
}

func (s *SQLiteStore) IncidentList(ctx context.Context, projectID string) ([]model.Incident, error) {
	var incidents []model.Incident
	err := s.db.SelectContext(ctx, &incidents, `SELECT id, monitor_id, project_id, status, root_cause, affected_trace_count, created_at, resolved_at FROM incidents WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	return incidents, err
}

func (s *SQLiteStore) IncidentGet(ctx context.Context, incidentID string) (model.Incident, error) {
	var incident model.Incident
	err := s.db.GetContext(ctx, &incident, `SELECT id, monitor_id, project_id, status, root_cause, affected_trace_count, created_at, resolved_at FROM incidents WHERE id = ?`, incidentID)
	return incident, err
}

func (s *SQLiteStore) IncidentUpdate(ctx context.Context, incidentID string, update model.IncidentUpdate) (model.Incident, error) {
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
	_, err = s.db.NamedExecContext(ctx, `UPDATE incidents SET status = :status, root_cause = :root_cause WHERE id = :id`, existing)
	return existing, err
}

// Webhooks
func (s *SQLiteStore) WebhookCreate(ctx context.Context, webhook model.WebhookCreate) (model.Webhook, error) {
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

func (s *SQLiteStore) WebhookList(ctx context.Context, projectID string) ([]model.Webhook, error) {
	var webhooks []model.Webhook
	err := s.db.SelectContext(ctx, &webhooks, `SELECT id, project_id, url, secret_hash, events, enabled, created_at, updated_at FROM webhooks WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	return webhooks, err
}

func (s *SQLiteStore) WebhookGet(ctx context.Context, webhookID string) (model.Webhook, error) {
	var webhook model.Webhook
	err := s.db.GetContext(ctx, &webhook, `SELECT id, project_id, url, secret_hash, events, enabled, created_at, updated_at FROM webhooks WHERE id = ?`, webhookID)
	return webhook, err
}

func (s *SQLiteStore) WebhookUpdate(ctx context.Context, webhookID string, update model.WebhookUpdate) (model.Webhook, error) {
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

func (s *SQLiteStore) WebhookDelete(ctx context.Context, webhookID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM webhooks WHERE id = ?`, webhookID)
	return err
}

func (s *SQLiteStore) WebhookDeliveryLog(ctx context.Context, webhookID string, limit int) ([]model.WebhookDelivery, error) {
	var deliveries []model.WebhookDelivery
	err := s.db.SelectContext(ctx, &deliveries, `SELECT id, webhook_id, event, payload, status_code, response, attempt, created_at, delivered_at FROM webhook_deliveries WHERE webhook_id = ? ORDER BY created_at DESC LIMIT ?`, webhookID, limit)
	return deliveries, err
}

// Dashboard aggregations
func (s *SQLiteStore) DashboardStats(ctx context.Context, projectID string) (model.DashboardStats, error) {
	var stats model.DashboardStats
	var total int
	var successCount int
	err := s.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM traces WHERE project_id = ?`, projectID)
	if err != nil {
		return stats, err
	}
	stats.TotalTraces = total
	if total > 0 {
		err = s.db.GetContext(ctx, &successCount, `SELECT COUNT(*) FROM traces WHERE project_id = ? AND status = 'success'`, projectID)
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
		FROM trace_stats WHERE project_id = ?`, projectID)
	if err == nil {
		stats.P95LatencyMs = result.TotalDuration
		stats.TotalLLMCalls = result.TotalLLM
		stats.TotalToolCalls = result.TotalTool
		stats.TotalInputTokens = result.TotalIn
		stats.TotalOutputTokens = result.TotalOut
	}
	var openIssues int
	err = s.db.GetContext(ctx, &openIssues, `SELECT COUNT(*) FROM issues WHERE project_id = ? AND status = 'open'`, projectID)
	if err == nil {
		stats.OpenIssues = openIssues
	}
	var activeIncidents int
	err = s.db.GetContext(ctx, &activeIncidents, `SELECT COUNT(*) FROM incidents WHERE project_id = ? AND status IN ('unresolved', 'analyzed')`, projectID)
	if err == nil {
		stats.ActiveIncidents = activeIncidents
	}
	return stats, nil
}

func (s *SQLiteStore) TracesByHour(ctx context.Context, projectID string, hours int) ([]model.TraceByHour, error) {
	var results []model.TraceByHour
	query := `SELECT strftime('%Y-%m-%d %H:00', created_at) as hour, COUNT(*) as count, 
		SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as success_count,
		SUM(CASE WHEN status = 'error' THEN 1 ELSE 0 END) as error_count,
		AVG(duration_ms) as avg_duration 
		FROM traces WHERE project_id = ? AND created_at >= datetime('now', '-24 hours') 
		GROUP BY hour ORDER BY hour DESC LIMIT ?`
	err := s.db.SelectContext(ctx, &results, query, projectID, hours)
	return results, err
}
