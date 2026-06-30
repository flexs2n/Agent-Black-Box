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
	plainKey := "bx_live_" + uuid.New().String() + uuid.New().String()
	keyPrefix := plainKey[:12]
	hash, _ := bcrypt.GenerateFromPassword([]byte(plainKey), bcrypt.DefaultCost)
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

	plainKey := "bx_live_" + uuid.New().String() + uuid.New().String()
	keyPrefix := plainKey[:12]
	hash, _ := bcrypt.GenerateFromPassword([]byte(plainKey), bcrypt.DefaultCost)

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
	now := time.Now()

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
		TraceID:   traceID,
		ProjectID: project.ID,
		TotalSpans: 1,
		CreatedAt: now,
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
		CreatedAt:       time.Now(),
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

func int64Ptr(n int64) *int64 {
	return &n
}

func stringPtr(s string) *string {
	return &s
}
