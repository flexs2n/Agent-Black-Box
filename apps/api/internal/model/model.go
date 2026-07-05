package model

import (
	"database/sql/driver"
	"fmt"
	"time"
)

type Time struct {
	time.Time
}

func (t *Time) Scan(value any) error {
	switch v := value.(type) {
	case time.Time:
		t.Time = v
	case string:
		formats := []string{
			time.RFC3339Nano,
			time.RFC3339,
			"2006-01-02 15:04:05.999999999-07:00",
			"2006-01-02 15:04:05.999999999",
			"2006-01-02T15:04:05.999999999-07:00",
			"2006-01-02T15:04:05.999999999",
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05",
			"2006-01-02",
		}
		for _, f := range formats {
			if parsed, err := time.Parse(f, v); err == nil {
				t.Time = parsed
				return nil
			}
		}
		return fmt.Errorf("cannot parse time string: %s", v)
	case []byte:
		return t.Scan(string(v))
	case nil:
		t.Time = time.Time{}
	default:
		return fmt.Errorf("cannot scan %T into Time", value)
	}
	return nil
}

func (t Time) Value() (driver.Value, error) {
	if t.IsZero() {
		return nil, nil
	}
	return t.Format(time.RFC3339Nano), nil
}

type Project struct {
	ID        string `db:"id" json:"id"`
	Name      string `db:"name" json:"name"`
	Slug      string `db:"slug" json:"slug"`
	CreatedAt Time   `db:"created_at" json:"created_at"`
	Settings  string `db:"settings" json:"settings,omitempty"`
}

type ProjectCreate struct {
	Name     string         `json:"name" validate:"required"`
	Slug     string         `json:"slug" validate:"required,slug"`
	Settings map[string]any `json:"settings,omitempty"`
}

type ProjectUpdate struct {
	Name     *string         `json:"name"`
	Settings *map[string]any `json:"settings"`
}

type APIKey struct {
	ID         string `db:"id" json:"id"`
	ProjectID  string `db:"project_id" json:"project_id"`
	Label      string `db:"label" json:"label"`
	KeyHash    string `db:"key_hash" json:"-"`
	KeyPrefix  string `db:"key_prefix" json:"key_prefix"`
	LastUsedAt *Time  `db:"last_used_at" json:"last_used_at,omitempty"`
	CreatedAt  Time   `db:"created_at" json:"created_at"`
}

type APIKeyCreate struct {
	ProjectID string `json:"project_id" validate:"required"`
	Label     string `json:"label" validate:"required"`
}

type KeyCreateResponse struct {
	APIKey
	PlainKey string `json:"plain_key"`
}

type Trace struct {
	ID          string  `db:"id" json:"id"`
	ProjectID   string  `db:"project_id" json:"project_id"`
	RunID       *string `db:"run_id" json:"run_id,omitempty"`
	AgentName   *string `db:"agent_name" json:"agent_name,omitempty"`
	Status      string  `db:"status" json:"status"`
	ThreadID    *string `db:"thread_id" json:"thread_id,omitempty"`
	UserID      *string `db:"user_id" json:"user_id,omitempty"`
	Environment string  `db:"environment" json:"environment"`
	Input       *string `db:"input" json:"input,omitempty"`
	Output      *string `db:"output" json:"output,omitempty"`
	Error       *string `db:"error" json:"error,omitempty"`
	StartedAt   *Time   `db:"started_at" json:"started_at,omitempty"`
	EndedAt     *Time   `db:"ended_at" json:"ended_at,omitempty"`
	DurationMs  *int64  `db:"duration_ms" json:"duration_ms,omitempty"`
	CreatedAt   Time    `db:"created_at" json:"created_at"`
}

type TraceStatus string

const (
	TraceStatusSuccess TraceStatus = "success"
	TraceStatusError   TraceStatus = "error"
	TraceStatusFlagged TraceStatus = "flagged"
)

func (s TraceStatus) String() string { return string(s) }

type Span struct {
	TraceID      string  `db:"trace_id" json:"trace_id"`
	SpanID       string  `db:"span_id" json:"span_id"`
	ParentSpanID *string `db:"parent_span_id" json:"parent_span_id,omitempty"`
	ProjectID    string  `db:"project_id" json:"project_id"`
	Name         string  `db:"name" json:"name"`
	SpanKind     string  `db:"span_kind" json:"span_kind"`
	Status       string  `db:"status" json:"status"`
	StartedAt    Time    `db:"started_at" json:"started_at"`
	EndedAt      Time    `db:"ended_at" json:"ended_at"`
	DurationMs   int64   `db:"duration_ms" json:"duration_ms"`
	Attributes   string  `db:"attributes" json:"attributes"`
}

