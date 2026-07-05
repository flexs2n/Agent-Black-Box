package issues

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/blackbox-agentdiff/api/internal/model"
	"github.com/blackbox-agentdiff/api/internal/store"
	"github.com/blackbox-agentdiff/api/internal/webhook"
)

func AnalyzeIncident(ctx context.Context, st store.Store, dispatcher *webhook.Dispatcher, incident model.Incident, monitor model.Monitor, value float64) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return
	}

	traces, err := st.TraceList(ctx, incident.ProjectID, store.TraceFilters{
		Sort: "created_at DESC",
		Page: 1,
	})
	if err != nil || len(traces) == 0 {
		log.Printf("rca: no traces for incident %s", incident.ID)
		return
	}

	summary := buildRCASummary(traces, monitor, value)

	rcaText, fixText, err := callLLMForRCA(ctx, apiKey, summary)
	if err != nil {
		log.Printf("rca: llm call failed for incident %s: %v", incident.ID, err)
		return
	}

	update := model.IncidentUpdate{
		Status:    modelPtr(model.IncidentStatusAnalyzed),
		RootCause: &rcaText,
	}
	if _, err := st.IncidentUpdate(ctx, incident.ID, update); err != nil {
		log.Printf("rca: incident update failed: %v", err)
		return
	}

	if dispatcher != nil {
		payload := map[string]interface{}{
			"incident":      incident,
			"root_cause":    rcaText,
			"suggested_fix": fixText,
			"monitor":       monitor,
			"threshold":     monitor.Threshold,
			"value":         value,
		}
		dispatcher.Dispatch(ctx, incident.ProjectID, "incident.analyzed", payload)
	}
}

func buildRCASummary(traces []model.Trace, monitor model.Monitor, value float64) string {
	summary := fmt.Sprintf("Monitor '%s' (%s) breached threshold %.2f with value %.2f.\n\n",
		monitor.MetricID, monitor.Condition, monitor.Threshold, value)

	for i, trace := range traces {
		if i >= 5 {
			break
		}
		name := "<nil>"
		if trace.AgentName != nil {
			name = *trace.AgentName
		}
		summary += fmt.Sprintf("- Agent: %s, Status: %s", name, trace.Status)
		if trace.Error != nil {
			summary += fmt.Sprintf(", Error: %s", truncateStr(*trace.Error, 200))
		}
		if trace.Input != nil {
			summary += fmt.Sprintf(", Input: %s", truncateStr(*trace.Input, 200))
		}
		if trace.Output != nil {
			summary += fmt.Sprintf(", Output: %s", truncateStr(*trace.Output, 200))
		}
		summary += "\n"
	}
	return summary
}

func callLLMForRCA(ctx context.Context, apiKey, summary string) (string, string, error) {
	prompt := fmt.Sprintf(`You are a root cause analysis assistant for an AI agent monitoring platform.
Based on the following monitor breach information, identify the most likely root cause and suggest a fix.

%s

Respond with JSON: {"root_cause": "...", "suggested_fix": "..."}
Only output the JSON object.`, summary)

	body := map[string]interface{}{
		"model": "gpt-4o-mini",
		"messages": []map[string]string{
			{"role": "system", "content": "You are an RCA assistant. Output valid JSON only."},
			{"role": "user", "content": prompt},
		},
		"temperature": 0,
		"max_tokens":  512,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", "", fmt.Errorf("marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer " + apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("llm returned %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &chatResp); err != nil || len(chatResp.Choices) == 0 {
		return "", "", fmt.Errorf("unmarshal failed: %v", err)
	}

	var rcaResult struct {
		RootCause    string `json:"root_cause"`
		SuggestedFix string `json:"suggested_fix"`
	}
	if err := json.Unmarshal([]byte(chatResp.Choices[0].Message.Content), &rcaResult); err != nil {
		return "", "", fmt.Errorf("rca parse failed: %v", err)
	}

	return rcaResult.RootCause, rcaResult.SuggestedFix, nil
}

func modelPtr(s model.IncidentStatus) *model.IncidentStatus {
	return &s
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
