package store

import (
	"context"

	"github.com/blackbox-agentdiff/api/internal/model"
)

type TraceFilters struct {
	ProjectID string
	Sort      string
	Page      int
}

type TraceSearchFilters struct {
	ProjectID   string
	AgentName   string
	Status      string
	Environment string
	UserID      string
	ThreadID    string
}

type Store interface {
	Close() error

	ProjectCreate(ctx context.Context, p model.ProjectCreate) (model.Project, error)
	ProjectGet(ctx context.Context, projectID string) (model.Project, error)
	ProjectList(ctx context.Context) ([]model.Project, error)
	ProjectUpdate(ctx context.Context, projectID string, p model.ProjectUpdate) (model.Project, error)
	ProjectDelete(ctx context.Context, projectID string) error

	APIKeyCreate(ctx context.Context, projectID, label, keyHash, keyPrefix string) (model.APIKey, error)
	APIKeyList(ctx context.Context, projectID string) ([]model.APIKey, error)
	APIKeyGetByPrefix(ctx context.Context, prefix string) (model.APIKey, error)
	APIKeyDelete(ctx context.Context, projectID, keyID string) error
	APIKeyMarkUsed(ctx context.Context, keyID string) error

	TraceCreate(ctx context.Context, t model.Trace) error
	TraceGet(ctx context.Context, traceID string) (model.Trace, error)
	TraceGetByIDs(ctx context.Context, ids []string) ([]model.Trace, error)
	TraceList(ctx context.Context, projectID string, filters TraceFilters) ([]model.Trace, error)
	TraceDelete(ctx context.Context, traceID string) error
	TraceDeleteBulk(ctx context.Context, traceIDs []string) error
	TraceSearch(ctx context.Context, projectID string, filters TraceSearchFilters) ([]model.Trace, error)

	SpanPutBatch(ctx context.Context, spans []model.Span) error
	SpanList(ctx context.Context, traceID string) ([]model.Span, error)

	TraceStatsPut(ctx context.Context, stats model.TraceStats) error
	TraceStatsGet(ctx context.Context, traceID string) (model.TraceStats, error)

	BaselineCreate(ctx context.Context, b model.BaselineCreate) (model.Baseline, error)
	BaselineList(ctx context.Context, projectID string) ([]model.Baseline, error)
	BaselineGet(ctx context.Context, projectID, label string) (model.Baseline, error)
	BaselineDelete(ctx context.Context, baselineID string) error

	DiffPut(ctx context.Context, d model.Diff) error
	DiffGet(ctx context.Context, diffID string) (model.Diff, error)
	DiffGetByTraces(ctx context.Context, projectID, traceAID, traceBID string) (model.Diff, error)

	// Issues
	IssueCreate(ctx context.Context, issue model.IssueCreate) (model.Issue, error)
	IssueGet(ctx context.Context, issueID string) (model.Issue, error)
	IssueList(ctx context.Context, projectID string, status model.IssueStatus) ([]model.Issue, error)
	IssueUpdate(ctx context.Context, issueID string, update model.IssueUpdate) (model.Issue, error)
	IssueGetByFingerprint(ctx context.Context, projectID, fingerprint string) (model.Issue, error)
	IssueOccurrenceCreate(ctx context.Context, occ model.IssueOccurrenceCreate) (model.IssueOccurrence, error)
	IssueOccurrenceList(ctx context.Context, issueID string) ([]model.IssueOccurrence, error)

	// Metrics
	MetricCreate(ctx context.Context, metric model.MetricCreate) (model.Metric, error)
	MetricGet(ctx context.Context, metricID string) (model.Metric, error)
	MetricList(ctx context.Context, projectID string) ([]model.Metric, error)
	MetricUpdate(ctx context.Context, metricID string, update model.MetricUpdate) (model.Metric, error)
	MetricDelete(ctx context.Context, metricID string) error
	MetricEventsGet(ctx context.Context, metricID string, limit int) ([]model.MetricEvent, error)

	// Monitors
	MonitorCreate(ctx context.Context, m model.MonitorCreate) (model.Monitor, error)
	MonitorGet(ctx context.Context, monitorID string) (model.Monitor, error)
	MonitorList(ctx context.Context, projectID string) ([]model.Monitor, error)
	MonitorUpdate(ctx context.Context, monitorID string, update model.MonitorUpdate) (model.Monitor, error)
	MonitorDelete(ctx context.Context, monitorID string) error
	MonitorSetFired(ctx context.Context, monitorID string, status model.MonitorStatus, firedAt model.Time) error

	// Incidents
	IncidentCreate(ctx context.Context, incident model.Incident) (model.Incident, error)
	IncidentList(ctx context.Context, projectID string) ([]model.Incident, error)
	IncidentGet(ctx context.Context, incidentID string) (model.Incident, error)
	IncidentUpdate(ctx context.Context, incidentID string, update model.IncidentUpdate) (model.Incident, error)
	IncidentListByMonitor(ctx context.Context, monitorID string) ([]model.Incident, error)

	// Webhooks
	WebhookCreate(ctx context.Context, webhook model.WebhookCreate) (model.Webhook, error)
	WebhookList(ctx context.Context, projectID string) ([]model.Webhook, error)
	WebhookGet(ctx context.Context, webhookID string) (model.Webhook, error)
	WebhookUpdate(ctx context.Context, webhookID string, update model.WebhookUpdate) (model.Webhook, error)
	WebhookDelete(ctx context.Context, webhookID string) error
	WebhookDeliveryLog(ctx context.Context, webhookID string, limit int) ([]model.WebhookDelivery, error)

	// Dashboard aggregations
	DashboardStats(ctx context.Context, projectID string) (model.DashboardStats, error)
	TracesByHour(ctx context.Context, projectID string, hours int) ([]model.TraceByHour, error)

	// Threads
	ThreadList(ctx context.Context, projectID string) ([]model.ThreadSummary, error)

	// Preset metrics
	PresetMetrics(ctx context.Context, projectID string, windowSecs int) ([]model.PresetMetric, error)

	// Webhook delivery
	WebhookListByEvent(ctx context.Context, projectID string, eventType string) ([]model.Webhook, error)
	WebhookDeliveryCreate(ctx context.Context, d model.WebhookDelivery) error

	// Embeddings
	EmbeddingPut(ctx context.Context, traceID, projectID string, embedding []float32) error
	EmbeddingGet(ctx context.Context, traceID string) ([]float32, error)
	EmbeddingListByProject(ctx context.Context, projectID string) ([]model.TraceEmbedding, error)
}
