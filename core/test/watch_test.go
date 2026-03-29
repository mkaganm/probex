package test

import (
	"testing"
	"time"

	"github.com/mkaganm/probex/internal/models"
	"github.com/mkaganm/probex/internal/watch"
)

// ---------------------------------------------------------------------------
// AnomalyDetector.CheckResponseTime — z-score anomaly detection
// ---------------------------------------------------------------------------

func TestWatchAnomalyResponseTime_Normal(t *testing.T) {
	d := watch.NewAnomalyDetector(2.0)
	baseline := &models.EndpointBaseline{
		EndpointID:      "ep1",
		AvgResponseTime: 100 * time.Millisecond,
		P50ResponseTime: 90 * time.Millisecond,
		P95ResponseTime: 200 * time.Millisecond,
		SampleCount:     100,
	}

	// 120ms is well within a 2-sigma band of a 100ms average.
	a := d.CheckResponseTime("ep1", 120*time.Millisecond, baseline)
	if a != nil {
		t.Errorf("Expected nil for normal response time, got anomaly: %s", a.Description)
	}
}

func TestWatchAnomalyResponseTime_Anomalous(t *testing.T) {
	d := watch.NewAnomalyDetector(2.0)
	baseline := &models.EndpointBaseline{
		EndpointID:      "ep1",
		AvgResponseTime: 100 * time.Millisecond,
		P50ResponseTime: 90 * time.Millisecond,
		P95ResponseTime: 200 * time.Millisecond,
		SampleCount:     100,
	}

	// 2s is far outside the expected range.
	a := d.CheckResponseTime("ep1", 2*time.Second, baseline)
	if a == nil {
		t.Fatal("Expected anomaly for 2s response time against 100ms baseline")
	}
	if a.Metric != "response_time" {
		t.Errorf("Expected metric 'response_time', got %q", a.Metric)
	}
	if a.EndpointID != "ep1" {
		t.Errorf("Expected endpoint ID 'ep1', got %q", a.EndpointID)
	}
	if a.ZScore <= 2.0 {
		t.Errorf("Expected z-score > 2.0, got %.2f", a.ZScore)
	}
}

func TestWatchAnomalyResponseTime_NilBaseline(t *testing.T) {
	d := watch.NewAnomalyDetector(2.0)
	a := d.CheckResponseTime("ep1", 500*time.Millisecond, nil)
	if a != nil {
		t.Error("Expected nil when baseline is nil")
	}
}

func TestWatchAnomalyResponseTime_ZeroSamples(t *testing.T) {
	d := watch.NewAnomalyDetector(2.0)
	baseline := &models.EndpointBaseline{
		AvgResponseTime: 100 * time.Millisecond,
		P50ResponseTime: 90 * time.Millisecond,
		P95ResponseTime: 200 * time.Millisecond,
		SampleCount:     0,
	}
	a := d.CheckResponseTime("ep1", 5*time.Second, baseline)
	if a != nil {
		t.Error("Expected nil when SampleCount is 0")
	}
}

func TestWatchAnomalyResponseTime_OneSample(t *testing.T) {
	d := watch.NewAnomalyDetector(2.0)
	baseline := &models.EndpointBaseline{
		AvgResponseTime: 100 * time.Millisecond,
		P50ResponseTime: 100 * time.Millisecond,
		P95ResponseTime: 100 * time.Millisecond,
		SampleCount:     1,
	}
	// SampleCount < 2 should return nil.
	a := d.CheckResponseTime("ep1", 5*time.Second, baseline)
	if a != nil {
		t.Error("Expected nil when SampleCount is 1 (< 2)")
	}
}

