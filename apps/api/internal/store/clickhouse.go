package store

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/blackbox-agentdiff/api/internal/model"
)

type ClickHouseSpanStore struct {
	baseURL    string
	client     *http.Client
}

func NewClickHouseSpanStore(baseURL string) *ClickHouseSpanStore {
	return &ClickHouseSpanStore{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *ClickHouseSpanStore) query(ctx context.Context, sql string, args ...interface{}) (string, error) {
	query := sql
	for _, arg := range args {
		query = strings.Replace(query, "?", fmt.Sprintf("'%v'", arg), 1)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"?query="+urlEncode(query), bytes.NewReader(nil))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("clickhouse returned %d: %s", resp.StatusCode, string(body))
	}
	return string(body), nil
}

func urlEncode(s string) string {
	return url.QueryEscape(s)
}

func (c *ClickHouseSpanStore) ensureTable(ctx context.Context) error {
	sql := `CREATE TABLE IF NOT EXISTS spans (
		trace_id String,
		span_id String,
		parent_span_id Nullable(String),
		project_id String,
		name String,
		span_kind String,
		status String,
		started_at DateTime64(3),
		ended_at DateTime64(3),
		duration_ms Int64,
		attributes String
	) ENGINE = MergeTree()
	ORDER BY (project_id, trace_id, started_at)`
	_, err := c.query(ctx, sql)
	return err
}

func (c *ClickHouseSpanStore) SpanPutBatch(ctx context.Context, spans []model.Span) error {
	if len(spans) == 0 {
		return nil
	}
	if err := c.ensureTable(ctx); err != nil {
		// Table may already exist; ignore error
		_ = err
	}

	// Build a VALUES batch insert
	rows := make([]string, len(spans))
	for i, sp := range spans {
		parentID := "NULL"
		if sp.ParentSpanID != nil {
			parentID = fmt.Sprintf("'%s'", *sp.ParentSpanID)
		}
		rows[i] = fmt.Sprintf("('%s','%s',%s,'%s','%s','%s','%s',toDateTime64('%.9f',9),toDateTime64('%.9f',9),%d,'%s')",
			escapeCH(sp.TraceID), escapeCH(sp.SpanID), parentID,
			escapeCH(sp.ProjectID), escapeCH(sp.Name), escapeCH(sp.SpanKind), escapeCH(sp.Status),
			float64(sp.StartedAt.UnixNano())/1e9,
			float64(sp.EndedAt.UnixNano())/1e9,
			sp.DurationMs, escapeCH(sp.Attributes))
	}

	sql := "INSERT INTO spans (trace_id, span_id, parent_span_id, project_id, name, span_kind, status, started_at, ended_at, duration_ms, attributes) VALUES " + strings.Join(rows, ",")
	_, err := c.query(ctx, sql)
	return err
}

func (c *ClickHouseSpanStore) SpanList(ctx context.Context, traceID string) ([]model.Span, error) {
	if err := c.ensureTable(ctx); err != nil {
		_ = err
	}

	raw, err := c.query(ctx, fmt.Sprintf("SELECT trace_id, span_id, parent_span_id, project_id, name, span_kind, status, started_at, ended_at, duration_ms, attributes FROM spans WHERE trace_id = '%s' ORDER BY started_at FORMAT JSONEachRow", escapeCH(traceID)))
	if err != nil {
		return nil, err
	}
	if raw == "" {
		return nil, nil
	}

	lines := strings.Split(strings.TrimSpace(raw), "\n")
	spans := make([]model.Span, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		var row struct {
			TraceID      string  `json:"trace_id"`
			SpanID       string  `json:"span_id"`
			ParentSpanID *string `json:"parent_span_id"`
			ProjectID    string  `json:"project_id"`
			Name         string  `json:"name"`
			SpanKind     string  `json:"span_kind"`
			Status       string  `json:"status"`
			StartedAt    float64 `json:"started_at"`
			EndedAt      float64 `json:"ended_at"`
			DurationMs   int64   `json:"duration_ms"`
			Attributes   string  `json:"attributes"`
		}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			continue
		}
		spans = append(spans, model.Span{
			TraceID:      row.TraceID,
			SpanID:       row.SpanID,
			ParentSpanID: row.ParentSpanID,
			ProjectID:    row.ProjectID,
			Name:         row.Name,
			SpanKind:     row.SpanKind,
			Status:       row.Status,
			StartedAt:    model.Time{time.UnixMilli(int64(row.StartedAt * 1000))},
			EndedAt:      model.Time{time.UnixMilli(int64(row.EndedAt * 1000))},
			DurationMs:   row.DurationMs,
			Attributes:   row.Attributes,
		})
	}
	return spans, nil
}

func escapeCH(s string) string {
	r := strings.NewReplacer("\\", "\\\\", "'", "\\'", "\n", "\\n", "\r", "\\r", "\x00", "")
	return r.Replace(s)
}
