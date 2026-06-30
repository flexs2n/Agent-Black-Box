package rest

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/blackbox-agentdiff/api/internal/auth"
	"github.com/blackbox-agentdiff/api/internal/diffproxy"
	"github.com/blackbox-agentdiff/api/internal/model"
	"github.com/blackbox-agentdiff/api/internal/store"
	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
)

type Handlers struct {
	store      store.Store
	diffClient *diffproxy.Client
}

func New(st store.Store, diffClient *diffproxy.Client) *Handlers {
	return &Handlers{store: st, diffClient: diffClient}
}

func (h *Handlers) Register(r chi.Router) {
	r.Get("/healthz", h.Healthz)
	r.Get("/readyz", h.Readyz)

	r.Route("/api/v1", func(api chi.Router) {
		api.Use(auth.Middleware(h.store))
		api.Get("/traces", h.ListTraces)
		api.Get("/traces/{id}", h.GetTrace)
		api.Get("/traces/{id}/spans", h.GetSpans)
		api.Delete("/traces/{id}", h.DeleteTrace)
		api.Delete("/traces", h.DeleteTraces)
		api.Post("/traces/search", h.SearchTraces)
		api.Post("/diffs", h.ComputeDiff)
		api.Get("/diffs/{id}", h.GetDiff)
		api.Post("/projects", h.CreateProject)
		api.Get("/projects", h.ListProjects)
		api.Get("/projects/{id}", h.GetProject)
		api.Post("/projects/{id}/api-keys", h.CreateAPIKey)
		api.Get("/projects/{id}/api-keys", h.ListAPIKeys)
		api.Delete("/projects/{id}/api-keys/{keyId}", h.DeleteAPIKey)
		api.Post("/baselines", h.CreateBaseline)
		api.Get("/baselines", h.ListBaselines)
		api.Delete("/baselines/{id}", h.DeleteBaseline)
	})
}

func (h *Handlers) Healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handlers) Readyz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}

func (h *Handlers) ListTraces(w http.ResponseWriter, r *http.Request) {
	projectID, ok := auth.ProjectIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	traces, err := h.store.TraceList(r.Context(), projectID, store.TraceFilters{Sort: "created_at DESC", Page: page})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(traces)
}

func (h *Handlers) GetTrace(w http.ResponseWriter, r *http.Request) {
	projectID, ok := auth.ProjectIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	traceID := chi.URLParam(r, "id")
	trace, err := h.store.TraceGet(r.Context(), traceID)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if trace.ProjectID != projectID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	spans, _ := h.store.SpanList(r.Context(), traceID)
	resp := struct {
		*model.Trace
		Spans []model.Span `json:"spans,omitempty"`
	}{Trace: &trace, Spans: spans}
	json.NewEncoder(w).Encode(resp)
}