func TestWatchAnomalyResponseTime_SeverityMedium(t *testing.T) {
	// z-score between threshold (2.0) and 2.5 => medium severity.
	d := watch.NewAnomalyDetector(2.0)
	// stddev = (p95 - p50) / 1.65 = (200 - 90) / 1.65 ≈ 66.67ms
	baseline := &models.EndpointBaseline{
		AvgResponseTime: 100 * time.Millisecond,
		P50ResponseTime: 90 * time.Millisecond,
		P95ResponseTime: 200 * time.Millisecond,
		SampleCount:     100,
	}

	// stddev in ns
	stddevNs := float64(200*time.Millisecond-90*time.Millisecond) / 1.65
	// z = 2.2 => actual = avg + 2.2 * stddev  (between 2.0 and 2.5)
	actualNs := float64(100*time.Millisecond) + 2.2*stddevNs
	a := d.CheckResponseTime("ep1", time.Duration(actualNs), baseline)
	if a == nil {
		t.Fatal("Expected anomaly for borderline response time")
	}
	if a.Severity != models.SeverityMedium {
		t.Errorf("Expected medium severity for z ~ 2.2, got %s (z=%.2f)", a.Severity, a.ZScore)
	}
}

func TestWatchAnomalyResponseTime_SeverityCritical(t *testing.T) {
	// z-score > 3.0 => critical severity.
	d := watch.NewAnomalyDetector(2.0)
	// stddev ≈ 66.67ms, for z > 3: actual > 100 + 3*66.67 = 300ms
	baseline := &models.EndpointBaseline{
		AvgResponseTime: 100 * time.Millisecond,
		P50ResponseTime: 90 * time.Millisecond,
		P95ResponseTime: 200 * time.Millisecond,
		SampleCount:     100,
	}

	a := d.CheckResponseTime("ep1", 2*time.Second, baseline)
	if a == nil {
		t.Fatal("Expected anomaly")
	}
	if a.Severity != models.SeverityCritical {
		t.Errorf("Expected critical severity for very high z-score, got %s", a.Severity)
	}
}

func TestWatchAnomalyResponseTime_SeverityHigh(t *testing.T) {
	// z-score between 2.5 and 3.0 => high severity.
	d := watch.NewAnomalyDetector(2.0)
	// stddev = (200 - 90) / 1.65 ≈ 66.67ms
	// For z=2.7: actual = 100 + 2.7*66.67 ≈ 280ms
	baseline := &models.EndpointBaseline{
		AvgResponseTime: 100 * time.Millisecond,
		P50ResponseTime: 90 * time.Millisecond,
		P95ResponseTime: 200 * time.Millisecond,
		SampleCount:     100,
	}

	// stddev in ns: (200_000_000 - 90_000_000) / 1.65 = 66_666_666 ns
	stddevNs := float64(200*time.Millisecond-90*time.Millisecond) / 1.65
	// z = 2.7 => actual = avg + 2.7 * stddev
	actualNs := float64(100*time.Millisecond) + 2.7*stddevNs
	a := d.CheckResponseTime("ep1", time.Duration(actualNs), baseline)
	if a == nil {
		t.Fatal("Expected anomaly for z ~ 2.7")
	}
	if a.Severity != models.SeverityHigh {
		t.Errorf("Expected high severity for z ~ 2.7, got %s (z=%.2f)", a.Severity, a.ZScore)
	}
}

func TestWatchAnomalyResponseTime_ZeroAvg(t *testing.T) {
	d := watch.NewAnomalyDetector(2.0)
	baseline := &models.EndpointBaseline{
		AvgResponseTime: 0,
		P50ResponseTime: 0,
		P95ResponseTime: 0,
		SampleCount:     50,
	}
	a := d.CheckResponseTime("ep1", 500*time.Millisecond, baseline)
	if a != nil {
		t.Error("Expected nil when AvgResponseTime is zero")
	}
}

