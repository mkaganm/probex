package test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mkaganm/probex/internal/server"
)

// newTestServer creates a server backed by a temp storage directory.
// It returns the server and a cleanup function.
func newTestServer(t *testing.T, opts ...server.Option) *httptest.Server {
	t.Helper()
	t.Setenv("PROBEX_DIR", t.TempDir()) // avoid polluting the real .probex

	// Override the storage dir by changing working directory.
	origDir := changeToTempDir(t)
	t.Cleanup(func() { _ = chdir(origDir) })

	srv, err := server.New("127.0.0.1:0", opts...)
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}

	// Use httptest to wrap the internal http.Handler for testing.
	return httptest.NewServer(srv.Handler())
}

func TestHealthEndpoint(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if body["status"] != "ok" {
		t.Errorf("expected status ok, got %v", body["status"])
	}
	if body["ai_enabled"] != false {
		t.Errorf("expected ai_enabled=false without AI, got %v", body["ai_enabled"])
	}
}

func TestGetProfileNotFound(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/profile")
	if err != nil {
		t.Fatalf("GET /profile: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestScanMissingBaseURL(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	body := `{}`
	resp, err := http.Post(ts.URL+"/api/v1/scan", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("POST /scan: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for missing base_url, got %d", resp.StatusCode)
	}
}

func TestScanInvalidJSON(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/api/v1/scan", "application/json", bytes.NewBufferString("{invalid"))
	if err != nil {
		t.Fatalf("POST /scan: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", resp.StatusCode)
	}
}

func TestRunWithoutProfile(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	body := `{"concurrency": 2}`
	resp, err := http.Post(ts.URL+"/api/v1/run", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("POST /run: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPreconditionFailed {
		t.Errorf("expected 412 without profile, got %d", resp.StatusCode)
	}
}

func TestGetResultsNotFound(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/results")
	if err != nil {
		t.Fatalf("GET /results: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 without results, got %d", resp.StatusCode)
	}
}

func TestGetResultByIDNotFound(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/results/nonexistent")
	if err != nil {
		t.Fatalf("GET /results/id: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

// --- AI endpoint tests ---

func TestAIEndpointsReturn503WithoutAI(t *testing.T) {
	ts := newTestServer(t) // no AI options
	defer ts.Close()

	endpoints := []struct {
		method string
		path   string
		body   string
	}{
		{"GET", "/api/v1/ai/health", ""},
		{"POST", "/api/v1/ai/scenarios", `{"endpoints": []}`},
		{"POST", "/api/v1/ai/security", `{"endpoints": []}`},
		{"POST", "/api/v1/ai/nl-to-test", `{"description": "test"}`},
		{"POST", "/api/v1/ai/anomaly", `{"endpoint_id": "e1"}`},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			var resp *http.Response
			var err error

			if ep.method == "GET" {
				resp, err = http.Get(ts.URL + ep.path)
			} else {
				resp, err = http.Post(ts.URL+ep.path, "application/json", bytes.NewBufferString(ep.body))
			}
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusServiceUnavailable {
				t.Errorf("expected 503 without AI, got %d", resp.StatusCode)
			}
		})
	}
}

func TestAIEndpointsWithMockBrain(t *testing.T) {
	// Create a mock AI brain that responds to all endpoints.
	brain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/health":
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status":  "ok",
				"version": "0.4.0",
				"ai_mode": "local",
				"model":   "test-model",
			})
		case "/api/v1/scenarios/generate":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"scenarios":   []map[string]any{{"name": "test-scenario", "description": "desc", "category": "happy_path", "severity": "medium", "request": map[string]any{"method": "GET", "url": "/test"}, "assertions": []any{}}},
				"model_used":  "test-model",
				"tokens_used": 100,
			})
		case "/api/v1/security/analyze":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"findings":    []any{},
				"model_used":  "test-model",
				"tokens_used": 50,
			})
		case "/api/v1/nl-to-test":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"test_cases":  []map[string]any{{"name": "nl-test", "description": "from NL", "category": "happy_path", "severity": "medium", "request": map[string]any{"method": "GET", "url": "/test"}, "assertions": []any{}}},
				"model_used":  "test-model",
				"tokens_used": 75,
			})
		case "/api/v1/anomaly/classify":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"classification": "normal",
				"confidence":     0.95,
				"explanation":    "within normal bounds",
				"severity":       "low",
				"model_used":     "test-model",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer brain.Close()

	ts := newTestServer(t, server.WithAIURL(brain.URL))
	defer ts.Close()

	t.Run("ai health", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/ai/health")
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}

		var body map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&body)
		if body["ai_mode"] != "local" {
			t.Errorf("expected ai_mode=local, got %v", body["ai_mode"])
		}
	})

	t.Run("ai health reflected in main health", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/health")
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		defer resp.Body.Close()

		var body map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&body)
		if body["ai_enabled"] != true {
			t.Errorf("expected ai_enabled=true with AI configured, got %v", body["ai_enabled"])
		}
	})

	t.Run("ai scenarios", func(t *testing.T) {
		reqBody := `{"endpoints": [{"method": "GET", "path": "/users", "base_url": "http://example.com"}], "max_scenarios": 5}`
		resp, err := http.Post(ts.URL+"/api/v1/ai/scenarios", "application/json", bytes.NewBufferString(reqBody))
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}

		var body map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&body)
		scenarios, ok := body["scenarios"].([]any)
		if !ok || len(scenarios) == 0 {
			t.Error("expected non-empty scenarios")
		}
	})

	t.Run("ai scenarios empty endpoints", func(t *testing.T) {
		resp, err := http.Post(ts.URL+"/api/v1/ai/scenarios", "application/json", bytes.NewBufferString(`{"endpoints": []}`))
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
	})

	t.Run("ai security", func(t *testing.T) {
		reqBody := `{"endpoints": [{"method": "POST", "path": "/login", "base_url": "http://example.com"}]}`
		resp, err := http.Post(ts.URL+"/api/v1/ai/security", "application/json", bytes.NewBufferString(reqBody))
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
	})

	t.Run("ai nl-to-test", func(t *testing.T) {
		reqBody := `{"description": "user should get 401 without auth token"}`
		resp, err := http.Post(ts.URL+"/api/v1/ai/nl-to-test", "application/json", bytes.NewBufferString(reqBody))
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}

		var body map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&body)
		tests, ok := body["test_cases"].([]any)
		if !ok || len(tests) == 0 {
			t.Error("expected non-empty test_cases")
		}
	})

	t.Run("ai nl-to-test missing description", func(t *testing.T) {
		resp, err := http.Post(ts.URL+"/api/v1/ai/nl-to-test", "application/json", bytes.NewBufferString(`{}`))
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
	})

	t.Run("ai anomaly", func(t *testing.T) {
		reqBody := `{"endpoint_id": "e1", "observed_status": 500, "expected_status": 200}`
		resp, err := http.Post(ts.URL+"/api/v1/ai/anomaly", "application/json", bytes.NewBufferString(reqBody))
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}

		var body map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&body)
		if body["classification"] != "normal" {
			t.Errorf("expected classification=normal, got %v", body["classification"])
		}
	})

	t.Run("ai anomaly missing endpoint_id", func(t *testing.T) {
		resp, err := http.Post(ts.URL+"/api/v1/ai/anomaly", "application/json", bytes.NewBufferString(`{}`))
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
	})
}

// Test that AI endpoints handle brain errors gracefully.
func TestAIEndpointsBrainError(t *testing.T) {
	// Mock brain that always returns 500.
	brain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer brain.Close()

	ts := newTestServer(t, server.WithAIURL(brain.URL))
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/api/v1/ai/nl-to-test", "application/json",
		bytes.NewBufferString(`{"description": "test"}`))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("expected 502 on brain error, got %d", resp.StatusCode)
	}
}

// --- helpers ---

func changeToTempDir(t *testing.T) string {
	t.Helper()
	orig, err := getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := chdir(dir); err != nil {
		t.Fatal(err)
	}
	return orig
}

// Wrappers to avoid import os in multiple places.
func getwd() (string, error) { return _getwd() }
func chdir(dir string) error { return _chdir(dir) }

// Unused context to suppress linter.
var _ = context.Background
