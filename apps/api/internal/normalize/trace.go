package normalize

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/blackbox-agentdiff/api/internal/model"
	"github.com/google/uuid"
)

type RawSpan struct {
	TraceID       string
	SpanID        string
	ParentSpanID  string
	Name          string
	Kind          int32
	StartTimeUnix int64
	EndTimeUnix   int64
	Attributes    map[string]any
	StatusCode    int32
	ResourceAttrs map[string]any
	ScopeName     string
	ScopeVersion  string
}

func NormalizeTrace(rawSpans []RawSpan) ([]model.Span, model.Trace, model.TraceStats, error) {
	if len(rawSpans) == 0 {
		return nil, model.Trace{}, model.TraceStats{}, fmt.Errorf("no spans to normalize")
	}

	traceID := rawSpans[0].TraceID
	projectID := getStringValue(rawSpans[0].ResourceAttrs, "service.name")
	if projectID == "" {
		projectID = "default"
	}

	spanMap := make(map[string]*model.Span)
	childrenMap := make(map[string][]string)

	for _, rs := range rawSpans {
		depth := calculateDepth(rs.SpanID, rawSpans)
		spanKind := DeriveSpanKind(rs.Attributes, depth, rs.Name)

		attrs := getAllStringAttributes(rs.Attributes)
		attrsJSON, _ := json.Marshal(attrs)

		status := "ok"
		if rs.StatusCode != 0 && rs.StatusCode != 1 {
			status = "error"
		}

		span := &model.Span{
			TraceID:      rs.TraceID,
			SpanID:       rs.SpanID,
			ParentSpanID: &rs.ParentSpanID,
			ProjectID:    projectID,
			Name:         rs.Name,
			SpanKind:     spanKind,
			Status:       status,
			StartedAt:    model.Time{Time: time.Unix(0, rs.StartTimeUnix)},
			EndedAt:      model.Time{Time: time.Unix(0, rs.EndTimeUnix)},
			DurationMs:   rs.EndTimeUnix - rs.StartTimeUnix,
			Attributes:   string(attrsJSON),
		}

		if rs.ParentSpanID != "" {
			childrenMap[rs.ParentSpanID] = append(childrenMap[rs.ParentSpanID], rs.SpanID)
		}

		spanMap[rs.SpanID] = span
	}

	rootSpan := findRootSpan(rawSpans)
	startedAt := model.Time{Time: time.Unix(0, rootSpan.StartTimeUnix)}
	endedAt := model.Time{Time: time.Unix(0, rootSpan.EndTimeUnix)}
	durationMs := rootSpan.EndTimeUnix - rootSpan.StartTimeUnix

	runID := getStringValue(rootSpan.ResourceAttrs, "run.id")
	agentName := getStringValue(rootSpan.ResourceAttrs, "agent.name")
	threadID := getStringValue(rootSpan.ResourceAttrs, "thread.id")
	userID := getStringValue(rootSpan.ResourceAttrs, "user.id")
	environment := getStringValue(rootSpan.ResourceAttrs, "deployment.environment")
	if environment == "" {
		environment = "production"
	}

	inputJSON := getStringValue(rootSpan.Attributes, "gen_ai.prompt")
	outputJSON := getStringValue(rootSpan.Attributes, "gen_ai.completion")
	errorMsg := getStringValue(rootSpan.Attributes, "error.message")

	trace := model.Trace{
		ID:          traceID,
		ProjectID:   projectID,
		RunID:       stringPtr(runID),
		AgentName:   stringPtr(agentName),
		Status:      inferTraceStatus(rawSpans),
		ThreadID:    stringPtr(threadID),
		UserID:      stringPtr(userID),
		Environment: environment,
		Input:       stringPtr(inputJSON),
		Output:      stringPtr(outputJSON),
		Error:       stringPtr(errorMsg),
		StartedAt:   &startedAt,
		EndedAt:     &endedAt,
		DurationMs:  &durationMs,
		CreatedAt:   model.Time{Time: time.Now()},
	}

	spans := make([]model.Span, 0, len(spanMap))
	var keys []string
	for k := range spanMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		spans = append(spans, *spanMap[k])
	}

	stats := computeTraceStats(spans, traceID, projectID)

	return spans, trace, stats, nil
}

func calculateDepth(spanID string, rawSpans []RawSpan) int {
	spanMap := make(map[string]RawSpan)
	for _, rs := range rawSpans {
		spanMap[rs.SpanID] = rs
	}

	depth := 0
	current := spanMap[spanID]
	for current.ParentSpanID != "" {
		depth++
		parent, ok := spanMap[current.ParentSpanID]
		if !ok {
			break
		}
		current = parent
	}
	return depth
}

func findRootSpan(rawSpans []RawSpan) RawSpan {
	spanMap := make(map[string]RawSpan)
	for _, rs := range rawSpans {
		spanMap[rs.SpanID] = rs
	}

	for _, rs := range rawSpans {
		if rs.ParentSpanID == "" {
			return rs
		}
		if _, ok := spanMap[rs.ParentSpanID]; !ok {
			return rs
		}
	}
	return rawSpans[0]
}

func inferTraceStatus(rawSpans []RawSpan) string {
	for _, rs := range rawSpans {
		if rs.StatusCode != 0 && rs.StatusCode != 1 {
			return "error"
		}
	}
	return "success"
}

func computeTraceStats(spans []model.Span, traceID, projectID string) model.TraceStats {
	stats := model.TraceStats{
		TraceID:           traceID,
		ProjectID:         projectID,
		TotalSpans:        int16(len(spans)),
		LLMCallCount:      0,
		ToolCallCount:     0,
		TotalInputTokens:  0,
		TotalOutputTokens: 0,
		TotalTokens:       0,
		CreatedAt:         model.Time{Time: time.Now()},
	}

	for _, sp := range spans {
		switch sp.SpanKind {
		case SpanKindGeneration:
			stats.LLMCallCount++
		case SpanKindTool:
			stats.ToolCallCount++
		}

		var attrs map[string]string
		if err := json.Unmarshal([]byte(sp.Attributes), &attrs); err == nil {
			if v := attrs["gen_ai.usage.input_tokens"]; v != "" {
				if iv := parseInt(v); iv > 0 {
					stats.TotalInputTokens += int32(iv)
				}
			}
			if v := attrs["gen_ai.usage.output_tokens"]; v != "" {
				if iv := parseInt(v); iv > 0 {
					stats.TotalOutputTokens += int32(iv)
				}
			}
		}
	}
	stats.TotalTokens = stats.TotalInputTokens + stats.TotalOutputTokens

	return stats
}

func parseInt(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func generateTraceID() string {
	return uuid.New().String()
}

func generateSpanID() string {
	return uuid.New().String()[:16]
}
