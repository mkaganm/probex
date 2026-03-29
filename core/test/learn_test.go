package test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mkaganm/probex/internal/learn"
	"github.com/mkaganm/probex/internal/models"
)

// sampleHAR generates a minimal valid HAR file for testing.
func sampleHAR() []byte {
	har := map[string]any{
		"log": map[string]any{
			"version": "1.2",
			"creator": map[string]any{"name": "test", "version": "1.0"},
			"entries": []map[string]any{
				{
					"startedDateTime": "2024-01-15T10:00:00Z",
					"time":            120.5,
					"request": map[string]any{
						"method":      "GET",
						"url":         "https://api.example.com/users",
						"httpVersion": "HTTP/1.1",
						"headers": []map[string]any{
							{"name": "Authorization", "value": "Bearer token123"},
							{"name": "Content-Type", "value": "application/json"},
						},
						"queryString": []map[string]any{
							{"name": "page", "value": "1"},
						},
						"headersSize": -1,
						"bodySize":    0,
					},
					"response": map[string]any{
						"status":      200,
						"statusText":  "OK",
						"httpVersion": "HTTP/1.1",
						"headers":     []map[string]any{{"name": "Content-Type", "value": "application/json"}},
						"content": map[string]any{
							"size":     100,
							"mimeType": "application/json",
							"text":     `[{"id": 1, "name": "Alice", "email": "alice@example.com"}, {"id": 2, "name": "Bob", "email": "bob@example.com"}]`,
						},
						"headersSize": -1,
						"bodySize":    100,
					},
					"timings": map[string]any{
						"blocked": 0, "dns": 5.0, "connect": 10.0,
						"send": 1.0, "wait": 100.0, "receive": 5.0, "ssl": 8.0,
					},
				},
				{
					"startedDateTime": "2024-01-15T10:00:01Z",
					"time":            80.0,
					"request": map[string]any{
						"method":      "POST",
						"url":         "https://api.example.com/users",
						"httpVersion": "HTTP/1.1",
						"headers": []map[string]any{
							{"name": "Authorization", "value": "Bearer token123"},
							{"name": "Content-Type", "value": "application/json"},
						},
						"queryString": []any{},
						"postData": map[string]any{
							"mimeType": "application/json",
							"text":     `{"name": "Charlie", "email": "charlie@example.com"}`,
						},
						"headersSize": -1,
						"bodySize":    50,
					},
					"response": map[string]any{
						"status":      201,
						"statusText":  "Created",
						"httpVersion": "HTTP/1.1",
						"headers":     []map[string]any{{"name": "Content-Type", "value": "application/json"}},
						"content": map[string]any{
							"size":     60,
							"mimeType": "application/json",
							"text":     `{"id": 3, "name": "Charlie", "email": "charlie@example.com"}`,
						},
						"headersSize": -1,
						"bodySize":    60,
					},
					"timings": map[string]any{
						"blocked": 0, "dns": -1, "connect": -1,
						"send": 2.0, "wait": 70.0, "receive": 3.0, "ssl": -1,
					},
				},
				{
					"startedDateTime": "2024-01-15T10:00:02Z",
					"time":            50.0,
					"request": map[string]any{
						"method":      "GET",
						"url":         "https://api.example.com/users/3",
						"httpVersion": "HTTP/1.1",
						"headers": []map[string]any{
							{"name": "Authorization", "value": "Bearer token123"},
						},
						"queryString": []any{},
						"headersSize": -1,
						"bodySize":    0,
					},
					"response": map[string]any{
						"status":      200,
						"statusText":  "OK",
						"httpVersion": "HTTP/1.1",
						"headers":     []map[string]any{{"name": "Content-Type", "value": "application/json"}},
						"content": map[string]any{
							"size":     60,
							"mimeType": "application/json",
							"text":     `{"id": 3, "name": "Charlie", "email": "charlie@example.com"}`,
						},
						"headersSize": -1,
						"bodySize":    60,
					},
					"timings": map[string]any{
						"blocked": 0, "dns": -1, "connect": -1,
						"send": 1.0, "wait": 45.0, "receive": 2.0, "ssl": -1,
					},
				},
			},
		},
	}
	data, _ := json.MarshalIndent(har, "", "  ")
	return data
}

func writeHARFile(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, sampleHAR(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseHARData(t *testing.T) {
	parsed, err := learn.ParseHARData(sampleHAR())
	if err != nil {
		t.Fatalf("ParseHARData: %v", err)
	}

	if len(parsed.Ordered) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(parsed.Ordered))
	}

	if len(parsed.Endpoints) == 0 {
		t.Fatal("expected at least 1 endpoint")
	}

	// Should have grouped entries (GET /users, POST /users, GET /users/{id}).
	if len(parsed.Grouped) < 2 {
		t.Errorf("expected at least 2 endpoint groups, got %d", len(parsed.Grouped))
	}
}