type TraceStats struct {
	TraceID           string `db:"trace_id" json:"trace_id"`
	ProjectID         string `db:"project_id" json:"project_id"`
	TotalSpans        int16  `db:"total_spans" json:"total_spans"`
	LLMCallCount      int16  `db:"llm_call_count" json:"llm_call_count"`
	ToolCallCount     int16  `db:"tool_call_count" json:"tool_call_count"`
	TotalInputTokens  int32  `db:"total_input_tokens"  json:"total_input_tokens"`
	TotalOutputTokens int32  `db:"total_output_tokens" json:"total_output_tokens"`
	TotalTokens       int32  `db:"total_tokens"        json:"total_tokens"`
	CreatedAt         Time   `db:"created_at" json:"created_at"`
}

type Baseline struct {
	ID        string  `db:"id" json:"id"`
	ProjectID string  `db:"project_id" json:"project_id"`
	TraceID   string  `db:"trace_id" json:"trace_id"`
	Label     string  `db:"label" json:"label"`
	Notes     *string `db:"notes" json:"notes,omitempty"`
	CreatedAt Time    `db:"created_at" json:"created_at"`
}

type BaselineCreate struct {
	ProjectID string  `json:"project_id" validate:"required"`
	TraceID   string  `json:"trace_id" validate:"required"`
	Label     string  `json:"label" validate:"required"`
	Notes     *string `json:"notes"`
}

type Diff struct {
	ID              string   `db:"id" json:"id"`
	ProjectID       string   `db:"project_id" json:"project_id"`
	TraceAID        string   `db:"trace_a_id" json:"trace_a_id"`
	TraceBID        string   `db:"trace_b_id" json:"trace_b_id"`
	SimilarityScore *float64 `db:"similarity_score" json:"similarity_score,omitempty"`
	DiffResultJSON  *string  `db:"diff_result" json:"diff_result,omitempty"`
	CreatedAt       Time     `db:"created_at" json:"created_at"`
}

type IssueSeverity string

const (
	IssueSeverityCritical IssueSeverity = "critical"
	IssueSeverityHigh     IssueSeverity = "high"
	IssueSeverityMedium   IssueSeverity = "medium"
	IssueSeverityLow      IssueSeverity = "low"
)

type IssueStatus string

const (
	IssueStatusOpen         IssueStatus = "open"
	IssueStatusAcknowledged IssueStatus = "acknowledged"
	IssueStatusResolved     IssueStatus = "resolved"
	IssueStatusDismissed    IssueStatus = "dismissed"
)

type Issue struct {
	ID              string        `db:"id" json:"id"`
	ProjectID       string        `db:"project_id" json:"project_id"`
	Fingerprint     string        `db:"fingerprint" json:"fingerprint"`
	Title           string        `db:"title" json:"title"`
	Evaluator       string        `db:"evaluator" json:"evaluator"`
	Severity        IssueSeverity `db:"severity" json:"severity"`
	Status          IssueStatus   `db:"status" json:"status"`
	FirstSeenAt     Time          `db:"first_seen_at" json:"first_seen_at"`
	LastSeenAt      Time          `db:"last_seen_at" json:"last_seen_at"`
	OccurrenceCount int           `db:"occurrence_count" json:"occurrence_count"`
	RootCause       *string       `db:"root_cause" json:"root_cause,omitempty"`
	SuggestedFix    *string       `db:"suggested_fix" json:"suggested_fix,omitempty"`
	CreatedAt       Time          `db:"created_at" json:"created_at"`
	UpdatedAt       Time          `db:"updated_at" json:"updated_at"`
}

type IssueOccurrence struct {
	ID        string `db:"id" json:"id"`
	IssueID   string `db:"issue_id" json:"issue_id"`
	TraceID   string `db:"trace_id" json:"trace_id"`
	Evidence  string `db:"evidence" json:"evidence"`
	CreatedAt Time   `db:"created_at" json:"created_at"`
}

type MetricAggregation string

const (
	MetricAggregationAvg   MetricAggregation = "avg"
	MetricAggregationSum   MetricAggregation = "sum"
	MetricAggregationCount MetricAggregation = "count"
	MetricAggregationP50   MetricAggregation = "p50"
	MetricAggregationP95   MetricAggregation = "p95"
	MetricAggregationP99   MetricAggregation = "p99"
	MetricAggregationRatio MetricAggregation = "ratio"
)

type Metric struct {
	ID          string            `db:"id" json:"id"`
	ProjectID   string            `db:"project_id" json:"project_id"`
	Name        string            `db:"name" json:"name"`
	Aggregation MetricAggregation `db:"aggregation" json:"aggregation"`
	FilterJSON  string            `db:"filter_json" json:"filter_json"`
	WindowSecs  int               `db:"window_secs" json:"window_secs"`
	CreatedAt   Time              `db:"created_at" json:"created_at"`
	UpdatedAt   Time              `db:"updated_at" json:"updated_at"`
}

