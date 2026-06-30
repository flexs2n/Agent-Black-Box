package ingest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/blackbox-agentdiff/api/internal/auth"
	"github.com/blackbox-agentdiff/api/internal/normalize"
	"github.com/blackbox-agentdiff/api/internal/store"
)

type Handler struct {
	store store.Store
}

func NewHandler(store store.Store) *Handler {
	return &Handler{store: store}
}

type OTLPExportRequest struct {
	ResourceSpans []ResourceSpans `json:"resourceSpans"`
}

type ResourceSpans struct {
	Resource   Resource     `json:"resource"`
	ScopeSpans []ScopeSpans `json:"scopeSpans"`
}

type Resource struct {
	Attributes []Attribute `json:"attributes"`
}

type ScopeSpans struct {
	Scope Scope  `json:"scope"`
	Spans []Span `json:"spans"`
}

type Scope struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Span struct {
	TraceID       string      `json:"traceId"`
	SpanID        string      `json:"spanId"`
	ParentSpanID  string      `json:"parentSpanId"`
	Name          string      `json:"name"`
	Kind          int32       `json:"kind"`
	StartTimeUnix string      `json:"startTimeUnixNano"`
	EndTimeUnix   string      `json:"endTimeUnixNano"`
	Attributes    []Attribute `json:"attributes"`
	Status        Status      `json:"status"`
	Events        []Event     `json:"events"`
}

type Status struct {
	Code int32 `json:"code"`
}

type Event struct {
	Name       string      `json:"name"`
	TimeUnix   string      `json:"timeUnixNano"`
	Attributes []Attribute `json:"attributes"`
}

type Attribute struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

func (h *Handler) HTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		http.Error(w, "content-type must be application/json", http.StatusBadRequest)
		return
	}

	var req OTLPExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	rawSpans := h.parseSpans(req)
	if len(rawSpans) == 0 {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	spans, trace, stats, err := normalize.NormalizeTrace(rawSpans)
	if err != nil {
		http.Error(w, "normalize failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	projectID, _ := auth.ProjectIDFromContext(r.Context())
	trace.ProjectID = projectID

	if err := h.store.TraceCreate(r.Context(), trace); err != nil {
		http.Error(w, "trace create failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.store.SpanPutBatch(r.Context(), spans); err != nil {
		http.Error(w, "span batch failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.store.TraceStatsPut(r.Context(), stats); err != nil {
		http.Error(w, "stats put failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) parseSpans(req OTLPExportRequest) []normalize.RawSpan {
	var rawSpans []normalize.RawSpan

	for _, rs := range req.ResourceSpans {
		resourceAttrs := attrsToMap(rs.Resource.Attributes)

		for _, ss := range rs.ScopeSpans {
			for _, span := range ss.Spans {
				startNano := parseNano(span.StartTimeUnix)
				endNano := parseNano(span.EndTimeUnix)

				spanAttrs := attrsToMap(span.Attributes)

				rawSpans = append(rawSpans, normalize.RawSpan{
					TraceID:       span.TraceID,
					SpanID:        span.SpanID,
					ParentSpanID:  span.ParentSpanID,
					Name:          span.Name,
					Kind:          span.Kind,
					StartTimeUnix: startNano,
					EndTimeUnix:   endNano,
					Attributes:    spanAttrs,
					StatusCode:    span.Status.Code,
					ResourceAttrs: resourceAttrs,
					ScopeName:     ss.Scope.Name,
					ScopeVersion:  ss.Scope.Version,
				})
			}
		}
	}

	return rawSpans
}

func attrsToMap(attrs []Attribute) map[string]any {
	result := make(map[string]any, len(attrs))
	for _, a := range attrs {
		result[a.Key] = a.Value
	}
	return result
}

func parseNano(s string) int64 {
	if s == "" {
		return time.Now().UnixNano()
	}
	var n int64
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return time.Now().UnixNano()
	}
	return n
}
