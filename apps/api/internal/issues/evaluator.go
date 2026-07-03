package issues

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/blackbox-agentdiff/api/internal/model"
	"github.com/blackbox-agentdiff/api/internal/store"
	"github.com/google/uuid"
)

type Evaluator string

const (
	EvaluatorEmptyToolResult      Evaluator = "empty_tool_result"
	EvaluatorToolCallError        Evaluator = "tool_call_error"
	EvaluatorLLMError             Evaluator = "llm_error"
	EvaluatorTokenBudget          Evaluator = "token_budget_exceeded"
	EvaluatorSlowTrace            Evaluator = "slow_trace"
	EvaluatorRepeatedToolCall     Evaluator = "repeated_tool_call"
	EvaluatorEmptyLLMOutput       Evaluator = "empty_llm_output"
	EvaluatorOutputFormatMismatch Evaluator = "output_format_mismatch"
)

const (
	SlowTraceThresholdMs = 30000
	TokenBudgetThreshold = 100000
)

type EvaluatorFunc func(ctx context.Context, trace model.Trace, spans []model.Span, stats model.TraceStats) ([]IssueFinding, error)

type IssueFinding struct {
	Evaluator Evaluator
	Severity  model.IssueSeverity
	Title     string
	Evidence  map[string]interface{}
}

func EmptyToolResult(ctx context.Context, trace model.Trace, spans []model.Span, stats model.TraceStats) ([]IssueFinding, error) {
	var findings []IssueFinding
	for _, sp := range spans {
		if sp.SpanKind != "tool" {
			continue
		}
		var attrs map[string]interface{}
		if sp.Attributes != "" {
			json.Unmarshal([]byte(sp.Attributes), &attrs)
		}
		output, _ := attrs["tool.output"].(string)
		var outputVal interface{}
		json.Unmarshal([]byte(output), &outputVal)
		switch v := outputVal.(type) {
		case string:
			if v == "" || len(v) == 0 {
				findings = append(findings, IssueFinding{
					Evaluator: EvaluatorEmptyToolResult,
					Severity:  model.IssueSeverityMedium,
					Title:     fmt.Sprintf("Tool '%s' returned empty result", sp.Name),
					Evidence:  map[string]interface{}{"span_id": sp.SpanID, "span_name": sp.Name},
				})
			}
		case []interface{}:
			if len(v) == 0 {
				findings = append(findings, IssueFinding{
					Evaluator: EvaluatorEmptyToolResult,
					Severity:  model.IssueSeverityMedium,
					Title:     fmt.Sprintf("Tool '%s' returned empty array", sp.Name),
					Evidence:  map[string]interface{}{"span_id": sp.SpanID, "span_name": sp.Name},
				})
			}
		}
	}
	return findings, nil
}

func ToolCallError(ctx context.Context, trace model.Trace, spans []model.Span, stats model.TraceStats) ([]IssueFinding, error) {
	var findings []IssueFinding
	for _, sp := range spans {
		if sp.SpanKind != "tool" {
			continue
		}
		if sp.Status == "error" || sp.Status == "ERROR" {
			findings = append(findings, IssueFinding{
				Evaluator: EvaluatorToolCallError,
				Severity:  model.IssueSeverityHigh,
				Title:     fmt.Sprintf("Tool '%s' returned error", sp.Name),
				Evidence:  map[string]interface{}{"span_id": sp.SpanID, "span_name": sp.Name, "status": sp.Status},
			})
		}
	}
	return findings, nil
}

func LLMError(ctx context.Context, trace model.Trace, spans []model.Span, stats model.TraceStats) ([]IssueFinding, error) {
	var findings []IssueFinding
	for _, sp := range spans {
		if sp.SpanKind != "generation" {
			continue
		}
		if sp.Status == "error" || sp.Status == "ERROR" {
			findings = append(findings, IssueFinding{
				Evaluator: EvaluatorLLMError,
				Severity:  model.IssueSeverityHigh,
				Title:     "LLM call returned error",
				Evidence:  map[string]interface{}{"span_id": sp.SpanID, "span_name": sp.Name},
			})
		}
	}
	return findings, nil
}

func TokenBudget(ctx context.Context, trace model.Trace, spans []model.Span, stats model.TraceStats) ([]IssueFinding, error) {
	var findings []IssueFinding
	totalTokens := int(stats.TotalTokens)
	if totalTokens > TokenBudgetThreshold {
		findings = append(findings, IssueFinding{
			Evaluator: EvaluatorTokenBudget,
			Severity:  model.IssueSeverityMedium,
			Title:     "Token budget exceeded",
			Evidence:  map[string]interface{}{"total_tokens": totalTokens, "threshold": TokenBudgetThreshold},
		})
	}
	return findings, nil
}

