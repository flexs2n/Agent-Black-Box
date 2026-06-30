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
}