type MonitorCondition string

const (
	MonitorConditionAbove MonitorCondition = "above"
	MonitorConditionBelow MonitorCondition = "below"
)

type MonitorSeverity string

const (
	MonitorSeverityCritical MonitorSeverity = "critical"
	MonitorSeverityHigh     MonitorSeverity = "high"
	MonitorSeverityMedium   MonitorSeverity = "medium"
	MonitorSeverityLow      MonitorSeverity = "low"
)

type MonitorStatus string

const (
	MonitorStatusOK       MonitorStatus = "ok"
	MonitorStatusAlerting MonitorStatus = "alerting"
	MonitorStatusResolved MonitorStatus = "resolved"
)

type Monitor struct {
	ID          string           `db:"id" json:"id"`
	MetricID    string           `db:"metric_id" json:"metric_id"`
	ProjectID   string           `db:"project_id" json:"project_id"`
	Condition   MonitorCondition `db:"condition" json:"condition"`
	Threshold   float64          `db:"threshold" json:"threshold"`
	Severity    MonitorSeverity  `db:"severity" json:"severity"`
	Status      MonitorStatus    `db:"status" json:"status"`
	LastFiredAt *Time            `db:"last_fired_at" json:"last_fired_at,omitempty"`
	NotifyJSON  string           `db:"notify_json" json:"notify_json"`
	CreatedAt   Time             `db:"created_at" json:"created_at"`
	UpdatedAt   Time             `db:"updated_at" json:"updated_at"`
}

type IncidentStatus string

const (
	IncidentStatusUnresolved IncidentStatus = "unresolved"
	IncidentStatusAnalyzed   IncidentStatus = "analyzed"
	IncidentStatusResolved   IncidentStatus = "resolved"
	IncidentStatusDismissed  IncidentStatus = "dismissed"
)

type Incident struct {
	ID                 string         `db:"id" json:"id"`
	MonitorID          string         `db:"monitor_id" json:"monitor_id"`
	ProjectID          string         `db:"project_id" json:"project_id"`
	Status             IncidentStatus `db:"status" json:"status"`
	RootCause          *string        `db:"root_cause" json:"root_cause,omitempty"`
	AffectedTraceCount int            `db:"affected_trace_count" json:"affected_trace_count"`
	CreatedAt          Time           `db:"created_at" json:"created_at"`
	ResolvedAt         *Time          `db:"resolved_at" json:"resolved_at,omitempty"`
}

type Webhook struct {
	ID         string `db:"id" json:"id"`
	ProjectID  string `db:"project_id" json:"project_id"`
	URL        string `db:"url" json:"url"`
	SecretHash string `db:"secret_hash" json:"-"`
	Events     string `db:"events" json:"events"`
	Enabled    bool   `db:"enabled" json:"enabled"`
	CreatedAt  Time   `db:"created_at" json:"created_at"`
	UpdatedAt  Time   `db:"updated_at" json:"updated_at"`
}

type WebhookDelivery struct {
	ID          string  `db:"id" json:"id"`
	WebhookID   string  `db:"webhook_id" json:"webhook_id"`
	Event       string  `db:"event" json:"event"`
	Payload     string  `db:"payload" json:"payload"`
	StatusCode  *int    `db:"status_code" json:"status_code,omitempty"`
	Response    *string `db:"response" json:"response,omitempty"`
	Attempt     int     `db:"attempt" json:"attempt"`
	CreatedAt   Time    `db:"created_at" json:"created_at"`
	DeliveredAt *Time   `db:"delivered_at" json:"delivered_at,omitempty"`
}

type IssueCreate struct {
	ProjectID    string        `json:"project_id" validate:"required"`
	Fingerprint  string        `json:"fingerprint" validate:"required"`
	Title        string        `json:"title" validate:"required"`
	Evaluator    string        `json:"evaluator" validate:"required"`
	Severity     IssueSeverity `json:"severity"`
	Status       IssueStatus   `json:"status"`
	RootCause    *string       `json:"root_cause"`
	SuggestedFix *string       `json:"suggested_fix"`
}

type IssueUpdate struct {
	Status       *IssueStatus `json:"status"`
	RootCause    *string      `json:"root_cause"`
	SuggestedFix *string      `json:"suggested_fix"`
}

type IssueOccurrenceCreate struct {
	IssueID  string `json:"issue_id" validate:"required"`
	TraceID  string `json:"trace_id" validate:"required"`
	Evidence string `json:"evidence"`
}