func (h *Handlers) GetSpans(w http.ResponseWriter, r *http.Request) {
	projectID, ok := auth.ProjectIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	traceID := chi.URLParam(r, "id")
	trace, err := h.store.TraceGet(r.Context(), traceID)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if trace.ProjectID != projectID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	spans, err := h.store.SpanList(r.Context(), traceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(spans)
}

func (h *Handlers) DeleteTrace(w http.ResponseWriter, r *http.Request) {
	projectID, ok := auth.ProjectIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	traceID := chi.URLParam(r, "id")
	trace, err := h.store.TraceGet(r.Context(), traceID)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if trace.ProjectID != projectID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := h.store.TraceDelete(r.Context(), traceID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) DeleteTraces(w http.ResponseWriter, r *http.Request) {
	_, ok := auth.ProjectIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(req.IDs) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err := h.store.TraceDeleteBulk(r.Context(), req.IDs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) SearchTraces(w http.ResponseWriter, r *http.Request) {
	projectID, ok := auth.ProjectIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var filters store.TraceSearchFilters
	if err := json.NewDecoder(r.Body).Decode(&filters); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	traces, err := h.store.TraceSearch(r.Context(), projectID, filters)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(traces)
}

func (h *Handlers) ComputeDiff(w http.ResponseWriter, r *http.Request) {
	projectID, ok := auth.ProjectIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req struct {
		TraceAID      string  `json:"trace_a_id"`
		TraceBID      string  `json:"trace_b_id"`
		BaselineLabel *string `json:"baseline_label"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	traceAID := req.TraceAID
	traceBID := req.TraceBID

	if req.BaselineLabel != nil && *req.BaselineLabel != "" {
		baseline, err := h.store.BaselineGet(r.Context(), projectID, *req.BaselineLabel)
		if err != nil {
			http.Error(w, "baseline not found", http.StatusNotFound)
			return
		}
		traceBID = baseline.TraceID
	}

	if traceAID == "" || traceBID == "" {
		http.Error(w, "trace_a_id and trace_b_id (or baseline_label) required", http.StatusBadRequest)
		return
	}

	if cached, err := h.store.DiffGetByTraces(r.Context(), projectID, traceAID, traceBID); err == nil {
		json.NewEncoder(w).Encode(cached)
		return
	}

	traceA, err := h.store.TraceGet(r.Context(), traceAID)
	if err != nil {
		http.Error(w, "trace A not found", http.StatusNotFound)
		return
	}
	if traceA.ProjectID != projectID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	traceB, err := h.store.TraceGet(r.Context(), traceBID)
	if err != nil {
		http.Error(w, "trace B not found", http.StatusNotFound)
		return
	}
	if traceB.ProjectID != projectID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	spansA, err := h.store.SpanList(r.Context(), traceAID)
	if err != nil {
		http.Error(w, "failed to load spans A", http.StatusInternalServerError)
		return
	}
	spansB, err := h.store.SpanList(r.Context(), traceBID)
	if err != nil {
		http.Error(w, "failed to load spans B", http.StatusInternalServerError)
		return
	}

	treeA := buildTraceTree(spansA)
	treeB := buildTraceTree(spansB)

	statsA := computeTraceStats(spansA)
	statsB := computeTraceStats(spansB)

	diffResult, err := h.diffClient.Compute(r.Context(), treeA, treeB, statsA, statsB)
	if err != nil {
		http.Error(w, "diff computation failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if traceAID != "" {
		diffResult["traceAId"] = traceAID
	}
	if traceBID != "" {
		diffResult["traceBId"] = traceBID
	}

	similarityScore, _ := diffResult["similarityScore"].(float64)
	diffResultJSON, _ := json.Marshal(diffResult)
	diffResultStr := string(diffResultJSON)

	diff := model.Diff{
		ID:              generateRandomHex(16),
		ProjectID:       projectID,
		TraceAID:        traceAID,
		TraceBID:        traceBID,
		SimilarityScore: &similarityScore,
		DiffResultJSON:  &diffResultStr,
		CreatedAt:       time.Now(),
	}
	if err := h.store.DiffPut(r.Context(), diff); err != nil {
		http.Error(w, "failed to cache diff: "+err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(diffResult)
}

func buildTraceTree(spans []model.Span) map[string]any {
	spanMap := make(map[string]map[string]any)
	var roots []map[string]any

	for _, sp := range spans {
		attrs := make(map[string]string)
		if sp.Attributes != "" {
			json.Unmarshal([]byte(sp.Attributes), &attrs)
		}

		node := map[string]any{
			"spanId":      sp.SpanID,
			"parentSpanId": sp.ParentSpanID,
			"name":         sp.Name,
			"spanKind":     sp.SpanKind,
			"attributes":   attrs,
			"startTime":    sp.StartedAt.UnixMilli(),
			"endTime":      sp.EndedAt.UnixMilli(),
			"children":     []map[string]any{},
		}
		spanMap[sp.SpanID] = node
	}

	for _, sp := range spans {
		node := spanMap[sp.SpanID]
		if sp.ParentSpanID != nil && *sp.ParentSpanID != "" {
			if parent, ok := spanMap[*sp.ParentSpanID]; ok {
				children := parent["children"].([]map[string]any)
				parent["children"] = append(children, node)
			} else {
				roots = append(roots, node)
			}
		} else {
			roots = append(roots, node)
		}
	}

	if len(roots) == 1 {
		return roots[0]
	}
	return map[string]any{
		"spanId":       "synthetic-root",
		"name":         "trace",
		"spanKind":     "root",
		"attributes":   map[string]string{},
		"startTime":    0,
		"endTime":      0,
		"children":     roots,
	}
}

func computeTraceStats(spans []model.Span) map[string]int64 {
	var totalSpans, llmCalls, toolCalls int64
	var inputTokens, outputTokens, totalDuration int64

	for _, sp := range spans {
		totalSpans++
		if sp.SpanKind == "generation" {
			llmCalls++
			var attrs map[string]string
			if sp.Attributes != "" {
				json.Unmarshal([]byte(sp.Attributes), &attrs)
			}
			if v, ok := attrs["gen_ai.usage.input_tokens"]; ok {
				if n, _ := strconv.ParseInt(v, 10, 64); n > 0 {
					inputTokens += n
				}
			}
			if v, ok := attrs["gen_ai.usage.output_tokens"]; ok {
				if n, _ := strconv.ParseInt(v, 10, 64); n > 0 {
					outputTokens += n
				}
			}
		}
		if sp.SpanKind == "tool" {
			toolCalls++
		}
		totalDuration += sp.DurationMs
	}

	return map[string]int64{
		"totalSpans":       totalSpans,
		"llmCallCount":     llmCalls,
		"toolCallCount":    toolCalls,
		"totalInputTokens": inputTokens,
		"totalOutputTokens": outputTokens,
		"totalDurationMs":  totalDuration,
	}
}

func (h *Handlers) GetDiff(w http.ResponseWriter, r *http.Request) {
	projectID, ok := auth.ProjectIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	diffID := chi.URLParam(r, "id")
	diff, err := h.store.DiffGet(r.Context(), diffID)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if diff.ProjectID != projectID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	json.NewEncoder(w).Encode(diff)
}

func (h *Handlers) CreateProject(w http.ResponseWriter, r *http.Request) {
	var req model.ProjectCreate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	project, err := h.store.ProjectCreate(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(project)
}

func (h *Handlers) ListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := h.store.ProjectList(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(projects)
}

func (h *Handlers) GetProject(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")
	project, err := h.store.ProjectGet(r.Context(), projectID)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(project)
}

func (h *Handlers) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	projectID, ok := auth.ProjectIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	keyID := chi.URLParam(r, "id")
	if keyID != projectID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var req model.APIKeyCreate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	plainKey := "bx_live_" + generateRandomHex(32)
	keyPrefix := plainKey[:12]

	hash, err := bcrypt.GenerateFromPassword([]byte(plainKey), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "failed to hash key", http.StatusInternalServerError)
		return
	}

	key, err := h.store.APIKeyCreate(r.Context(), projectID, req.Label, string(hash), keyPrefix)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := model.KeyCreateResponse{
		APIKey:   key,
		PlainKey: plainKey,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func generateRandomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

func (h *Handlers) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")
	keys, err := h.store.APIKeyList(r.Context(), projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(keys)
}

func (h *Handlers) DeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")
	keyID := chi.URLParam(r, "keyId")
	if err := h.store.APIKeyDelete(r.Context(), projectID, keyID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) CreateBaseline(w http.ResponseWriter, r *http.Request) {
	projectID, ok := auth.ProjectIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req model.BaselineCreate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req.ProjectID = projectID
	baseline, err := h.store.BaselineCreate(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(baseline)
}

func (h *Handlers) ListBaselines(w http.ResponseWriter, r *http.Request) {
	projectID, ok := auth.ProjectIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	baselines, err := h.store.BaselineList(r.Context(), projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(baselines)
}

func (h *Handlers) DeleteBaseline(w http.ResponseWriter, r *http.Request) {
	_, ok := auth.ProjectIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	baselineID := chi.URLParam(r, "id")
	_ = h.store.BaselineDelete(r.Context(), baselineID)
	w.WriteHeader(http.StatusNoContent)
}