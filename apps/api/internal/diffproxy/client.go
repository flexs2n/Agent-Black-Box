package diffproxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = "http://diff-service:5001"
	}
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type DiffRequest struct {
	TraceA map[string]any   `json:"traceA"`
	TraceB map[string]any   `json:"traceB"`
	StatsA map[string]int64 `json:"statsA"`
	StatsB map[string]int64 `json:"statsB"`
}

type DiffResult struct {
	TraceAID        string      `json:"traceAId"`
	TraceBID        string      `json:"traceBId"`
	SimilarityScore float64     `json:"similarityScore"`
	SpanDiffs       []SpanDiff  `json:"spanDiffs"`
	MetricDelta     MetricDelta `json:"metricDelta"`
	CreatedAt       string      `json:"createdAt"`
}

type SpanDiff struct {
	Status         string          `json:"status"`
	SpanAID        string          `json:"spanAId,omitempty"`
	SpanBID        string          `json:"spanBId,omitempty"`
	Name           string          `json:"name"`
	SpanKind       string          `json:"spanKind"`
	Depth          int             `json:"depth"`
	AttributeDiffs []AttributeDiff `json:"attributeDiffs,omitempty"`
	ContentDiff    *ContentDiff    `json:"contentDiff,omitempty"`
}

type AttributeDiff struct {
	Key        string `json:"key"`
	ValueA     string `json:"valueA,omitempty"`
	ValueB     string `json:"valueB,omitempty"`
	ChangeType string `json:"changeType"`
}

type ContentDiff struct {
	Type     string          `json:"type"`
	WordDiff []WordDiffChunk `json:"wordDiff,omitempty"`
	JsonDiff []JsonDiffNode  `json:"jsonDiff,omitempty"`
}

type WordDiffChunk struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type JsonDiffNode struct {
	key      string         `json:"key"`
	Type     string         `json:"type"`
	ValueA   interface{}    `json:"valueA,omitempty"`
	ValueB   interface{}    `json:"valueB,omitempty"`
	Children []JsonDiffNode `json:"children,omitempty"`
}

type MetricDelta struct {
	DurationMs    MetricDeltaItem `json:"durationMs"`
	InputTokens   MetricDeltaItem `json:"inputTokens"`
	OutputTokens  MetricDeltaItem `json:"outputTokens"`
	ToolCallCount MetricDeltaItem `json:"toolCallCount"`
	LLMCallCount  MetricDeltaItem `json:"llmCallCount"`
}

type MetricDeltaItem struct {
	A            int64   `json:"a"`
	B            int64   `json:"b"`
	Delta        int64   `json:"delta"`
	DeltaPercent float64 `json:"deltaPercent"`
}

type BatchDiffRequest struct {
	ReferenceTrace map[string]any   `json:"referenceTrace"`
	ReferenceStats map[string]int64 `json:"referenceStats"`
	Comparisons    []BatchCompareItem `json:"comparisons"`
}

type BatchCompareItem struct {
	TraceID string         `json:"traceId"`
	Trace   map[string]any `json:"trace"`
	Stats   map[string]int64 `json:"stats"`
}

func (c *Client) ComputeBatch(ctx context.Context, req BatchDiffRequest) (map[string]any, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/internal/diffs/batch", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var errResp map[string]any
		json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("diff-service batch returned %d: %v", resp.StatusCode, errResp)
	}
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) Compute(ctx context.Context, traceA, traceB map[string]any, statsA, statsB map[string]int64) (map[string]any, error) {
	reqBody := DiffRequest{
		TraceA: traceA,
		TraceB: traceB,
		StatsA: statsA,
		StatsB: statsB,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/internal/diff", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]any
		json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("diff-service returned %d: %v", resp.StatusCode, errResp)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}