func TestWatchAnomalyResponseTime_FallbackStddev(t *testing.T) {
	// When p95 <= p50, stddev falls back to avg * 0.2.
	d := watch.NewAnomalyDetector(2.0)
	baseline := &models.EndpointBaseline{
		AvgResponseTime: 100 * time.Millisecond,
		P50ResponseTime: 100 * time.Millisecond,
		P95ResponseTime: 100 * time.Millisecond, // same as p50 => stddev would be 0 => fallback
		SampleCount:     50,
	}
	// fallback stddev = 100ms * 0.2 = 20ms
	// z for 200ms: (200 - 100) / 20 = 5.0 => anomaly, critical
	a := d.CheckResponseTime("ep1", 200*time.Millisecond, baseline)
	if a == nil {
		t.Fatal("Expected anomaly with fallback stddev")
	}
	if a.Severity != models.SeverityCritical {
		t.Errorf("Expected critical for z=5.0, got %s", a.Severity)
	}
}

// ---------------------------------------------------------------------------
// AnomalyDetector.CheckStatusCode — status code anomaly
// ---------------------------------------------------------------------------

func TestWatchAnomalyStatusCode_Expected(t *testing.T) {
	d := watch.NewAnomalyDetector(2.0)
	baseline := &models.EndpointBaseline{
		StatusCodeDist: map[int]int{200: 95, 404: 5},
		SampleCount:    100,
	}
	a := d.CheckStatusCode("ep1", 200, baseline)
	if a != nil {
		t.Error("Expected nil for known status code 200")
	}
}

func TestWatchAnomalyStatusCode_ExpectedMinor(t *testing.T) {
	d := watch.NewAnomalyDetector(2.0)
	baseline := &models.EndpointBaseline{
		StatusCodeDist: map[int]int{200: 95, 404: 5},
		SampleCount:    100,
	}
	// 404 was seen in the baseline, so it is not anomalous.
	a := d.CheckStatusCode("ep1", 404, baseline)
	if a != nil {
		t.Error("Expected nil for known status code 404")
	}
}

func TestWatchAnomalyStatusCode_Unexpected(t *testing.T) {
	d := watch.NewAnomalyDetector(2.0)
	baseline := &models.EndpointBaseline{
		StatusCodeDist: map[int]int{200: 100},
		SampleCount:    100,
	}
	a := d.CheckStatusCode("ep1", 302, baseline)
	if a == nil {
		t.Fatal("Expected anomaly for unseen status code 302")
	}
	if a.Metric != "status_code" {
		t.Errorf("Expected metric 'status_code', got %q", a.Metric)
	}
	if a.Actual != 302 {
		t.Errorf("Expected Actual=302, got %.0f", a.Actual)
	}
}

func TestWatchAnomalyStatusCode_500Critical(t *testing.T) {
	d := watch.NewAnomalyDetector(2.0)
	baseline := &models.EndpointBaseline{
		StatusCodeDist: map[int]int{200: 100},
		SampleCount:    100,
	}
	a := d.CheckStatusCode("ep1", 500, baseline)
	if a == nil {
		t.Fatal("Expected anomaly for 500")
	}
	if a.Severity != models.SeverityCritical {
		t.Errorf("Expected critical severity for 500, got %s", a.Severity)
	}
}

func TestWatchAnomalyStatusCode_502Critical(t *testing.T) {
	d := watch.NewAnomalyDetector(2.0)
	baseline := &models.EndpointBaseline{
		StatusCodeDist: map[int]int{200: 100},
		SampleCount:    100,
	}
	a := d.CheckStatusCode("ep1", 502, baseline)
	if a == nil {
		t.Fatal("Expected anomaly for 502")
	}
	if a.Severity != models.SeverityCritical {
		t.Errorf("Expected critical severity for 502, got %s", a.Severity)
	}
}

func TestWatchAnomalyStatusCode_400High(t *testing.T) {
	d := watch.NewAnomalyDetector(2.0)
	baseline := &models.EndpointBaseline{
		StatusCodeDist: map[int]int{200: 100},
		SampleCount:    100,
	}
	a := d.CheckStatusCode("ep1", 400, baseline)
	if a == nil {
		t.Fatal("Expected anomaly for 400")
	}
	if a.Severity != models.SeverityHigh {
		t.Errorf("Expected high severity for 400, got %s", a.Severity)
	}
}

