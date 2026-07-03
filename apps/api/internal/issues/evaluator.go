package issues

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/blackbox-agentdiff/api/internal/model"
	"github.com/blackbox-agentdiff/api/internal/store"
	"github.com/blackbox-agentdiff/api/internal/webhook"
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
	EvaluatorHallucination        Evaluator = "hallucination"
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

func OutputFormatMismatch(ctx context.Context, trace model.Trace, spans []model.Span, stats model.TraceStats) ([]IssueFinding, error) {
	var findings []IssueFinding
	for _, sp := range spans {
		if sp.SpanKind != "generation" {
			continue
		}
		var attrs map[string]interface{}
		if sp.Attributes != "" {
			json.Unmarshal([]byte(sp.Attributes), &attrs)
		}
		prompt, _ := attrs["gen_ai.prompt"].(string)
		completion, _ := attrs["gen_ai.completion"].(string)
		if prompt == "" || completion == "" {
			continue
		}

		lower := strings.ToLower(prompt)
		expectsJSON := strings.Contains(lower, "json") || strings.Contains(lower, "return a") || strings.Contains(lower, "format:")
		if !expectsJSON {
			continue
		}

		trimmed := strings.TrimSpace(completion)
		if !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[") {
			findings = append(findings, IssueFinding{
				Evaluator: EvaluatorOutputFormatMismatch,
				Severity:  model.IssueSeverityMedium,
				Title:     fmt.Sprintf("LLM output format mismatch: expected JSON but got non-JSON response"),
				Evidence: map[string]interface{}{
					"span_id":      sp.SpanID,
					"span_name":    sp.Name,
					"output_start": trimmed[:min(len(trimmed), 100)],
				},
			})
			continue
		}

		var parsed interface{}
		if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
			findings = append(findings, IssueFinding{
				Evaluator: EvaluatorOutputFormatMismatch,
				Severity:  model.IssueSeverityMedium,
				Title:     "LLM output format mismatch: invalid JSON",
				Evidence: map[string]interface{}{
					"span_id":   sp.SpanID,
					"span_name": sp.Name,
					"error":     err.Error(),
				},
			})
		}
	}
	return findings, nil
}

func Hallucination(ctx context.Context, trace model.Trace, spans []model.Span, stats model.TraceStats) ([]IssueFinding, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, nil
	}

	var findings []IssueFinding
	for _, sp := range spans {
		if sp.SpanKind != "generation" {
			continue
		}
		var attrs map[string]interface{}
		if sp.Attributes != "" {
			json.Unmarshal([]byte(sp.Attributes), &attrs)
		}
		prompt, _ := attrs["gen_ai.prompt"].(string)
		completion, _ := attrs["gen_ai.completion"].(string)
		if prompt == "" || completion == "" {
			continue
		}

		body := map[string]interface{}{
			"model": "gpt-4o-mini",
			"messages": []map[string]string{
				{"role": "system", "content": "You are an evaluator. Determine if the following assistant response contains hallucinations (statements not supported by the given context or prompt). Reply with a JSON object: {\"has_hallucination\": bool, \"reason\": \"...\", \"severity\": \"low|medium|high\"}. Only output JSON."},
				{"role": "user", "content": fmt.Sprintf("Prompt/Context: %s\n\nAssistant Response: %s", prompt[:min(len(prompt), 2000)], completion[:min(len(completion), 2000)])},
			},
			"temperature": 0,
			"max_tokens":  256,
		}
		bodyBytes, _ := json.Marshal(body)

		req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(bodyBytes))
		if err != nil {
			continue
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 15 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != 200 {
			continue
		}

		var chatResp struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		if err := json.Unmarshal(respBody, &chatResp); err != nil || len(chatResp.Choices) == 0 {
			continue
		}

		var judgment struct {
			HasHallucination bool   `json:"has_hallucination"`
			Reason           string `json:"reason"`
			Severity         string `json:"severity"`
		}
		if err := json.Unmarshal([]byte(chatResp.Choices[0].Message.Content), &judgment); err != nil {
			continue
		}

		if judgment.HasHallucination {
			var sev model.IssueSeverity
			switch judgment.Severity {
			case "high":
				sev = model.IssueSeverityHigh
			case "medium":
				sev = model.IssueSeverityMedium
			default:
				sev = model.IssueSeverityLow
			}
			findings = append(findings, IssueFinding{
				Evaluator: EvaluatorHallucination,
				Severity:  sev,
				Title:     "Potential hallucination detected",
				Evidence: map[string]interface{}{
					"span_id":   sp.SpanID,
					"span_name": sp.Name,
					"reason":    judgment.Reason,
				},
			})
		}
	}
	return findings, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

var BuiltinEvaluators = map[Evaluator]EvaluatorFunc{
	EvaluatorEmptyToolResult:      EmptyToolResult,
	EvaluatorToolCallError:        ToolCallError,
	EvaluatorLLMError:             LLMError,
	EvaluatorTokenBudget:          TokenBudget,
	EvaluatorSlowTrace:            SlowTrace,
	EvaluatorRepeatedToolCall:     RepeatedToolCall,
	EvaluatorEmptyLLMOutput:       EmptyLLMOutput,
	EvaluatorOutputFormatMismatch: OutputFormatMismatch,
	EvaluatorHallucination:        Hallucination,
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

func ProcessFindings(ctx context.Context, store store.Store, dispatcher *webhook.Dispatcher, projectID, traceID string, findings []IssueFinding) error {
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
			// Notify webhooks about new issue
			if dispatcher != nil {
				dispatcher.IssueOpened(ctx, projectID, issue)
			}
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
