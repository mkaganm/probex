package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mkaganm/probex/internal/ai"
	"github.com/mkaganm/probex/internal/server"
)

// --- AI Client ↔ mock brain integration tests ---

// startMockBrain starts an httptest server that mimics the Python brain FastAPI.
func startMockBrain(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"version": "0.4.0-test",
			"ai_mode": "local",
			"model":   "test-model",
		})
	})

	mux.HandleFunc("/api/v1/scenarios/generate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"scenarios": []map[string]any{
				{
					"name":        "test_scenario_1",
					"description": "AI generated scenario",
					"category":    "happy_path",
					"severity":    "medium",
					"request":     map[string]any{"method": "GET", "url": "/api/users"},
					"assertions":  []map[string]any{{"type": "status_code", "expected": "200"}},
				},
			},
			"model_used":  "test-model",
			"tokens_used": 42,
		})
	})

	mux.HandleFunc("/api/v1/security/analyze", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"findings": []map[string]any{
				{
					"title":       "Missing Rate Limiting",
					"description": "No rate limit on POST /api/users",
					"severity":    "high",
					"category":    "security",
					"endpoint":    "POST /api/users",
				},
			},
			"model_used":  "test-model",
			"tokens_used": 55,
		})
	})

	mux.HandleFunc("/api/v1/nl-to-test", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"test_cases": []map[string]any{
				{
					"name":        "nl_generated_test",
					"description": "Test from NL description",
					"category":    "happy_path",
					"severity":    "low",
					"request":     map[string]any{"method": "GET", "url": "/api/health"},
					"assertions":  []map[string]any{{"type": "status_code", "expected": "200"}},
				},
			},
			"model_used":  "test-model",
			"tokens_used": 30,
		})
	})

	mux.HandleFunc("/api/v1/anomaly/classify", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"classification": "degradation",
			"confidence":     0.85,
			"explanation":    "Response time significantly above baseline",
			"severity":       "high",
			"model_used":     "test-model",
		})
	})

	return httptest.NewServer(mux)
}

