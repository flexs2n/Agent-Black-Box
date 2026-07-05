package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/blackbox-agentdiff/api/internal/auth"
	"github.com/blackbox-agentdiff/api/internal/diffproxy"
	"github.com/blackbox-agentdiff/api/internal/model"
	"github.com/blackbox-agentdiff/api/internal/store"
)

type Server struct {
	store      store.Store
	diffClient *diffproxy.Client
}

func NewServer(st store.Store, diffClient *diffproxy.Client) *Server {
	return &Server{store: st, diffClient: diffClient}
}

type jsonrpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonrpcResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	projectID, ok := auth.ProjectIDFromContext(r.Context())
	if !ok {
		writeError(w, nil, -32000, "unauthorized")
		return
	}

	var req jsonrpcRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, nil, -32700, "parse error")
		return
	}

	var result interface{}
	var err error

	switch req.Method {
	case "tools/list":
		result = s.handleToolsList()
	case "tools/call":
		result, err = s.handleToolCall(r.Context(), projectID, req.Params)
	default:
		writeError(w, req.ID, -32601, "method not found")
		return
	}

	if err != nil {
		writeError(w, req.ID, -32000, err.Error())
		return
	}

	writeResult(w, req.ID, result)
}

func (s *Server) handleToolsList() interface{} {
	tools := []map[string]interface{}{
		{
			"name":        "list_traces",
			"description": "List traces with optional pagination",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"page": map[string]interface{}{"type": "integer", "description": "Page number (default 1)"},
				},
			},
		},
		{
			"name":        "get_trace",
			"description": "Get a trace by ID with its spans",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"trace_id": map[string]interface{}{"type": "string", "description": "Trace ID"},
				},
				"required": []string{"trace_id"},
			},
		},
		{
			"name":        "search_traces",
			"description": "Search traces with structural filters",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"agent_name":  map[string]interface{}{"type": "string"},
					"status":      map[string]interface{}{"type": "string"},
					"environment": map[string]interface{}{"type": "string"},
					"user_id":     map[string]interface{}{"type": "string"},
					"thread_id":   map[string]interface{}{"type": "string"},
				},
			},
		},
		{
			"name":        "list_issues",
			"description": "List issues with optional status filter",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"status": map[string]interface{}{"type": "string", "description": "Filter by status (open/acknowledged/resolved/dismissed)"},
				},
			},
		},
		{
			"name":        "get_issue",
			"description": "Get an issue by ID with occurrences",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"issue_id": map[string]interface{}{"type": "string", "description": "Issue ID"},
				},
				"required": []string{"issue_id"},
			},
		},
		{
			"name":        "diff_traces",
			"description": "Compute diff between two traces",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"trace_a_id":     map[string]interface{}{"type": "string", "description": "First trace ID"},
					"trace_b_id":     map[string]interface{}{"type": "string", "description": "Second trace ID"},
					"baseline_label": map[string]interface{}{"type": "string", "description": "Optional baseline label instead of trace_b_id"},
				},
				"required": []string{"trace_a_id"},
			},
		},
		{
			"name":        "list_metrics",
			"description": "List preset and custom metrics",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name":        "get_monitor_status",
			"description": "List monitors and their current alerting status",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{},
			},
		},
	}

	return map[string]interface{}{
		"tools": tools,
	}
}

func (s *Server) handleToolCall(ctx context.Context, projectID string, raw json.RawMessage) (interface{}, error) {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, err
	}

	switch params.Name {
	case "list_traces":
		return s.listTraces(ctx, projectID, params.Arguments)
	case "get_trace":
		return s.getTrace(ctx, projectID, params.Arguments)
	case "search_traces":
		return s.searchTraces(ctx, projectID, params.Arguments)
	case "list_issues":
		return s.listIssues(ctx, projectID, params.Arguments)
	case "get_issue":
		return s.getIssue(ctx, projectID, params.Arguments)
	case "diff_traces":
		return s.diffTraces(ctx, projectID, params.Arguments)
	case "list_metrics":
		return s.listMetrics(ctx, projectID, params.Arguments)
	case "get_monitor_status":
		return s.getMonitorStatus(ctx, projectID, params.Arguments)
	default:
		return nil, nil
	}
}

func (s *Server) listTraces(ctx context.Context, projectID string, args json.RawMessage) (interface{}, error) {
	var input struct {
		Page int `json:"page"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, err
	}
	if input.Page < 1 {
		input.Page = 1
	}
	traces, err := s.store.TraceList(ctx, projectID, store.TraceFilters{Sort: "created_at DESC", Page: input.Page})
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"traces": traces}, nil
}

func (s *Server) getTrace(ctx context.Context, projectID string, args json.RawMessage) (interface{}, error) {
	var input struct {
		TraceID string `json:"trace_id"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, err
	}
	if input.TraceID == "" {
		return nil, nil
	}
	trace, err := s.store.TraceGet(ctx, input.TraceID)
	if err != nil {
		return nil, err
	}
	if trace.ProjectID != projectID {
		return nil, nil
	}
	spans, _ := s.store.SpanList(ctx, input.TraceID)
	return map[string]interface{}{
		"trace": trace,
		"spans": spans,
	}, nil
}

