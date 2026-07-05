package monitors

import (
	"context"
	"log"
	"time"

	"github.com/blackbox-agentdiff/api/internal/issues"
	"github.com/blackbox-agentdiff/api/internal/model"
	"github.com/blackbox-agentdiff/api/internal/store"
	"github.com/blackbox-agentdiff/api/internal/webhook"
	"github.com/google/uuid"
)

type IncidentPayload struct {
	Incident  model.Incident  `json:"incident"`
	Monitor   model.Monitor   `json:"monitor"`
	MetricID  string          `json:"metric_id"`
	Threshold float64         `json:"threshold"`
	Value     float64         `json:"value"`
	Condition string          `json:"condition"`
}

func StartWorker(ctx context.Context, st store.Store, dispatcher *webhook.Dispatcher, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			evaluateAll(ctx, st, dispatcher)
		}
	}
}

func evaluateAll(ctx context.Context, st store.Store, dispatcher *webhook.Dispatcher) {
	projects, err := st.ProjectList(ctx)
	if err != nil {
		log.Printf("monitor worker: project list: %v", err)
		return
	}
	for _, project := range projects {
		if err := evaluateProject(ctx, st, dispatcher, project.ID); err != nil {
			log.Printf("monitor worker: project %s: %v", project.ID, err)
		}
	}
}

func evaluateProject(ctx context.Context, st store.Store, dispatcher *webhook.Dispatcher, projectID string) error {
	monitors, err := st.MonitorList(ctx, projectID)
	if err != nil {
		return err
	}
	if len(monitors) == 0 {
		return nil
	}

	presetMetrics, err := st.PresetMetrics(ctx, projectID, 3600)
	if err != nil {
		presetMetrics = nil
	}

	presetMap := make(map[string]float64)
	if presetMetrics != nil {
		for _, pm := range presetMetrics {
			presetMap[pm.Slug] = pm.Value
		}
	}

	for _, monitor := range monitors {
		evaluateMonitor(ctx, st, dispatcher, monitor, presetMap)
	}
	return nil
}

func evaluateMonitor(ctx context.Context, st store.Store, dispatcher *webhook.Dispatcher, monitor model.Monitor, presetMap map[string]float64) {
	value, ok := presetMap[monitor.MetricID]
	if !ok {
		events, err := st.MetricEventsGet(ctx, monitor.MetricID, 1)
		if err != nil || len(events) == 0 {
			return
		}
		value = events[0].Value
	}

	breaching := false
	if monitor.Condition == model.MonitorConditionAbove && value > monitor.Threshold {
		breaching = true
	} else if monitor.Condition == model.MonitorConditionBelow && value < monitor.Threshold {
		breaching = true
	}

	now := time.Now()

	if breaching {
		if monitor.Status == model.MonitorStatusAlerting {
			return
		}
		incident := model.Incident{
			ID:        uuid.New().String(),
			MonitorID: monitor.ID,
			ProjectID: monitor.ProjectID,
			Status:    model.IncidentStatusUnresolved,
			CreatedAt: model.Time{now},
		}
		if _, err := st.IncidentCreate(ctx, incident); err != nil {
			log.Printf("monitor worker: incident create: %v", err)
			return
		}
		if err := st.MonitorSetFired(ctx, monitor.ID, model.MonitorStatusAlerting, model.Time{now}); err != nil {
			log.Printf("monitor worker: monitor set fired: %v", err)
		}
		payload := IncidentPayload{
			Incident:  incident,
			Monitor:   monitor,
			MetricID:  monitor.MetricID,
			Threshold: monitor.Threshold,
			Value:     value,
			Condition: string(monitor.Condition),
		}
		if dispatcher != nil {
			dispatcher.Dispatch(ctx, monitor.ProjectID, "incident.created", payload)
		}
		rcaCtx := context.Background()
		go issues.AnalyzeIncident(rcaCtx, st, dispatcher, incident, monitor, value)
	} else {
		if monitor.Status == model.MonitorStatusAlerting {
			if err := st.MonitorSetFired(ctx, monitor.ID, model.MonitorStatusOK, model.Time{now}); err != nil {
				log.Printf("monitor worker: monitor set ok: %v", err)
			}
		}
	}
}
