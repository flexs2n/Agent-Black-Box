package rest

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/blackbox-agentdiff/api/internal/auth"
	"github.com/blackbox-agentdiff/api/internal/model"
	"github.com/blackbox-agentdiff/api/internal/store"
	"github.com/go-chi/chi/v5"
)

type Handlers struct {
	store store.Store
}

func New(st store.Store) *Handlers {
	return &Handlers{store: st}
}

func (h *Handlers) Register(r chi.Router) {
	r.Get("/healthz", h.Healthz)
	r.Get("/readyz", h.Readyz)

	r.Group(func(api chi.Router) {
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
	_ = req.TraceBID
	if req.BaselineLabel != nil && *req.BaselineLabel != "" {
		baseline, err := h.store.BaselineGet(r.Context(), projectID, *req.BaselineLabel)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = baseline.TraceID
	}
	w.WriteHeader(http.StatusNotImplemented)
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
	w.WriteHeader(http.StatusNotImplemented)
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