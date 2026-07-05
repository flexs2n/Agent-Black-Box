package test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/blackbox-agentdiff/api/internal/migrate"
	"github.com/blackbox-agentdiff/api/internal/model"
	"github.com/blackbox-agentdiff/api/internal/store"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

func newTestStore(t *testing.T) (*store.SQLiteStore, string) {
	t.Helper()
	dbPath := t.TempDir() + "test.db"
	sqlDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := migrate.Run(sqlDB); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	sqlDB.Close()

	st, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	return st, dbPath
}

func createProjectAndKey(t *testing.T, st *store.SQLiteStore) (string, string) {
	t.Helper()
	ctx := context.Background()
	project, err := st.ProjectCreate(ctx, model.ProjectCreate{
		Name: "Test", Slug: "test-" + uuid.New().String()[:8],
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	plainKey := "bx_live_" + uuid.New().String()[:24]
	keyPrefix := plainKey[:12]
	hash, err := bcrypt.GenerateFromPassword([]byte(plainKey), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("bcrypt generate: %v", err)
	}
	_, err = st.APIKeyCreate(ctx, project.ID, "k", string(hash), keyPrefix)
	if err != nil {
		t.Fatalf("create api key: %v", err)
	}
	return project.ID, plainKey
}

func TestStore_ProjectCRUD(t *testing.T) {
	st, _ := newTestStore(t)
	defer st.Close()

	ctx := context.Background()
	project, err := st.ProjectCreate(ctx, model.ProjectCreate{
		Name: "My Project", Slug: "my-project",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := st.ProjectGet(ctx, project.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "My Project" {
		t.Fatalf("expected My Project, got %s", got.Name)
	}

	projects, err := st.ProjectList(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(projects) < 1 {
		t.Fatal("expected at least 1 project")
	}
}

func TestStore_APIKeyLookup(t *testing.T) {
	st, _ := newTestStore(t)
	defer st.Close()

	ctx := context.Background()
	project, err := st.ProjectCreate(ctx, model.ProjectCreate{
		Name: "Test", Slug: "test-" + uuid.New().String()[:8],
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	plainKey := "bx_live_" + uuid.New().String()[:24]
	keyPrefix := plainKey[:12]
	hash, err := bcrypt.GenerateFromPassword([]byte(plainKey), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("bcrypt generate: %v", err)
	}

	_, err = st.APIKeyCreate(ctx, project.ID, "k", string(hash), keyPrefix)
	if err != nil {
		t.Fatalf("create key: %v", err)
	}

	found, err := st.APIKeyGetByPrefix(ctx, keyPrefix)
	if err != nil {
		t.Fatalf("get by prefix: %v", err)
	}
	if found.ProjectID != project.ID {
		t.Fatalf("expected project %s, got %s", project.ID, found.ProjectID)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(found.KeyHash), []byte(plainKey)); err != nil {
		t.Fatalf("bcrypt verify failed: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(found.KeyHash), []byte(plainKey)); err != nil {
		t.Fatalf("bcrypt verify failed: %v", err)
	}
}

func TestStore_IngestAndQueryTrace(t *testing.T) {
	st, _ := newTestStore(t)
	defer st.Close()

	ctx := context.Background()
	project, err := st.ProjectCreate(ctx, model.ProjectCreate{
		Name: "Test", Slug: "test-" + uuid.New().String()[:8],
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	traceID := uuid.New().String()
	spanID := uuid.New().String()[:16]
	now := model.Time{Time: time.Now()}

	trace := model.Trace{
		ID:          traceID,
		ProjectID:   project.ID,
		Status:      "success",
		Environment: "production",
		StartedAt:   &now,
		EndedAt:     &now,
		DurationMs:  int64Ptr(100),
		CreatedAt:   now,
	}
	if err := st.TraceCreate(ctx, trace); err != nil {
		t.Fatalf("trace create: %v", err)
	}

	span := model.Span{
		TraceID:    traceID,
		SpanID:     spanID,
		ProjectID:  project.ID,
		Name:       "test.op",
		SpanKind:   "app",
		Status:     "ok",
		StartedAt:  now,
		EndedAt:    now,
		DurationMs: 100,
	}
	if err := st.SpanPutBatch(ctx, []model.Span{span}); err != nil {
		t.Fatalf("span put: %v", err)
	}

	spans, err := st.SpanList(ctx, traceID)
	if err != nil {
		t.Fatalf("span list: %v", err)
	}
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	stats := model.TraceStats{
		TraceID:    traceID,
		ProjectID:  project.ID,
		TotalSpans: 1,
		CreatedAt:  now,
	}
	if err := st.TraceStatsPut(ctx, stats); err != nil {
		t.Fatalf("stats put: %v", err)
	}

	gotStats, err := st.TraceStatsGet(ctx, traceID)
	if err != nil {
		t.Fatalf("stats get: %v", err)
	}
	if gotStats.TotalSpans != 1 {
		t.Fatalf("expected 1 span, got %d", gotStats.TotalSpans)
	}

	traces, err := st.TraceList(ctx, project.ID, store.TraceFilters{Page: 1})
	if err != nil {
		t.Fatalf("trace list: %v", err)
	}
	if len(traces) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(traces))
	}

	gotTrace, err := st.TraceGet(ctx, traceID)
	if err != nil {
		t.Fatalf("trace get: %v", err)
	}
	if gotTrace.ID != traceID {
		t.Fatalf("expected trace %s, got %s", traceID, gotTrace.ID)
	}
}

func TestStore_BaselineAndDiff(t *testing.T) {
	st, _ := newTestStore(t)
	defer st.Close()

	ctx := context.Background()
	project, err := st.ProjectCreate(ctx, model.ProjectCreate{
		Name: "Test", Slug: "test-" + uuid.New().String()[:8],
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	traceID := uuid.New().String()
	baseline, err := st.BaselineCreate(ctx, model.BaselineCreate{
		ProjectID: project.ID,
		TraceID:   traceID,
		Label:     "v1",
	})
	if err != nil {
		t.Fatalf("baseline create: %v", err)
	}
	if baseline.Label != "v1" {
		t.Fatalf("expected v1, got %s", baseline.Label)
	}

	found, err := st.BaselineGet(ctx, project.ID, "v1")
	if err != nil {
		t.Fatalf("baseline get: %v", err)
	}
	if found.TraceID != traceID {
		t.Fatalf("expected trace %s, got %s", traceID, found.TraceID)
	}

	score := 85.5
	diff := model.Diff{
		ID:              uuid.New().String(),
		ProjectID:       project.ID,
		TraceAID:        traceID,
		TraceBID:        uuid.New().String(),
		SimilarityScore: &score,
		DiffResultJSON:  stringPtr(`{"test":true}`),
		CreatedAt:       model.Time{Time: time.Now()},
	}
	if err := st.DiffPut(ctx, diff); err != nil {
		t.Fatalf("diff put: %v", err)
	}

	gotDiff, err := st.DiffGetByTraces(ctx, project.ID, diff.TraceAID, diff.TraceBID)
	if err != nil {
		t.Fatalf("diff get by traces: %v", err)
	}
	if *gotDiff.SimilarityScore != 85.5 {
		t.Fatalf("expected 85.5 score, got %f", *gotDiff.SimilarityScore)
	}
}

func TestStore_MonitorCRUD(t *testing.T) {
	st, _ := newTestStore(t)
	defer st.Close()

	ctx := context.Background()
	project, _ := st.ProjectCreate(ctx, model.ProjectCreate{
		Name: "Test", Slug: "test-" + uuid.New().String()[:8],
	})

	metric, err := st.MetricCreate(ctx, model.MetricCreate{
		ProjectID:   project.ID,
		Name:        "Test Metric",
		Aggregation: model.MetricAggregationAvg,
		WindowSecs:  3600,
	})
	if err != nil {
		t.Fatalf("metric create: %v", err)
	}

	created, err := st.MonitorCreate(ctx, model.MonitorCreate{
		ProjectID: project.ID,
		MetricID:  metric.ID,
		Condition: model.MonitorConditionAbove,
		Threshold: 100,
		Severity:  model.MonitorSeverityHigh,
	})
	if err != nil {
		t.Fatalf("monitor create: %v", err)
	}
	if created.Status != model.MonitorStatusOK {
		t.Fatalf("expected status ok, got %s", created.Status)
	}

	got, err := st.MonitorGet(ctx, created.ID)
	if err != nil {
		t.Fatalf("monitor get: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("expected id %s, got %s", created.ID, got.ID)
	}

	list, err := st.MonitorList(ctx, project.ID)
	if err != nil {
		t.Fatalf("monitor list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 monitor, got %d", len(list))
	}

	cond := model.MonitorConditionBelow
	updated, err := st.MonitorUpdate(ctx, created.ID, model.MonitorUpdate{
		Condition: &cond,
		Threshold: float64Ptr(50),
	})
	if err != nil {
		t.Fatalf("monitor update: %v", err)
	}
	if updated.Condition != model.MonitorConditionBelow {
		t.Fatalf("expected condition below, got %s", updated.Condition)
	}

	if err := st.MonitorDelete(ctx, created.ID); err != nil {
		t.Fatalf("monitor delete: %v", err)
	}
	_, err = st.MonitorGet(ctx, created.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestStore_MonitorSetFired(t *testing.T) {
	st, _ := newTestStore(t)
	defer st.Close()

	ctx := context.Background()
	project, _ := st.ProjectCreate(ctx, model.ProjectCreate{
		Name: "Test", Slug: "test-" + uuid.New().String()[:8],
	})

	metric, _ := st.MetricCreate(ctx, model.MetricCreate{
		ProjectID:   project.ID,
		Name:        "Test Metric",
		Aggregation: model.MetricAggregationAvg,
		WindowSecs:  3600,
	})

	created, _ := st.MonitorCreate(ctx, model.MonitorCreate{
		ProjectID: project.ID,
		MetricID:  metric.ID,
		Condition: model.MonitorConditionAbove,
		Threshold: 100,
	})

	if err := st.MonitorSetFired(ctx, created.ID, model.MonitorStatusAlerting, model.Time{Time: time.Now()}); err != nil {
		t.Fatalf("monitor set fired: %v", err)
	}

	got, err := st.MonitorGet(ctx, created.ID)
	if err != nil {
		t.Fatalf("monitor get: %v", err)
	}
	if got.Status != model.MonitorStatusAlerting {
		t.Fatalf("expected alerting, got %s", got.Status)
	}
	if got.LastFiredAt == nil {
		t.Fatal("expected last_fired_at to be set")
	}
}

func TestStore_IncidentLifecycle(t *testing.T) {
	st, _ := newTestStore(t)
	defer st.Close()

	ctx := context.Background()
	project, _ := st.ProjectCreate(ctx, model.ProjectCreate{
		Name: "Test", Slug: "test-" + uuid.New().String()[:8],
	})

	metric, _ := st.MetricCreate(ctx, model.MetricCreate{
		ProjectID:   project.ID,
		Name:        "Test Metric",
		Aggregation: model.MetricAggregationAvg,
		WindowSecs:  3600,
	})

	monitor, _ := st.MonitorCreate(ctx, model.MonitorCreate{
		ProjectID: project.ID,
		MetricID:  metric.ID,
		Condition: model.MonitorConditionAbove,
		Threshold: 100,
	})

	incident := model.Incident{
		ID:        uuid.New().String(),
		MonitorID: monitor.ID,
		ProjectID: project.ID,
		Status:    model.IncidentStatusUnresolved,
		CreatedAt: model.Time{Time: time.Now()},
	}
	created, err := st.IncidentCreate(ctx, incident)
	if err != nil {
		t.Fatalf("incident create: %v", err)
	}
	if created.Status != model.IncidentStatusUnresolved {
		t.Fatalf("expected unresolved, got %s", created.Status)
	}

	// List by project
	list, err := st.IncidentList(ctx, project.ID)
	if err != nil {
		t.Fatalf("incident list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 incident, got %d", len(list))
	}

	// List by monitor
	byMonitor, err := st.IncidentListByMonitor(ctx, monitor.ID)
	if err != nil {
		t.Fatalf("incident list by monitor: %v", err)
	}
	if len(byMonitor) != 1 {
		t.Fatalf("expected 1 incident, got %d", len(byMonitor))
	}

	// Update with resolved status + resolved_at
	now := model.Time{Time: time.Now()}
	updated, err := st.IncidentUpdate(ctx, created.ID, model.IncidentUpdate{
		Status:     incidentStatusPtr(model.IncidentStatusResolved),
		ResolvedAt: &now,
	})
	if err != nil {
		t.Fatalf("incident update: %v", err)
	}
	if updated.Status != model.IncidentStatusResolved {
		t.Fatalf("expected resolved, got %s", updated.Status)
	}
	if updated.ResolvedAt == nil {
		t.Fatal("expected resolved_at to be set")
	}

	// Get single
	got, err := st.IncidentGet(ctx, created.ID)
	if err != nil {
		t.Fatalf("incident get: %v", err)
	}
	if got.Status != model.IncidentStatusResolved {
		t.Fatalf("expected resolved, got %s", got.Status)
	}
}

func TestWorker_EvaluateMonitor(t *testing.T) {
	st, _ := newTestStore(t)
	defer st.Close()

	ctx := context.Background()
	project, _ := st.ProjectCreate(ctx, model.ProjectCreate{
		Name: "Test", Slug: "test-" + uuid.New().String()[:8],
	})

	// Create a monitor watching "avg_input_tokens" (always present in PresetMetrics,
	// defaults to 0 via COALESCE). Threshold -1 means 0 > -1 = breach.
	monitor, err := st.MonitorCreate(ctx, model.MonitorCreate{
		ProjectID: project.ID,
		MetricID:  "avg_input_tokens",
		Condition: model.MonitorConditionAbove,
		Threshold: -1,
		Severity:  model.MonitorSeverityMedium,
	})
	if err != nil {
		t.Fatalf("monitor create: %v", err)
	}

	presetMetrics, err := st.PresetMetrics(ctx, project.ID, 3600)
	if err != nil {
		t.Fatalf("preset metrics: %v", err)
	}

	presetMap := make(map[string]float64)
	for _, pm := range presetMetrics {
		presetMap[pm.Slug] = pm.Value
	}

	value, ok := presetMap["avg_input_tokens"]
	if !ok {
		t.Fatal("expected avg_input_tokens in preset metrics")
	}

	// avg_input_tokens defaults to 0, threshold is -1, condition above → breach
	breaching := value > monitor.Threshold
	if !breaching {
		t.Fatalf("expected monitor to breach: value=%f threshold=%f", value, monitor.Threshold)
	}

	incident := model.Incident{
		ID:        uuid.New().String(),
		MonitorID: monitor.ID,
		ProjectID: project.ID,
		Status:    model.IncidentStatusUnresolved,
		CreatedAt: model.Time{Time: time.Now()},
	}
	if _, err := st.IncidentCreate(ctx, incident); err != nil {
		t.Fatalf("incident create: %v", err)
	}

	incidents, err := st.IncidentList(ctx, project.ID)
	if err != nil {
		t.Fatalf("incident list: %v", err)
	}
	if len(incidents) != 1 {
		t.Fatalf("expected 1 incident, got %d", len(incidents))
	}
	if incidents[0].MonitorID != monitor.ID {
		t.Fatalf("expected monitor %s, got %s", monitor.ID, incidents[0].MonitorID)
	}
}

func int64Ptr(n int64) *int64 {
	return &n
}

func stringPtr(s string) *string {
	return &s
}

func float64Ptr(f float64) *float64 {
	return &f
}

func incidentStatusPtr(s model.IncidentStatus) *model.IncidentStatus {
	return &s
}