func TestWatchAnomalyStatusCode_403High(t *testing.T) {
	d := watch.NewAnomalyDetector(2.0)
	baseline := &models.EndpointBaseline{
		StatusCodeDist: map[int]int{200: 100},
		SampleCount:    100,
	}
	a := d.CheckStatusCode("ep1", 403, baseline)
	if a == nil {
		t.Fatal("Expected anomaly for 403")
	}
	if a.Severity != models.SeverityHigh {
		t.Errorf("Expected high severity for 403, got %s", a.Severity)
	}
}

func TestWatchAnomalyStatusCode_301Medium(t *testing.T) {
	d := watch.NewAnomalyDetector(2.0)
	baseline := &models.EndpointBaseline{
		StatusCodeDist: map[int]int{200: 100},
		SampleCount:    100,
	}
	// 301 is < 400, so severity defaults to medium.
	a := d.CheckStatusCode("ep1", 301, baseline)
	if a == nil {
		t.Fatal("Expected anomaly for unseen 301")
	}
	if a.Severity != models.SeverityMedium {
		t.Errorf("Expected medium severity for 301, got %s", a.Severity)
	}
}

func TestWatchAnomalyStatusCode_NilBaseline(t *testing.T) {
	d := watch.NewAnomalyDetector(2.0)
	a := d.CheckStatusCode("ep1", 500, nil)
	if a != nil {
		t.Error("Expected nil when baseline is nil")
	}
}

func TestWatchAnomalyStatusCode_EmptyDist(t *testing.T) {
	d := watch.NewAnomalyDetector(2.0)
	baseline := &models.EndpointBaseline{
		StatusCodeDist: map[int]int{},
		SampleCount:    10,
	}
	a := d.CheckStatusCode("ep1", 200, baseline)
	if a != nil {
		t.Error("Expected nil when status code distribution is empty")
	}
}

func TestWatchAnomalyStatusCode_MostCommonReported(t *testing.T) {
	d := watch.NewAnomalyDetector(2.0)
	baseline := &models.EndpointBaseline{
		StatusCodeDist: map[int]int{200: 80, 201: 20},
		SampleCount:    100,
	}
	a := d.CheckStatusCode("ep1", 500, baseline)
	if a == nil {
		t.Fatal("Expected anomaly for unseen 500")
	}
	// Expected field should report the most common status code (200).
	if a.Expected != 200 {
		t.Errorf("Expected most common code 200 in Expected, got %.0f", a.Expected)
	}
}

// ---------------------------------------------------------------------------
// DriftDetector.Compare — schema comparison
// ---------------------------------------------------------------------------

func TestWatchDrift_IdenticalSchemas(t *testing.T) {
	d := watch.NewDriftDetector()
	s := &models.Schema{
		Type: "object",
		Properties: map[string]*models.Schema{
			"id":   {Type: "integer"},
			"name": {Type: "string"},
		},
	}
	drifts := d.Compare("ep1", s, s)
	if len(drifts) != 0 {
		t.Errorf("Expected 0 drifts for identical schemas, got %d", len(drifts))
	}
}

func TestWatchDrift_TypeChanged(t *testing.T) {
	d := watch.NewDriftDetector()
	known := &models.Schema{
		Type: "object",
		Properties: map[string]*models.Schema{
			"count": {Type: "string"},
		},
	}
	actual := &models.Schema{
		Type: "object",
		Properties: map[string]*models.Schema{
			"count": {Type: "integer"},
		},
	}
	drifts := d.Compare("ep1", known, actual)
	if len(drifts) != 1 {
		t.Fatalf("Expected 1 drift for type change, got %d", len(drifts))
	}
	if drifts[0].Change != "type_changed" {
		t.Errorf("Expected 'type_changed', got %q", drifts[0].Change)
	}
	if drifts[0].OldValue != "string" || drifts[0].NewValue != "integer" {
		t.Errorf("Expected old=string new=integer, got old=%s new=%s", drifts[0].OldValue, drifts[0].NewValue)
	}
	if drifts[0].Severity != models.SeverityHigh {
		t.Errorf("Expected high severity for type change, got %s", drifts[0].Severity)
	}
}