func (s *Server) searchTraces(ctx context.Context, projectID string, args json.RawMessage) (interface{}, error) {
	var input struct {
		AgentName   string `json:"agent_name"`
		Status      string `json:"status"`
		Environment string `json:"environment"`
		UserID      string `json:"user_id"`
		ThreadID    string `json:"thread_id"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, err
	}
	filters := store.TraceSearchFilters{
		ProjectID:   projectID,
		AgentName:   input.AgentName,
		Status:      input.Status,
		Environment: input.Environment,
		UserID:      input.UserID,
		ThreadID:    input.ThreadID,
	}
	traces, err := s.store.TraceSearch(ctx, projectID, filters)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"traces": traces}, nil
}

func (s *Server) listIssues(ctx context.Context, projectID string, args json.RawMessage) (interface{}, error) {
	var input struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, err
	}
	issues, err := s.store.IssueList(ctx, projectID, model.IssueStatus(input.Status))
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"issues": issues}, nil
}

func (s *Server) getIssue(ctx context.Context, projectID string, args json.RawMessage) (interface{}, error) {
	var input struct {
		IssueID string `json:"issue_id"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, err
	}
	if input.IssueID == "" {
		return nil, nil
	}
	issue, err := s.store.IssueGet(ctx, input.IssueID)
	if err != nil {
		return nil, err
	}
	if issue.ProjectID != projectID {
		return nil, nil
	}
	occurrences, _ := s.store.IssueOccurrenceList(ctx, input.IssueID)
	return map[string]interface{}{
		"issue":       issue,
		"occurrences": occurrences,
	}, nil
}

func (s *Server) diffTraces(ctx context.Context, projectID string, args json.RawMessage) (interface{}, error) {
	var input struct {
		TraceAID      string  `json:"trace_a_id"`
		TraceBID      string  `json:"trace_b_id"`
		BaselineLabel *string `json:"baseline_label"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, err
	}

	traceBID := input.TraceBID
	if input.BaselineLabel != nil && *input.BaselineLabel != "" {
		baseline, err := s.store.BaselineGet(ctx, projectID, *input.BaselineLabel)
		if err != nil {
			return nil, err
		}
		traceBID = baseline.TraceID
	}

	if input.TraceAID == "" || traceBID == "" {
		return nil, fmt.Errorf("trace_a_id and trace_b_id (or baseline_label) required")
	}

	if cached, err := s.store.DiffGetByTraces(ctx, projectID, input.TraceAID, traceBID); err == nil {
		return map[string]interface{}{"diff": cached}, nil
	}

	traceA, err := s.store.TraceGet(ctx, input.TraceAID)
	if err != nil {
		return nil, err
	}
	if traceA.ProjectID != projectID {
		return nil, nil
	}

	traceB, err := s.store.TraceGet(ctx, traceBID)
	if err != nil {
		return nil, err
	}
	if traceB.ProjectID != projectID {
		return nil, nil
	}

	spansA, err := s.store.SpanList(ctx, input.TraceAID)
	if err != nil {
		return nil, err
	}
	spansB, err := s.store.SpanList(ctx, traceBID)
	if err != nil {
		return nil, err
	}

	treeA := buildTraceTree(spansA)
	treeB := buildTraceTree(spansB)

	statsA := computeTraceStats(spansA)
	statsB := computeTraceStats(spansB)

	result, err := s.diffClient.Compute(ctx, treeA, treeB, statsA, statsB)
	if err != nil {
		return nil, err
	}

	result["traceAId"] = input.TraceAID
	result["traceBId"] = traceBID

	return map[string]interface{}{"diff": result}, nil
}

func (s *Server) listMetrics(ctx context.Context, projectID string, args json.RawMessage) (interface{}, error) {
	customMetrics, _ := s.store.MetricList(ctx, projectID)
	presetMetrics, _ := s.store.PresetMetrics(ctx, projectID, 3600)
	return map[string]interface{}{
		"preset": presetMetrics,
		"custom": customMetrics,
	}, nil
}

func (s *Server) getMonitorStatus(ctx context.Context, projectID string, args json.RawMessage) (interface{}, error) {
	monitors, err := s.store.MonitorList(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"monitors": monitors}, nil
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
			"spanId":       sp.SpanID,
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
		"spanId":     "synthetic-root",
		"name":       "trace",
		"spanKind":   "root",
		"attributes": map[string]string{},
		"startTime":  0,
		"endTime":    0,
		"children":   roots,
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
		"totalSpans":        totalSpans,
		"llmCallCount":      llmCalls,
		"toolCallCount":     toolCalls,
		"totalInputTokens":  inputTokens,
		"totalOutputTokens": outputTokens,
		"totalDurationMs":   totalDuration,
	}
}

func writeResult(w http.ResponseWriter, id interface{}, result interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jsonrpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	})
}

func writeError(w http.ResponseWriter, id interface{}, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jsonrpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &rpcError{Code: code, Message: message},
	})
}
