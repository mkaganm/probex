package watch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/probex/probex/internal/models"
	"github.com/probex/probex/internal/schema"
)

// WatchEvent is the result of a single watch cycle.
type WatchEvent struct {
	Timestamp        time.Time `json:"timestamp"`
	EndpointsChecked int       `json:"endpoints_checked"`
	Anomalies        []Anomaly `json:"anomalies,omitempty"`
	Drifts           []Drift   `json:"drifts,omitempty"`
}

// EventHandler is called after each watch cycle.
type EventHandler func(event WatchEvent)

// Watcher continuously monitors API endpoints for anomalies and drift.
type Watcher struct {
	config   models.WatchOptions
	profile  *models.APIProfile
	anomaly  *AnomalyDetector
	drift    *DriftDetector
	alerter  *Alerter
	client   *http.Client
	inferrer *schema.Inferrer
	onEvent  EventHandler
}

// New creates a new Watcher.
func New(profile *models.APIProfile, config models.WatchOptions, alerter *Alerter) *Watcher {
	if config.Interval == 0 {
		config.Interval = 5 * time.Minute
	}
	return &Watcher{
		config:   config,
		profile:  profile,
		anomaly:  NewAnomalyDetector(2.0),
		drift:    NewDriftDetector(),
		alerter:  alerter,
		client:   &http.Client{Timeout: 30 * time.Second},
		inferrer: schema.New(),
	}
}

// OnEvent sets a callback for watch events.
func (w *Watcher) OnEvent(fn EventHandler) {
	w.onEvent = fn
}

// Start begins the watch loop. It blocks until the context is cancelled.
func (w *Watcher) Start(ctx context.Context) error {
	// Run immediately on start
	w.runCycle(ctx)

	ticker := time.NewTicker(w.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			w.runCycle(ctx)
		}
	}
}

// runCycle performs a single watch cycle: probe endpoints, detect anomalies/drift, alert.
func (w *Watcher) runCycle(ctx context.Context) {
	endpoints := w.filterEndpoints()
	event := WatchEvent{
		Timestamp:        time.Now(),
		EndpointsChecked: len(endpoints),
	}

	for _, ep := range endpoints {
		select {
		case <-ctx.Done():
			return
		default:
		}

		anomalies, drifts := w.probeEndpoint(ctx, ep)
		event.Anomalies = append(event.Anomalies, anomalies...)
		event.Drifts = append(event.Drifts, drifts...)
	}

	// Send alerts for any findings
	if len(event.Anomalies) > 0 || len(event.Drifts) > 0 {
		w.sendAlerts(ctx, endpoints, event)
	}

	if w.onEvent != nil {
		w.onEvent(event)
	}
}

// filterEndpoints returns the subset of endpoints to watch.
func (w *Watcher) filterEndpoints() []models.Endpoint {
	if len(w.config.Endpoints) == 0 {
		return w.profile.Endpoints
	}

	filterSet := make(map[string]bool)
	for _, ep := range w.config.Endpoints {
		filterSet[ep] = true
	}

	var filtered []models.Endpoint
	for _, ep := range w.profile.Endpoints {
		key := fmt.Sprintf("%s %s", ep.Method, ep.Path)
		if filterSet[key] || filterSet[ep.Path] || filterSet[ep.ID] {
			filtered = append(filtered, ep)
		}
	}
	return filtered
}