func TestParseHARDataInvalid(t *testing.T) {
	_, err := learn.ParseHARData([]byte(`{invalid}`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseHARFileFromDisk(t *testing.T) {
	dir := t.TempDir()
	path := writeHARFile(t, dir, "test.har")

	parsed, err := learn.ParseHARFile(path)
	if err != nil {
		t.Fatalf("ParseHARFile: %v", err)
	}
	if len(parsed.Ordered) != 3 {
		t.Errorf("expected 3 entries, got %d", len(parsed.Ordered))
	}
}

func TestEndpointAuthDetection(t *testing.T) {
	parsed, err := learn.ParseHARData(sampleHAR())
	if err != nil {
		t.Fatal(err)
	}

	foundBearer := false
	for _, ep := range parsed.Endpoints {
		if ep.Auth != nil && ep.Auth.Type == models.AuthBearer {
			foundBearer = true
			break
		}
	}
	if !foundBearer {
		t.Error("expected Bearer auth to be detected from Authorization header")
	}
}

func TestEndpointQueryParams(t *testing.T) {
	parsed, err := learn.ParseHARData(sampleHAR())
	if err != nil {
		t.Fatal(err)
	}

	for _, ep := range parsed.Endpoints {
		if ep.Method == "GET" && ep.Path == "/users" {
			if len(ep.QueryParams) == 0 {
				t.Error("expected query params on GET /users")
			}
			return
		}
	}
}

func TestEndpointRequestBodySchema(t *testing.T) {
	parsed, err := learn.ParseHARData(sampleHAR())
	if err != nil {
		t.Fatal(err)
	}

	for _, ep := range parsed.Endpoints {
		if ep.Method == "POST" {
			if ep.RequestBody == nil {
				t.Error("expected request body schema on POST endpoint")
			} else if ep.RequestBody.Type != "object" {
				t.Errorf("expected object schema, got %s", ep.RequestBody.Type)
			}
			return
		}
	}
	t.Error("POST endpoint not found")
}

func TestLearnerLearn(t *testing.T) {
	dir := t.TempDir()
	writeHARFile(t, dir, "traffic.har")

	learner := learn.NewLearner()
	result, err := learner.Learn(context.Background(), dir, nil)
	if err != nil {
		t.Fatalf("Learn: %v", err)
	}

	if result.HARFilesRead != 1 {
		t.Errorf("HARFilesRead: got %d, want 1", result.HARFilesRead)
	}
	if result.EntriesAnalyzed != 3 {
		t.Errorf("EntriesAnalyzed: got %d, want 3", result.EntriesAnalyzed)
	}
	if result.Profile == nil {
		t.Fatal("expected non-nil profile")
	}
	if result.Profile.BaseURL != "https://api.example.com" {
		t.Errorf("BaseURL: got %s, want https://api.example.com", result.Profile.BaseURL)
	}
	if len(result.Profile.Endpoints) == 0 {
		t.Error("expected endpoints in profile")
	}
	if result.TrafficAnalysis == nil {
		t.Error("expected non-nil TrafficAnalysis")
	}
	if result.PatternReport == nil {
		t.Error("expected non-nil PatternReport")
	}
}

func TestLearnerLearnWithExistingProfile(t *testing.T) {
	dir := t.TempDir()
	writeHARFile(t, dir, "traffic.har")

	existing := &models.APIProfile{
		ID:      "existing",
		BaseURL: "https://api.example.com",
		Endpoints: []models.Endpoint{
			{Method: "GET", Path: "/health", BaseURL: "https://api.example.com"},
		},
	}

	learner := learn.NewLearner()
	result, err := learner.Learn(context.Background(), dir, existing)
	if err != nil {
		t.Fatal(err)
	}

	if result.Profile.ID != "existing" {
		t.Errorf("expected existing profile ID preserved, got %s", result.Profile.ID)
	}
	// Should have the original /health endpoint plus new ones from HAR.
	if len(result.Profile.Endpoints) <= 1 {
		t.Errorf("expected more than 1 endpoint after merge, got %d", len(result.Profile.Endpoints))
	}
}

func TestLearnerNoHARFiles(t *testing.T) {
	dir := t.TempDir()
	learner := learn.NewLearner()
	_, err := learner.Learn(context.Background(), dir, nil)
	if err == nil {
		t.Error("expected error when no HAR files found")
	}
}

func TestLearnerContextCancellation(t *testing.T) {
	dir := t.TempDir()
	writeHARFile(t, dir, "traffic.har")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	learner := learn.NewLearner()
	_, err := learner.Learn(ctx, dir, nil)
	if err == nil {
		t.Error("expected error on cancelled context")
	}
}

func TestAnalyzeTraffic(t *testing.T) {
	parsed, err := learn.ParseHARData(sampleHAR())
	if err != nil {
		t.Fatal(err)
	}

	analysis := learn.AnalyzeTraffic(parsed)

	if len(analysis.Frequency) == 0 {
		t.Error("expected non-empty frequency map")
	}

	// Total frequency should equal total entries.
	var total int
	for _, count := range analysis.Frequency {
		total += count
	}
	if total != 3 {
		t.Errorf("total frequency: got %d, want 3", total)
	}
}

func TestBuildBaseline(t *testing.T) {
	parsed, err := learn.ParseHARData(sampleHAR())
	if err != nil {
		t.Fatal(err)
	}

	baseline := learn.BuildBaseline(parsed.Grouped)

	if len(baseline.Endpoints) == 0 {
		t.Fatal("expected at least 1 endpoint baseline")
	}

	for key, eb := range baseline.Endpoints {
		if eb.SampleCount <= 0 {
			t.Errorf("%s: expected positive SampleCount", key)
		}
		if eb.AvgResponseTime <= 0 {
			t.Errorf("%s: expected positive AvgResponseTime", key)
		}
	}
}

func TestMinePatterns(t *testing.T) {
	parsed, err := learn.ParseHARData(sampleHAR())
	if err != nil {
		t.Fatal(err)
	}

	report := learn.MinePatterns(parsed.Grouped)

	if report == nil {
		t.Fatal("expected non-nil PatternReport")
	}

	// Should detect email patterns in response bodies.
	foundEmail := false
	for _, patterns := range report.Endpoints {
		for _, p := range patterns {
			if p.Format == "email" {
				foundEmail = true
				break
			}
		}
	}
	if !foundEmail {
		t.Error("expected email pattern to be detected in response bodies")
	}
}