func TestWatchDrift_FieldAdded(t *testing.T) {
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
		t.Errorf("Expected 'added', got %q", drifts[0].Change)
	}
	if drifts[0].Field != "new_field" {
		t.Errorf("Expected field 'new_field', got %q", drifts[0].Field)
	}
	if drifts[0].NewValue != "string" {
		t.Errorf("Expected NewValue='string', got %q", drifts[0].NewValue)
	}
	if drifts[0].Severity != models.SeverityLow {
		t.Errorf("Expected low severity for added field, got %s", drifts[0].Severity)
	}
}

func TestWatchDrift_FieldRemoved(t *testing.T) {
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
		},
	}
	drifts := d.Compare("ep1", known, actual)
	if len(drifts) != 1 {
		t.Fatalf("Expected 1 drift (removed field), got %d", len(drifts))
	}
	if drifts[0].Change != "removed" {
		t.Errorf("Expected 'removed', got %q", drifts[0].Change)
	}
	if drifts[0].Field != "email" {
		t.Errorf("Expected field 'email', got %q", drifts[0].Field)
	}
	if drifts[0].OldValue != "string" {
		t.Errorf("Expected OldValue='string', got %q", drifts[0].OldValue)
	}
	if drifts[0].Severity != models.SeverityHigh {
		t.Errorf("Expected high severity for removed field, got %s", drifts[0].Severity)
	}
}

func TestWatchDrift_FormatChanged(t *testing.T) {
	d := watch.NewDriftDetector()
	known := &models.Schema{
		Type: "object",
		Properties: map[string]*models.Schema{
			"created_at": {Type: "string", Format: "date-time"},
		},
	}
	actual := &models.Schema{
		Type: "object",
		Properties: map[string]*models.Schema{
			"created_at": {Type: "string", Format: "date"},
		},
	}
	drifts := d.Compare("ep1", known, actual)
	if len(drifts) != 1 {
		t.Fatalf("Expected 1 drift (format changed), got %d", len(drifts))
	}
	if drifts[0].Change != "format_changed" {
		t.Errorf("Expected 'format_changed', got %q", drifts[0].Change)
	}
	if drifts[0].OldValue != "date-time" || drifts[0].NewValue != "date" {
		t.Errorf("Expected old=date-time new=date, got old=%s new=%s", drifts[0].OldValue, drifts[0].NewValue)
	}
	if drifts[0].Severity != models.SeverityMedium {
		t.Errorf("Expected medium severity for format change, got %s", drifts[0].Severity)
	}
}

func TestWatchDrift_NestedPropertyChange(t *testing.T) {
	d := watch.NewDriftDetector()
	known := &models.Schema{
		Type: "object",
		Properties: map[string]*models.Schema{
			"user": {
				Type: "object",
				Properties: map[string]*models.Schema{
					"name": {Type: "string"},
					"age":  {Type: "integer"},
				},
			},
		},
	}
	actual := &models.Schema{
		Type: "object",
		Properties: map[string]*models.Schema{
			"user": {
				Type: "object",
				Properties: map[string]*models.Schema{
					"name": {Type: "string"},
					"age":  {Type: "string"}, // type changed inside nested object
				},
			},
		},
	}
	drifts := d.Compare("ep1", known, actual)
	if len(drifts) != 1 {
		t.Fatalf("Expected 1 drift for nested type change, got %d", len(drifts))
	}
	if drifts[0].Change != "type_changed" {
		t.Errorf("Expected 'type_changed', got %q", drifts[0].Change)
	}
	if drifts[0].Field != "user.age" {
		t.Errorf("Expected field 'user.age', got %q", drifts[0].Field)
	}
}