func SlowTrace(ctx context.Context, trace model.Trace, spans []model.Span, stats model.TraceStats) ([]IssueFinding, error) {
	var findings []IssueFinding
	if stats.TotalTokens > 0 && stats.TotalTokens < 1000 {
		if len(spans) > 0 && spans[0].DurationMs > SlowTraceThresholdMs {
			findings = append(findings, IssueFinding{
				Evaluator: EvaluatorSlowTrace,
				Severity:  model.IssueSeverityMedium,
				Title:     "Slow trace detected",
				Evidence:  map[string]interface{}{"duration_ms": spans[0].DurationMs, "threshold_ms": SlowTraceThresholdMs},
			})
		}
	}
	return findings, nil
}

func RepeatedToolCall(ctx context.Context, trace model.Trace, spans []model.Span, stats model.TraceStats) ([]IssueFinding, error) {
	var findings []IssueFinding
	toolCounts := make(map[string]int)
	for _, sp := range spans {
		if sp.SpanKind == "tool" {
			toolCounts[sp.Name]++
			if toolCounts[sp.Name] > 5 {
				findings = append(findings, IssueFinding{
					Evaluator: EvaluatorRepeatedToolCall,
					Severity:  model.IssueSeverityLow,
					Title:     fmt.Sprintf("Tool '%s' called more than 5 times", sp.Name),
					Evidence:  map[string]interface{}{"tool_name": sp.Name, "count": toolCounts[sp.Name]},
				})
			}
		}
	}
	return findings, nil
}

func EmptyLLMOutput(ctx context.Context, trace model.Trace, spans []model.Span, stats model.TraceStats) ([]IssueFinding, error) {
	var findings []IssueFinding
	for _, sp := range spans {
		if sp.SpanKind != "generation" {
			continue
		}
		var attrs map[string]interface{}
		if sp.Attributes != "" {
			json.Unmarshal([]byte(sp.Attributes), &attrs)
		}
		output, _ := attrs["gen_ai.completion"].(string)
		if output == "" || len(output) == 0 {
			findings = append(findings, IssueFinding{
				Evaluator: EvaluatorEmptyLLMOutput,
				Severity:  model.IssueSeverityMedium,
				Title:     "LLM returned empty output",
				Evidence:  map[string]interface{}{"span_id": sp.SpanID, "span_name": sp.Name},
			})
		}
	}
	return findings, nil
}

var BuiltinEvaluators = map[Evaluator]EvaluatorFunc{
	EvaluatorEmptyToolResult:  EmptyToolResult,
	EvaluatorToolCallError:    ToolCallError,
	EvaluatorLLMError:         LLMError,
	EvaluatorTokenBudget:      TokenBudget,
	EvaluatorSlowTrace:        SlowTrace,
	EvaluatorRepeatedToolCall: RepeatedToolCall,
	EvaluatorEmptyLLMOutput:   EmptyLLMOutput,
}

func RunAllEvaluators(ctx context.Context, trace model.Trace, spans []model.Span, stats model.TraceStats) ([]IssueFinding, error) {
	var findings []IssueFinding
	for _, fn := range BuiltinEvaluators {
		f, err := fn(ctx, trace, spans, stats)
		if err != nil {
			continue
		}
		findings = append(findings, f...)
	}
	return findings, nil
}

func Fingerprint(findings []IssueFinding) string {
	if len(findings) == 0 {
		return ""
	}
	keys := make([]string, len(findings))
	for i, f := range findings {
		keys[i] = string(f.Evaluator) + ":" + f.Title
	}
	return uuid.NewSHA1(uuid.Nil, []byte(fmt.Sprintf("%v", keys))).String()
}

func ProcessFindings(ctx context.Context, store store.Store, projectID, traceID string, findings []IssueFinding) error {
	for _, f := range findings {
		fp := Fingerprint([]IssueFinding{f})
		existing, _ := store.IssueGetByFingerprint(ctx, projectID, fp)
		evidenceJSON, _ := json.Marshal(f.Evidence)
		if existing.ID == "" {
			newIssue := model.IssueCreate{
				ProjectID:   projectID,
				Fingerprint: fp,
				Title:       f.Title,
				Evaluator:   string(f.Evaluator),
				Severity:    f.Severity,
				Status:      model.IssueStatusOpen,
			}
			issue, err := store.IssueCreate(ctx, newIssue)
			if err != nil {
				continue
			}
			_, _ = store.IssueOccurrenceCreate(ctx, model.IssueOccurrenceCreate{
				IssueID:  issue.ID,
				TraceID:  traceID,
				Evidence: string(evidenceJSON),
			})
		} else if existing.Status == "open" || existing.Status == "acknowledged" {
			_, _ = store.IssueOccurrenceCreate(ctx, model.IssueOccurrenceCreate{
				IssueID:  existing.ID,
				TraceID:  traceID,
				Evidence: string(evidenceJSON),
			})
		}
	}
	return nil
}
