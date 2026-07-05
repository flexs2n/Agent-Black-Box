package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/blackbox-agentdiff/api/internal/model"
	"github.com/blackbox-agentdiff/api/internal/store"
	"github.com/google/uuid"
)

const (
	maxRetries    = 5
	requestTimeout = 10 * time.Second
)

type Dispatcher struct {
	store  store.Store
	client *http.Client
}

func NewDispatcher(st store.Store) *Dispatcher {
	return &Dispatcher{
		store: st,
		client: &http.Client{
			Timeout: requestTimeout,
		},
	}
}

type Payload struct {
	Event     string      `json:"event"`
	Timestamp string      `json:"timestamp"`
	ProjectID string      `json:"projectId"`
	Issue     interface{} `json:"issue,omitempty"`
}

func (d *Dispatcher) Dispatch(ctx context.Context, projectID, eventType string, payload interface{}) {
	webhooks, err := d.store.WebhookListByEvent(ctx, projectID, eventType)
	if err != nil || len(webhooks) == 0 {
		return
	}
	for _, wh := range webhooks {
		go d.dispatchOne(wh, eventType, payload)
	}
}

func (d *Dispatcher) dispatchOne(wh model.Webhook, eventType string, payload interface{}) {
	body, _ := json.Marshal(payload)
	for attempt := 1; attempt <= maxRetries; attempt++ {
		delivery := model.WebhookDelivery{
			ID:        uuid.New().String(),
			WebhookID: wh.ID,
			Event:     eventType,
			Payload:   string(body),
			Attempt:   attempt,
			CreatedAt: model.Time{Time: time.Now()},
		}
		mac := hmac.New(sha256.New, []byte(wh.SecretHash))
		mac.Write(body)
		sig := hex.EncodeToString(mac.Sum(nil))

		req, err := http.NewRequest("POST", wh.URL, bytes.NewReader(body))
		if err != nil {
			errStr := err.Error()
			delivery.Response = &errStr
			delivery.StatusCode = intPtr(0)
			d.store.WebhookDeliveryCreate(context.Background(), delivery)
			d.backoff(attempt)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Blackbox-Signature", "sha256="+sig)
		req.Header.Set("User-Agent", "Blackbox-Webhook/1.0")

		ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
		req = req.WithContext(ctx)

		resp, err := d.client.Do(req)
		cancel()

		if err != nil {
			errStr := err.Error()
			delivery.Response = &errStr
			delivery.StatusCode = intPtr(0)
			d.store.WebhookDeliveryCreate(context.Background(), delivery)
			d.backoff(attempt)
			continue
		}

		code := resp.StatusCode
		delivery.StatusCode = &code
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		respStr := string(respBody)
		delivery.Response = &respStr

		if code >= 200 && code < 300 {
			now := model.Time{Time: time.Now()}
			delivery.DeliveredAt = &now
			d.store.WebhookDeliveryCreate(context.Background(), delivery)
			return
		}

		d.store.WebhookDeliveryCreate(context.Background(), delivery)
		if attempt < maxRetries {
			d.backoff(attempt)
		}
	}
}

func (d *Dispatcher) backoff(attempt int) {
	delay := time.Duration(1<<uint(attempt-1)) * time.Second
	time.Sleep(delay)
}

func intPtr(n int) *int {
	return &n
}

func (d *Dispatcher) IssueOpened(ctx context.Context, projectID string, issue interface{}) {
	payload := Payload{
		Event:     "issue.opened",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		ProjectID: projectID,
		Issue:     issue,
	}
	d.Dispatch(ctx, projectID, "issue.opened", payload)
}

func (d *Dispatcher) IssueResolved(ctx context.Context, projectID string, issue interface{}) {
	payload := Payload{
		Event:     "issue.resolved",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		ProjectID: projectID,
		Issue:     issue,
	}
	d.Dispatch(ctx, projectID, "issue.resolved", payload)
}