func TestWatchDrift_NilSchemas(t *testing.T) {
	d := watch.NewDriftDetector()

	// Both nil.
	drifts := d.Compare("ep1", nil, nil)
	if len(drifts) != 0 {
		t.Errorf("Expected 0 drifts when both schemas nil, got %d", len(drifts))
	}

	// Known nil.
	drifts = d.Compare("ep1", nil, &models.Schema{Type: "object"})
	if len(drifts) != 0 {
		t.Errorf("Expected 0 drifts when known schema nil, got %d", len(drifts))
	}

	// Actual nil.
	drifts = d.Compare("ep1", &models.Schema{Type: "object"}, nil)
	if len(drifts) != 0 {
		t.Errorf("Expected 0 drifts when actual schema nil, got %d", len(drifts))
	}
}

func TestWatchDrift_RootTypeChanged(t *testing.T) {
	d := watch.NewDriftDetector()
	known := &models.Schema{Type: "object"}
	actual := &models.Schema{Type: "array"}
	drifts := d.Compare("ep1", known, actual)
	if len(drifts) != 1 {
		t.Fatalf("Expected 1 drift for root type change, got %d", len(drifts))
	}
	if drifts[0].Change != "type_changed" {
		t.Errorf("Expected 'type_changed', got %q", drifts[0].Change)
	}
	if drifts[0].Field != "(root)" {
		t.Errorf("Expected field '(root)', got %q", drifts[0].Field)
	}
}

func TestWatchDrift_ArrayItemsTypeChanged(t *testing.T) {
	d := watch.NewDriftDetector()
	known := &models.Schema{
		Type:  "array",
		Items: &models.Schema{Type: "integer"},
	}
	actual := &models.Schema{
		Type:  "array",
		Items: &models.Schema{Type: "string"},
	}
	drifts := d.Compare("ep1", known, actual)
	if len(drifts) != 1 {
		t.Fatalf("Expected 1 drift for array items type change, got %d", len(drifts))
	}
	if drifts[0].Change != "type_changed" {
		t.Errorf("Expected 'type_changed', got %q", drifts[0].Change)
	}
	if drifts[0].OldValue != "integer" || drifts[0].NewValue != "string" {
		t.Errorf("Expected old=integer new=string, got old=%s new=%s", drifts[0].OldValue, drifts[0].NewValue)
	}
}

func TestWatchDrift_MultipleChanges(t *testing.T) {
	d := watch.NewDriftDetector()
	known := &models.Schema{
		Type: "object",
		Properties: map[string]*models.Schema{
			"id":      {Type: "integer"},
			"name":    {Type: "string"},
			"removed": {Type: "boolean"},
		},
	}
	actual := &models.Schema{
		Type: "object",
		Properties: map[string]*models.Schema{
			"id":    {Type: "integer"},
			"name":  {Type: "integer"}, // type changed
			"added": {Type: "string"},  // added
			// "removed" is missing
		},
	}
	drifts := d.Compare("ep1", known, actual)
	if len(drifts) != 3 {
		t.Fatalf("Expected 3 drifts (type_changed + added + removed), got %d", len(drifts))
	}

	changes := make(map[string]bool)
	for _, drift := range drifts {
		changes[drift.Change] = true
	}
	for _, expected := range []string{"type_changed", "added", "removed"} {
		if !changes[expected] {
			t.Errorf("Expected change %q not found in drifts", expected)
		}
	}
}

// ---------------------------------------------------------------------------
// Alerter — ParseTargets
// ---------------------------------------------------------------------------

func TestWatchParseTargets_Stdout(t *testing.T) {
	targets := watch.ParseTargets("stdout")
	if len(targets) != 1 {
		t.Fatalf("Expected 1 target, got %d", len(targets))
	}
	if targets[0].Type() != "stdout" {
		t.Errorf("Expected stdout target, got %s", targets[0].Type())
	}
}

