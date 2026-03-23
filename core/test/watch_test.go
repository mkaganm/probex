package test

import (
	"testing"
	"time"

	"github.com/mkaganm/probex/internal/models"
	"github.com/mkaganm/probex/internal/watch"
)

func TestAnomalyDetectorResponseTime(t *testing.T) {
	d := watch.NewAnomalyDetector(2.0)
	baseline := &models.EndpointBaseline{
		EndpointID:      "ep1",
		AvgResponseTime: 100 * time.Millisecond,
		P50ResponseTime: 90 * time.Millisecond,
		P95ResponseTime: 200 * time.Millisecond,
		SampleCount:     100,
	}

	// Normal response — no anomaly expected
	a := d.CheckResponseTime("ep1", 120*time.Millisecond, baseline)
	if a != nil {
		t.Errorf("Expected no anomaly for normal response time, got: %v", a.Description)
	}

	// Extremely slow response — anomaly expected
	a = d.CheckResponseTime("ep1", 2*time.Second, baseline)
	if a == nil {
		t.Fatal("Expected anomaly for 2s response time against 100ms baseline")
	}
	if a.Metric != "response_time" {
		t.Errorf("Expected metric 'response_time', got %q", a.Metric)
	}
}

func TestAnomalyDetectorStatusCode(t *testing.T) {
	d := watch.NewAnomalyDetector(2.0)
	baseline := &models.EndpointBaseline{
		StatusCodeDist: map[int]int{200: 95, 404: 5},
		SampleCount:    100,
	}

	// Known status code — no anomaly
	a := d.CheckStatusCode("ep1", 200, baseline)
	if a != nil {
		t.Errorf("Expected no anomaly for known status code 200")
	}

	// Unknown status code — anomaly
	a = d.CheckStatusCode("ep1", 500, baseline)
	if a == nil {
		t.Fatal("Expected anomaly for unknown status code 500")
	}
	if a.Severity != models.SeverityCritical {
		t.Errorf("Expected critical severity for 500, got %s", a.Severity)
	}
}

func TestDriftDetectorFieldRemoved(t *testing.T) {
	d := watch.NewDriftDetector()
	known := &models.Schema{
		Type: "object",
		Properties: map[string]*models.Schema{
			"id":    {Type: "integer"},
			"name":  {Type: "string"},
			"email": {Type: "string"},
		},
	}
	actual := &models.Schema{
		Type: "object",
		Properties: map[string]*models.Schema{
			"id":   {Type: "integer"},
			"name": {Type: "string"},
			// email removed
		},
	}

	drifts := d.Compare("ep1", known, actual)
	if len(drifts) != 1 {
		t.Fatalf("Expected 1 drift (removed field), got %d", len(drifts))
	}
	if drifts[0].Change != "removed" {
		t.Errorf("Expected 'removed' change, got %q", drifts[0].Change)
	}
	if drifts[0].Field != "email" {
		t.Errorf("Expected field 'email', got %q", drifts[0].Field)
	}
	if drifts[0].Severity != models.SeverityHigh {
		t.Errorf("Expected high severity for removed field, got %s", drifts[0].Severity)
	}
}

func TestDriftDetectorFieldAdded(t *testing.T) {
	d := watch.NewDriftDetector()
	known := &models.Schema{
		Type: "object",
		Properties: map[string]*models.Schema{
			"id": {Type: "integer"},
		},
	}
	actual := &models.Schema{
		Type: "object",
		Properties: map[string]*models.Schema{
			"id":        {Type: "integer"},
			"new_field": {Type: "string"},
		},
	}

	drifts := d.Compare("ep1", known, actual)
	if len(drifts) != 1 {
		t.Fatalf("Expected 1 drift (added field), got %d", len(drifts))
	}
	if drifts[0].Change != "added" {
		t.Errorf("Expected 'added' change, got %q", drifts[0].Change)
	}
	if drifts[0].Severity != models.SeverityLow {
		t.Errorf("Expected low severity for added field, got %s", drifts[0].Severity)
	}
}

func TestDriftDetectorTypeChanged(t *testing.T) {
	d := watch.NewDriftDetector()
	known := &models.Schema{
		Type: "object",
		Properties: map[string]*models.Schema{
			"count": {Type: "integer"},
		},
	}
	actual := &models.Schema{
		Type: "object",
		Properties: map[string]*models.Schema{
			"count": {Type: "string"},
		},
	}

	drifts := d.Compare("ep1", known, actual)
	if len(drifts) != 1 {
		t.Fatalf("Expected 1 drift (type changed), got %d", len(drifts))
	}
	if drifts[0].Change != "type_changed" {
		t.Errorf("Expected 'type_changed', got %q", drifts[0].Change)
	}
	if drifts[0].OldValue != "integer" || drifts[0].NewValue != "string" {
		t.Errorf("Expected old=integer new=string, got old=%s new=%s", drifts[0].OldValue, drifts[0].NewValue)
	}
}

func TestParseTargets(t *testing.T) {
	// Default to stdout
	targets := watch.ParseTargets("")
	if len(targets) != 1 {
		t.Fatalf("Expected 1 default target, got %d", len(targets))
	}
	if targets[0].Type() != "stdout" {
		t.Errorf("Expected stdout target, got %s", targets[0].Type())
	}

	// Webhook
	targets = watch.ParseTargets("webhook:https://example.com/hook")
	if len(targets) != 1 || targets[0].Type() != "webhook" {
		t.Error("Expected webhook target")
	}

	// Slack
	targets = watch.ParseTargets("slack:https://hooks.slack.com/xxx")
	if len(targets) != 1 || targets[0].Type() != "slack" {
		t.Error("Expected slack target")
	}

	// Multiple
	targets = watch.ParseTargets("stdout,webhook:https://example.com")
	if len(targets) != 2 {
		t.Errorf("Expected 2 targets, got %d", len(targets))
	}
}