func TestIntegrationAIClientHealth(t *testing.T) {
	brain := startMockBrain(t)
	defer brain.Close()

	client := ai.NewClient(brain.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.Health(ctx)
	if err != nil {
		t.Fatalf("Health() error: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("status = %q, want %q", resp.Status, "ok")
	}
	if resp.AIMode != "local" {
		t.Errorf("ai_mode = %q, want %q", resp.AIMode, "local")
	}
	if resp.Model != "test-model" {
		t.Errorf("model = %q, want %q", resp.Model, "test-model")
	}
}

func TestIntegrationAIClientScenarios(t *testing.T) {
	brain := startMockBrain(t)
	defer brain.Close()

	client := ai.NewClient(brain.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &ai.ScenarioRequest{
		Endpoints: []ai.EndpointInfo{
			{Method: "GET", Path: "/api/users", BaseURL: "http://example.com"},
		},
		MaxScenarios: 5,
	}

	resp, err := client.GenerateScenarios(ctx, req)
	if err != nil {
		t.Fatalf("GenerateScenarios() error: %v", err)
	}
	if len(resp.Scenarios) != 1 {
		t.Fatalf("scenarios count = %d, want 1", len(resp.Scenarios))
	}
	if resp.Scenarios[0].Name != "test_scenario_1" {
		t.Errorf("scenario name = %q, want %q", resp.Scenarios[0].Name, "test_scenario_1")
	}
	if resp.ModelUsed != "test-model" {
		t.Errorf("model_used = %q, want %q", resp.ModelUsed, "test-model")
	}
	if resp.TokensUsed != 42 {
		t.Errorf("tokens_used = %d, want 42", resp.TokensUsed)
	}
}

func TestIntegrationAIClientSecurity(t *testing.T) {
	brain := startMockBrain(t)
	defer brain.Close()

	client := ai.NewClient(brain.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &ai.SecurityAnalysisRequest{
		Endpoints: []ai.EndpointInfo{
			{Method: "POST", Path: "/api/users", BaseURL: "http://example.com"},
		},
		Depth: "standard",
	}

	resp, err := client.AnalyzeSecurity(ctx, req)
	if err != nil {
		t.Fatalf("AnalyzeSecurity() error: %v", err)
	}
	if len(resp.Findings) != 1 {
		t.Fatalf("findings count = %d, want 1", len(resp.Findings))
	}
	if resp.Findings[0].Severity != "high" {
		t.Errorf("finding severity = %q, want %q", resp.Findings[0].Severity, "high")
	}
}

func TestIntegrationAIClientNLToTest(t *testing.T) {
	brain := startMockBrain(t)
	defer brain.Close()

	client := ai.NewClient(brain.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &ai.NLTestRequest{
		Description: "Test that the health endpoint returns 200",
	}

	resp, err := client.NLToTest(ctx, req)
	if err != nil {
		t.Fatalf("NLToTest() error: %v", err)
	}
	if len(resp.TestCases) != 1 {
		t.Fatalf("test_cases count = %d, want 1", len(resp.TestCases))
	}
	if resp.TestCases[0].Name != "nl_generated_test" {
		t.Errorf("test name = %q, want %q", resp.TestCases[0].Name, "nl_generated_test")
	}
}

func TestIntegrationAIClientAnomaly(t *testing.T) {
	brain := startMockBrain(t)
	defer brain.Close()

	client := ai.NewClient(brain.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &ai.AnomalyClassifyRequest{
		EndpointID:     "/api/users",
		ObservedStatus: 200,
		ExpectedStatus: 200,
		ResponseTimeMs: 500,
		BaselineTimeMs: 100,
	}

	resp, err := client.ClassifyAnomaly(ctx, req)
	if err != nil {
		t.Fatalf("ClassifyAnomaly() error: %v", err)
	}
	if resp.Classification != "degradation" {
		t.Errorf("classification = %q, want %q", resp.Classification, "degradation")
	}
	if resp.Severity != "high" {
		t.Errorf("severity = %q, want %q", resp.Severity, "high")
	}
	if resp.Confidence < 0.8 {
		t.Errorf("confidence = %f, want >= 0.8", resp.Confidence)
	}
}

// --- Go Server ↔ mock brain full pipeline integration ---

func TestIntegrationServerWithAI(t *testing.T) {
	brain := startMockBrain(t)
	defer brain.Close()

	// Start Go server with AI pointing to mock brain.
	port := getFreePort(t)
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	srv, err := server.New(addr, server.WithAIURL(brain.URL))
	if err != nil {
		t.Fatalf("server.New() error: %v", err)
	}

	handler := srv.Handler()
	ts := httptest.NewServer(handler)
	defer ts.Close()

	client := &http.Client{Timeout: 10 * time.Second}

	// 1. Health should show AI enabled.
	resp, err := client.Get(ts.URL + "/api/v1/health")
	if err != nil {
		t.Fatalf("GET /health error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("GET /health status = %d, want 200", resp.StatusCode)
	}
	var health map[string]any
	json.NewDecoder(resp.Body).Decode(&health)
	if health["ai_enabled"] != true {
		t.Errorf("ai_enabled = %v, want true", health["ai_enabled"])
	}

	// 2. AI health endpoint proxies to brain.
	resp2, err := client.Get(ts.URL + "/api/v1/ai/health")
	if err != nil {
		t.Fatalf("GET /ai/health error: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != 200 {
		t.Fatalf("GET /ai/health status = %d, want 200", resp2.StatusCode)
	}
	var aiHealth map[string]any
	json.NewDecoder(resp2.Body).Decode(&aiHealth)
	if aiHealth["status"] != "ok" {
		t.Errorf("ai health status = %v, want ok", aiHealth["status"])
	}

	// 3. AI scenarios endpoint.
	scenarioBody := `{"endpoints":[{"method":"GET","path":"/api/users"}],"max_scenarios":5}`
	resp3, err := client.Post(ts.URL+"/api/v1/ai/scenarios", "application/json",
		bytes.NewBufferString(scenarioBody))
	if err != nil {
		t.Fatalf("POST /ai/scenarios error: %v", err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != 200 {
		t.Fatalf("POST /ai/scenarios status = %d, want 200", resp3.StatusCode)
	}

	// 4. AI NL-to-test endpoint.
	nlBody := `{"description":"Test health endpoint returns 200"}`
	resp4, err := client.Post(ts.URL+"/api/v1/ai/nl-to-test", "application/json",
		bytes.NewBufferString(nlBody))
	if err != nil {
		t.Fatalf("POST /ai/nl-to-test error: %v", err)
	}
	defer resp4.Body.Close()
	if resp4.StatusCode != 200 {
		t.Fatalf("POST /ai/nl-to-test status = %d, want 200", resp4.StatusCode)
	}
}

// --- Helper ---

func getFreePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("getFreePort: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}