type MetricCreate struct {
	ProjectID   string            `json:"project_id" validate:"required"`
	Name        string            `json:"name" validate:"required"`
	Aggregation MetricAggregation `json:"aggregation" validate:"required"`
	FilterJSON  string            `json:"filter_json"`
	WindowSecs  int               `json:"window_secs"`
}

type MetricUpdate struct {
	Name        *string            `json:"name"`
	Aggregation *MetricAggregation `json:"aggregation"`
	FilterJSON  *string            `json:"filter_json"`
	WindowSecs  *int               `json:"window_secs"`
}

type MonitorCreate struct {
	ProjectID  string           `json:"project_id" validate:"required"`
	MetricID   string           `json:"metric_id" validate:"required"`
	Condition  MonitorCondition `json:"condition" validate:"required"`
	Threshold  float64          `json:"threshold" validate:"required"`
	Severity   MonitorSeverity  `json:"severity"`
	NotifyJSON string           `json:"notify_json"`
}

type MonitorUpdate struct {
	Condition  *MonitorCondition `json:"condition"`
	Threshold  *float64          `json:"threshold"`
	Severity   *MonitorSeverity  `json:"severity"`
	Status     *MonitorStatus    `json:"status"`
	NotifyJSON *string           `json:"notify_json"`
}

type IncidentUpdate struct {
	Status     *IncidentStatus `json:"status"`
	RootCause  *string         `json:"root_cause"`
	ResolvedAt *Time           `json:"resolved_at,omitempty"`
}

type WebhookCreate struct {
	ProjectID string   `json:"project_id" validate:"required"`
	URL       string   `json:"url" validate:"required"`
	Secret    string   `json:"secret" validate:"required"`
	Events    []string `json:"events" validate:"required"`
	Enabled   bool     `json:"enabled"`
}

type WebhookUpdate struct {
	URL     *string   `json:"url"`
	Secret  *string   `json:"secret"`
	Events  *[]string `json:"events"`
	Enabled *bool     `json:"enabled"`
}

type DashboardStats struct {
	TotalTraces       int     `json:"total_traces"`
	SuccessRate       float64 `json:"success_rate"`
	P95LatencyMs      int64   `json:"p95_latency_ms"`
	OpenIssues        int     `json:"open_issues"`
	ActiveIncidents   int     `json:"active_incidents"`
	TotalLLMCalls     int     `json:"total_llm_calls"`
	TotalToolCalls    int     `json:"total_tool_calls"`
	TotalInputTokens  int     `json:"total_input_tokens"`
	TotalOutputTokens int     `json:"total_output_tokens"`
}

type TraceByHour struct {
	Hour         string `db:"hour" json:"hour"`
	Count        int    `db:"count" json:"count"`
	SuccessCount int    `db:"success_count" json:"success_count"`
	ErrorCount   int    `db:"error_count" json:"error_count"`
	AvgDuration  int64  `db:"avg_duration" json:"avg_duration"`
}

type MetricEvent struct {
	ID          string  `db:"id" json:"id"`
	MetricID    string  `db:"metric_id" json:"metric_id"`
	ProjectID   string  `db:"project_id" json:"project_id"`
	Value       float64 `db:"value" json:"value"`
	EvaluatedAt Time    `db:"evaluated_at" json:"evaluated_at"`
	CreatedAt   Time    `db:"created_at" json:"created_at"`
}

type ThreadSummary struct {
	ThreadID      string  `db:"thread_id" json:"thread_id"`
	TraceCount    int     `db:"trace_count" json:"trace_count"`
	FirstSeenAt   Time    `db:"first_seen_at" json:"first_seen_at"`
	LastSeenAt    Time    `db:"last_seen_at" json:"last_seen_at"`
	LastAgentName *string `db:"last_agent_name" json:"last_agent_name,omitempty"`
	LastStatus    string  `db:"last_status" json:"last_status"`
}

type DashboardResponse struct {
	Stats        DashboardStats `json:"stats"`
	TracesByHour []TraceByHour  `json:"traces_by_hour"`
	OpenIssues   []Issue        `json:"open_issues"`
}

type MetricWithSparkline struct {
	Metric
	CurrentValue   float64   `json:"current_value"`
	Sparkline      []float64 `json:"sparkline"`
}

type TraceEmbedding struct {
	TraceID   string  `db:"trace_id" json:"trace_id"`
	ProjectID string  `db:"project_id" json:"project_id"`
	Embedding string  `db:"embedding" json:"-"`
	CreatedAt Time    `db:"created_at" json:"created_at"`
	Vector    []float32 `json:"-"`
}

type PresetMetric struct {
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	Value     float64   `json:"value"`
	Format    string    `json:"format"`
	Sparkline []float64 `json:"sparkline"`
}