// probeEndpoint sends a request to an endpoint and checks for anomalies/drift.
func (w *Watcher) probeEndpoint(ctx context.Context, ep models.Endpoint) ([]Anomaly, []Drift) {
	var anomalies []Anomaly
	var drifts []Drift

	url := ep.FullURL()
	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(ep.Method), url, nil)
	if err != nil {
		return nil, nil
	}

	for k, v := range ep.Headers {
		if v != "{{auth_token}}" {
			req.Header.Set(k, v)
		}
	}

	start := time.Now()
	resp, err := w.client.Do(req)
	duration := time.Since(start)

	if err != nil {
		// Connection error is itself an anomaly
		anomalies = append(anomalies, Anomaly{
			EndpointID:  ep.ID,
			Metric:      "connection",
			Severity:    models.SeverityCritical,
			Description: fmt.Sprintf("Connection failed: %v", err),
		})
		return anomalies, nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))

	// Check against baseline
	if w.profile.Baseline != nil {
		baseline := w.profile.Baseline.Endpoints[ep.ID]
		if baseline != nil {
			// Response time anomaly
			if a := w.anomaly.CheckResponseTime(ep.ID, duration, baseline); a != nil {
				anomalies = append(anomalies, *a)
			}
			// Status code anomaly
			if a := w.anomaly.CheckStatusCode(ep.ID, resp.StatusCode, baseline); a != nil {
				anomalies = append(anomalies, *a)
			}
		}
	}

	// Schema drift detection
	if len(body) > 0 && isJSONResponse(resp) {
		actualSchema, err := w.inferrer.InferFromJSON(body)
		if err == nil {
			knownSchema := findResponseSchema(ep, resp.StatusCode)
			if knownSchema != nil {
				d := w.drift.Compare(ep.ID, knownSchema, actualSchema)
				drifts = append(drifts, d...)
			}
		}
	}

	return anomalies, drifts
}

// sendAlerts groups findings by endpoint and sends alerts.
func (w *Watcher) sendAlerts(ctx context.Context, endpoints []models.Endpoint, event WatchEvent) {
	if w.alerter == nil || !w.alerter.HasTargets() {
		return
	}

	// Group by endpoint
	byEndpoint := make(map[string]*Alert)
	for _, a := range event.Anomalies {
		if _, ok := byEndpoint[a.EndpointID]; !ok {
			byEndpoint[a.EndpointID] = &Alert{
				Timestamp: event.Timestamp,
				Endpoint:  a.EndpointID,
			}
		}
		byEndpoint[a.EndpointID].Anomalies = append(byEndpoint[a.EndpointID].Anomalies, a)
	}
	for _, d := range event.Drifts {
		if _, ok := byEndpoint[d.EndpointID]; !ok {
			byEndpoint[d.EndpointID] = &Alert{
				Timestamp: event.Timestamp,
				Endpoint:  d.EndpointID,
			}
		}
		byEndpoint[d.EndpointID].Drifts = append(byEndpoint[d.EndpointID].Drifts, d)
	}

	for _, alert := range byEndpoint {
		alert.Severity = highestSeverity(alert.Anomalies, alert.Drifts)
		alert.Message = buildAlertMessage(alert)
		_ = w.alerter.Send(ctx, *alert)
	}
}

func isJSONResponse(resp *http.Response) bool {
	ct := resp.Header.Get("Content-Type")
	return strings.Contains(ct, "json") || (ct == "" && resp.StatusCode < 400)
}

func findResponseSchema(ep models.Endpoint, statusCode int) *models.Schema {
	for _, r := range ep.Responses {
		if r.StatusCode == statusCode && r.Schema != nil {
			return r.Schema
		}
	}
	// Try to infer from sample body
	for _, r := range ep.Responses {
		if r.StatusCode == statusCode && r.SampleBody != "" {
			var v any
			if err := json.Unmarshal([]byte(r.SampleBody), &v); err == nil {
				inf := schema.New()
				s, err := inf.InferFromJSON([]byte(r.SampleBody))
				if err == nil {
					return s
				}
			}
		}
	}
	return nil
}

func highestSeverity(anomalies []Anomaly, drifts []Drift) models.Severity {
	sevOrder := map[models.Severity]int{
		models.SeverityCritical: 4,
		models.SeverityHigh:     3,
		models.SeverityMedium:   2,
		models.SeverityLow:      1,
		models.SeverityInfo:     0,
	}
	highest := models.SeverityInfo
	for _, a := range anomalies {
		if sevOrder[a.Severity] > sevOrder[highest] {
			highest = a.Severity
		}
	}
	for _, d := range drifts {
		if sevOrder[d.Severity] > sevOrder[highest] {
			highest = d.Severity
		}
	}
	return highest
}

func buildAlertMessage(alert *Alert) string {
	parts := make([]string, 0, 2)
	if len(alert.Anomalies) > 0 {
		parts = append(parts, fmt.Sprintf("%d anomalies", len(alert.Anomalies)))
	}
	if len(alert.Drifts) > 0 {
		parts = append(parts, fmt.Sprintf("%d schema drifts", len(alert.Drifts)))
	}
	return fmt.Sprintf("Detected %s", strings.Join(parts, " and "))
}
