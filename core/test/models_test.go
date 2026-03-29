package test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/mkaganm/probex/internal/models"
)

func TestDefaultConfig(t *testing.T) {
	cfg := models.DefaultConfig()

	if cfg.AI.Mode != "offline" {
		t.Errorf("AI.Mode: got %s, want offline", cfg.AI.Mode)
	}
	if cfg.AI.Local.Provider != "ollama" {
		t.Errorf("AI.Local.Provider: got %s, want ollama", cfg.AI.Local.Provider)
	}
	if cfg.AI.Local.Model != "qwen3:4b" {
		t.Errorf("AI.Local.Model: got %s, want qwen3:4b", cfg.AI.Local.Model)
	}
	if cfg.AI.Cloud.Provider != "anthropic" {
		t.Errorf("AI.Cloud.Provider: got %s, want anthropic", cfg.AI.Cloud.Provider)
	}
	if cfg.AI.Budget.MaxMonthlyCost != 20 {
		t.Errorf("AI.Budget.MaxMonthlyCost: got %f, want 20", cfg.AI.Budget.MaxMonthlyCost)
	}
	if cfg.Scan.Concurrency != 10 {
		t.Errorf("Scan.Concurrency: got %d, want 10", cfg.Scan.Concurrency)
	}
	if cfg.Run.Concurrency != 5 {
		t.Errorf("Run.Concurrency: got %d, want 5", cfg.Run.Concurrency)
	}
	if cfg.Watch.Interval != 5*time.Minute {
		t.Errorf("Watch.Interval: got %v, want 5m", cfg.Watch.Interval)
	}
}

func TestConfigJSONRoundtrip(t *testing.T) {
	cfg := models.DefaultConfig()
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}

	var decoded models.Config
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.AI.Mode != cfg.AI.Mode {
		t.Errorf("AI.Mode roundtrip failed: got %s, want %s", decoded.AI.Mode, cfg.AI.Mode)
	}
}

func TestTestCaseCategories(t *testing.T) {
	categories := []models.TestCategory{
		models.CategoryHappyPath,
		models.CategoryEdgeCase,
		models.CategorySecurity,
		models.CategoryFuzz,
		models.CategoryRelation,
		models.CategoryConcurrency,
	}

	for _, c := range categories {
		if string(c) == "" {
			t.Error("category should not be empty")
		}
	}
}

func TestSeverityLevels(t *testing.T) {
	levels := []models.Severity{
		models.SeverityCritical,
		models.SeverityHigh,
		models.SeverityMedium,
		models.SeverityLow,
		models.SeverityInfo,
	}

	for _, l := range levels {
		if string(l) == "" {
			t.Error("severity should not be empty")
		}
	}
}

func TestEndpointDiscoverySources(t *testing.T) {
	sources := []models.DiscoverySource{
		models.SourceOpenAPI,
		models.SourceCrawl,
		models.SourceWordlist,
		models.SourceTraffic,
		models.SourceGraphQL,
		models.SourceWebSocket,
		models.SourceGRPC,
	}

	for _, s := range sources {
		if string(s) == "" {
			t.Error("source should not be empty")
		}
	}
}

func TestTestResultStatuses(t *testing.T) {
	statuses := []models.TestStatus{
		models.StatusPassed,
		models.StatusFailed,
		models.StatusError,
		models.StatusSkipped,
	}

	for _, s := range statuses {
		if string(s) == "" {
			t.Error("status should not be empty")
		}
	}
}

func TestRunSummaryJSONRoundtrip(t *testing.T) {
	summary := &models.RunSummary{
		TotalTests: 10,
		Passed:     7,
		Failed:     2,
		Errors:     1,
		Duration:   5 * time.Second,
		Results: []models.TestResult{
			{
				TestName: "test-1",
				Status:   models.StatusPassed,
				Category: models.CategoryHappyPath,
				Severity: models.SeverityMedium,
				Duration: 200 * time.Millisecond,
			},
		},
	}

	data, err := json.Marshal(summary)
	if err != nil {
		t.Fatal(err)
	}

	var decoded models.RunSummary
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.TotalTests != 10 {
		t.Errorf("TotalTests: got %d, want 10", decoded.TotalTests)
	}
	if len(decoded.Results) != 1 {
		t.Errorf("Results: got %d, want 1", len(decoded.Results))
	}
	if decoded.Results[0].Status != models.StatusPassed {
		t.Errorf("Result status: got %s, want passed", decoded.Results[0].Status)
	}
}

func TestAPIProfileJSONRoundtrip(t *testing.T) {
	profile := &models.APIProfile{
		ID:      "p1",
		Name:    "Test API",
		BaseURL: "http://example.com",
		Endpoints: []models.Endpoint{
			{
				Method:  "GET",
				Path:    "/users",
				BaseURL: "http://example.com",
				Auth: &models.AuthInfo{
					Type:     models.AuthBearer,
					Location: "header",
					Key:      "Authorization",
				},
				RequestBody: &models.Schema{
					Type: "object",
					Properties: map[string]*models.Schema{
						"name": {Type: "string"},
					},
				},
			},
		},
	}

	data, err := json.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}

	var decoded models.APIProfile
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.ID != "p1" {
		t.Errorf("ID: got %s, want p1", decoded.ID)
	}
	if len(decoded.Endpoints) != 1 {
		t.Fatalf("Endpoints: got %d, want 1", len(decoded.Endpoints))
	}
	if decoded.Endpoints[0].Auth.Type != models.AuthBearer {
		t.Errorf("Auth.Type: got %s, want bearer", decoded.Endpoints[0].Auth.Type)
	}
}
