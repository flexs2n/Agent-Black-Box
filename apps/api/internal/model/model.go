package model

import "time"

type Project struct {
	ID        string    `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	Slug      string    `db:"slug" json:"slug"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	Settings  string     `db:"settings" json:"settings,omitempty"`
}

type ProjectCreate struct {
	Name     string         `json:"name" validate:"required"`
	Slug     string         `json:"slug" validate:"required,slug"`
	Settings map[string]any `json:"settings" omitempty`
}

type ProjectUpdate struct {
	Name     *string        `json:"name"`
	Settings *map[string]any `json:"settings"`
}

type APIKey struct {
	ID         string    `db:"id" json:"id"`
	ProjectID  string    `db:"project_id" json:"project_id"`
	Label      string    `db:"label" json:"label"`
	KeyHash    string    `db:"key_hash" json:"-"`
	KeyPrefix  string    `db:"key_prefix" json:"key_prefix"`
	LastUsedAt *time.Time `db:"last_used_at" json:"last_used_at,omitempty"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
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
	ID           string     `db:"id" json:"id"`
	ProjectID    string     `db:"project_id" json:"project_id"`
	RunID        *string    `db:"run_id" json:"run_id,omitempty"`
	AgentName    *string    `db:"agent_name" json:"agent_name,omitempty"`
	Status       string     `db:"status" json:"status"`
	ThreadID     *string    `db:"thread_id" json:"thread_id,omitempty"`
	UserID       *string    `db:"user_id" json:"user_id,omitempty"`
	Environment  string     `db:"environment" json:"environment"`
	Input        *string    `db:"input" json:"input,omitempty"`
	Output       *string    `db:"output" json:"output,omitempty"`
	Error        *string    `db:"error" json:"error,omitempty"`
	StartedAt    *time.Time `db:"started_at" json:"started_at,omitempty"`
	EndedAt      *time.Time `db:"ended_at" json:"ended_at,omitempty"`
	DurationMs   *int64     `db:"duration_ms" json:"duration_ms,omitempty"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
}

type TraceStatus string

const (
	TraceStatusSuccess TraceStatus = "success"
	TraceStatusError   TraceStatus = "error"
	TraceStatusFlagged TraceStatus = "flagged"
)

func (s TraceStatus) String() string { return string(s) }

type Span struct {
	TraceID      string    `db:"trace_id" json:"trace_id"`
	SpanID       string    `db:"span_id" json:"span_id"`
	ParentSpanID *string   `db:"parent_span_id" json:"parent_span_id,omitempty"`
	ProjectID    string    `db:"project_id" json:"project_id"`
	Name         string    `db:"name" json:"name"`
	SpanKind     string    `db:"span_kind" json:"span_kind"`
	Status       string    `db:"status" json:"status"`
	StartedAt    time.Time `db:"started_at" json:"started_at"`
	EndedAt      time.Time `db:"ended_at" json:"ended_at"`
	DurationMs   int64     `db:"duration_ms" json:"duration_ms"`
	Attributes   string     `db:"attributes" json:"attributes"`
}

type TraceStats struct {
	TraceID         string  `db:"trace_id" json:"trace_id"`
	ProjectID       string  `db:"project_id" json:"project_id"`
	TotalSpans      int16   `db:"total_spans" json:"total_spans"`
	LLMCallCount    int16   `db:"llm_call_count" json:"llm_call_count"`
	ToolCallCount   int16   `db:"tool_call_count" json:"tool_call_count"`
	TotalInputTokens  int32 `db:"total_input_tokens"  json:"total_input_tokens"`
	TotalOutputTokens int32 `db:"total_output_tokens" json:"total_output_tokens"`
	TotalTokens       int32 `db:"total_tokens"        json:"total_tokens"`
	CreatedAt         time.Time `db:"created_at" json:"created_at"`
}

type Baseline struct {
	ID        string    `db:"id" json:"id"`
	ProjectID string    `db:"project_id" json:"project_id"`
	TraceID   string    `db:"trace_id" json:"trace_id"`
	Label     string    `db:"label" json:"label"`
	Notes     *string   `db:"notes" json:"notes,omitempty"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type BaselineCreate struct {
	ProjectID string  `json:"project_id" validate:"required"`
	TraceID   string  `json:"trace_id" validate:"required"`
	Label     string  `json:"label" validate:"required"`
	Notes     *string `json:"notes"`
}

type Diff struct {
	ID               string     `db:"id" json:"id"`
	ProjectID        string     `db:"project_id" json:"project_id"`
	TraceAID         string     `db:"trace_a_id" json:"trace_a_id"`
	TraceBID         string     `db:"trace_b_id" json:"trace_b_id"`
	SimilarityScore  *float64    `db:"similarity_score" json:"similarity_score,omitempty"`
	DiffResultJSON   *string     `db:"diff_result" json:"diff_result,omitempty"`
	CreatedAt        time.Time  `db:"created_at" json:"created_at"`
}