func TestWatchParseTargets_Webhook(t *testing.T) {
	targets := watch.ParseTargets("webhook:http://example.com")
	if len(targets) != 1 {
		t.Fatalf("Expected 1 target, got %d", len(targets))
	}
	if targets[0].Type() != "webhook" {
		t.Errorf("Expected webhook target, got %s", targets[0].Type())
	}
}

func TestWatchParseTargets_Slack(t *testing.T) {
	targets := watch.ParseTargets("slack:http://hooks.slack.com/xxx")
	if len(targets) != 1 {
		t.Fatalf("Expected 1 target, got %d", len(targets))
	}
	if targets[0].Type() != "slack" {
		t.Errorf("Expected slack target, got %s", targets[0].Type())
	}
}

func TestWatchParseTargets_MultipleSeparated(t *testing.T) {
	targets := watch.ParseTargets("stdout,webhook:https://example.com,slack:https://hooks.slack.com/xxx")
	if len(targets) != 3 {
		t.Fatalf("Expected 3 targets, got %d", len(targets))
	}
	types := make(map[string]bool)
	for _, tgt := range targets {
		types[tgt.Type()] = true
	}
	for _, expected := range []string{"stdout", "webhook", "slack"} {
		if !types[expected] {
			t.Errorf("Expected target type %q not found", expected)
		}
	}
}

func TestWatchParseTargets_EmptyString(t *testing.T) {
	targets := watch.ParseTargets("")
	if len(targets) != 1 {
		t.Fatalf("Expected 1 default target, got %d", len(targets))
	}
	if targets[0].Type() != "stdout" {
		t.Errorf("Expected default stdout target, got %s", targets[0].Type())
	}
}

func TestWatchParseTargets_WithSpaces(t *testing.T) {
	targets := watch.ParseTargets("stdout , webhook:https://example.com")
	if len(targets) != 2 {
		t.Fatalf("Expected 2 targets, got %d", len(targets))
	}
	if targets[0].Type() != "stdout" {
		t.Errorf("Expected first target stdout, got %s", targets[0].Type())
	}
	if targets[1].Type() != "webhook" {
		t.Errorf("Expected second target webhook, got %s", targets[1].Type())
	}
}

func TestWatchParseTargets_UnknownDefaultsToStdout(t *testing.T) {
	targets := watch.ParseTargets("unknown_target")
	if len(targets) != 1 {
		t.Fatalf("Expected 1 target, got %d", len(targets))
	}
	if targets[0].Type() != "stdout" {
		t.Errorf("Expected stdout fallback for unknown target, got %s", targets[0].Type())
	}
}

// ---------------------------------------------------------------------------
// Alerter — HasTargets
// ---------------------------------------------------------------------------

func TestWatchAlerterHasTargets_WithTargets(t *testing.T) {
	targets := watch.ParseTargets("stdout")
	alerter := watch.NewAlerter(targets...)
	if !alerter.HasTargets() {
		t.Error("Expected HasTargets() to return true when targets are configured")
	}
}

func TestWatchAlerterHasTargets_NoTargets(t *testing.T) {
	alerter := watch.NewAlerter()
	if alerter.HasTargets() {
		t.Error("Expected HasTargets() to return false when no targets")
	}
}

// ---------------------------------------------------------------------------
// NewAnomalyDetector — threshold validation
// ---------------------------------------------------------------------------

func TestWatchNewAnomalyDetector_NegativeThreshold(t *testing.T) {
	// Negative threshold should be corrected to default 2.0.
	d := watch.NewAnomalyDetector(-1.0)
	baseline := &models.EndpointBaseline{
		AvgResponseTime: 100 * time.Millisecond,
		P50ResponseTime: 90 * time.Millisecond,
		P95ResponseTime: 200 * time.Millisecond,
		SampleCount:     100,
	}
	// With threshold 2.0, a normal response should not trigger.
	a := d.CheckResponseTime("ep1", 120*time.Millisecond, baseline)
	if a != nil {
		t.Error("Expected nil for normal response time with corrected threshold")
	}
}
