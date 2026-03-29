package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mkaganm/probex/internal/models"
	"github.com/mkaganm/probex/internal/report"
	"github.com/mkaganm/probex/internal/server"
)

// mockAPI creates a realistic mock REST API for E2E testing.
// It serves OpenAPI spec + standard CRUD endpoints.
func mockAPI() *httptest.Server {
	mux := http.NewServeMux()

	// OpenAPI spec for auto-discovery.
	mux.HandleFunc("GET /openapi.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"openapi": "3.0.0",
			"info":    map[string]any{"title": "Test API", "version": "1.0.0"},
			"paths": map[string]any{
				"/users": map[string]any{
					"get": map[string]any{
						"summary":   "List users",
						"responses": map[string]any{"200": map[string]any{"description": "OK"}},
					},
					"post": map[string]any{
						"summary":   "Create user",
						"responses": map[string]any{"201": map[string]any{"description": "Created"}},
					},
				},
				"/users/{id}": map[string]any{
					"get": map[string]any{
						"summary":   "Get user by ID",
						"responses": map[string]any{"200": map[string]any{"description": "OK"}},
					},
					"delete": map[string]any{
						"summary":   "Delete user",
						"responses": map[string]any{"204": map[string]any{"description": "Deleted"}},
					},
				},
				"/health": map[string]any{
					"get": map[string]any{
						"summary":   "Health check",
						"responses": map[string]any{"200": map[string]any{"description": "OK"}},
					},
				},
			},
		})
	})

	// Standard endpoint responses.
	mux.HandleFunc("GET /users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"id": 1, "name": "Alice", "email": "alice@example.com"},
			{"id": 2, "name": "Bob", "email": "bob@example.com"},
		})
	})

	mux.HandleFunc("POST /users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 3, "name": "Charlie", "email": "charlie@example.com",
		})
	})

	mux.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 1, "name": "Alice", "email": "alice@example.com",
		})
	})

	mux.HandleFunc("DELETE /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	})

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
	})

	// Catch-all for probed paths.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	return httptest.NewServer(mux)
}

// TestE2EScanRunReport tests the full pipeline: scan → run → get results → report.
func TestE2EScanRunReport(t *testing.T) {
	// Start mock API.
	api := mockAPI()
	defer api.Close()

	// Start PROBEX server in a temp directory.
	orig := changeToTempDir(t)
	t.Cleanup(func() { _ = chdir(orig) })

	srv, err := server.New("127.0.0.1:0")
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}
	probex := httptest.NewServer(srv.Handler())
	defer probex.Close()

	// --- Step 1: Scan the mock API ---
	t.Log("Step 1: Scanning mock API...")
	scanBody, _ := json.Marshal(map[string]any{
		"base_url":    api.URL,
		"max_depth":   1,
		"concurrency": 2,
	})

	scanResp, err := http.Post(probex.URL+"/api/v1/scan", "application/json", bytes.NewReader(scanBody))
	if err != nil {
		t.Fatalf("POST /scan: %v", err)
	}
	defer scanResp.Body.Close()

	if scanResp.StatusCode != http.StatusOK {
		var errBody map[string]string
		_ = json.NewDecoder(scanResp.Body).Decode(&errBody)
		t.Fatalf("scan failed with %d: %v", scanResp.StatusCode, errBody)
	}

	var profile models.APIProfile
	if err := json.NewDecoder(scanResp.Body).Decode(&profile); err != nil {
		t.Fatalf("decode scan response: %v", err)
	}

	t.Logf("  Discovered %d endpoints", len(profile.Endpoints))
	if len(profile.Endpoints) == 0 {
		t.Fatal("scan discovered no endpoints")
	}

	// Verify we found at least the key endpoints.
	foundMethods := make(map[string]bool)
	for _, ep := range profile.Endpoints {
		foundMethods[ep.Method] = true
		t.Logf("  - %s %s", ep.Method, ep.Path)
	}

	if !foundMethods["GET"] {
		t.Error("expected GET endpoints")
	}

	// --- Step 2: Get the saved profile ---
	t.Log("Step 2: Verifying profile was saved...")
	profileResp, err := http.Get(probex.URL + "/api/v1/profile")
	if err != nil {
		t.Fatalf("GET /profile: %v", err)
	}
	defer profileResp.Body.Close()

	if profileResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for profile, got %d", profileResp.StatusCode)
	}

	// --- Step 3: Run tests ---
	t.Log("Step 3: Running tests...")
	runBody, _ := json.Marshal(map[string]any{
		"concurrency": 3,
		"timeout":     10,
	})

	runResp, err := http.Post(probex.URL+"/api/v1/run", "application/json", bytes.NewReader(runBody))
	if err != nil {
		t.Fatalf("POST /run: %v", err)
	}
	defer runResp.Body.Close()

	if runResp.StatusCode != http.StatusOK {
		var errBody map[string]string
		_ = json.NewDecoder(runResp.Body).Decode(&errBody)
		t.Fatalf("run failed with %d: %v", runResp.StatusCode, errBody)
	}

	var summary models.RunSummary
	if err := json.NewDecoder(runResp.Body).Decode(&summary); err != nil {
		t.Fatalf("decode run response: %v", err)
	}

	t.Logf("  Total: %d, Passed: %d, Failed: %d, Errors: %d",
		summary.TotalTests, summary.Passed, summary.Failed, summary.Errors)

	if summary.TotalTests == 0 {
		t.Fatal("no tests were generated or executed")
	}

	// --- Step 4: Get results ---
	t.Log("Step 4: Getting results...")
	resultsResp, err := http.Get(probex.URL + "/api/v1/results")
	if err != nil {
		t.Fatalf("GET /results: %v", err)
	}
	defer resultsResp.Body.Close()

	if resultsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for results, got %d", resultsResp.StatusCode)
	}

	var storedSummary models.RunSummary
	if err := json.NewDecoder(resultsResp.Body).Decode(&storedSummary); err != nil {
		t.Fatalf("decode results: %v", err)
	}

	if storedSummary.TotalTests != summary.TotalTests {
		t.Errorf("stored total (%d) doesn't match run total (%d)",
			storedSummary.TotalTests, summary.TotalTests)
	}

	// --- Step 5: Generate reports in all formats ---
	t.Log("Step 5: Generating reports...")

	for _, format := range []string{"json", "junit", "html"} {
		r, err := report.NewReporter(format)
		if err != nil {
			t.Errorf("NewReporter(%s): %v", format, err)
			continue
		}
		var buf bytes.Buffer
		if err := r.Generate(&summary, &buf); err != nil {
			t.Errorf("%s report generation failed: %v", format, err)
			continue
		}
		if buf.Len() == 0 {
			t.Errorf("%s report is empty", format)
		}
		t.Logf("  %s report: %d bytes", format, buf.Len())
	}

	t.Log("E2E pipeline completed successfully!")
}

// TestE2EScanWithCategories tests running only specific test categories.
func TestE2EScanWithCategories(t *testing.T) {
	api := mockAPI()
	defer api.Close()

	orig := changeToTempDir(t)
	t.Cleanup(func() { _ = chdir(orig) })

	srv, err := server.New("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	probex := httptest.NewServer(srv.Handler())
	defer probex.Close()

	// Scan.
	scanBody, _ := json.Marshal(map[string]any{"base_url": api.URL, "max_depth": 1})
	resp, err := http.Post(probex.URL+"/api/v1/scan", "application/json", bytes.NewReader(scanBody))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	// Run only security tests.
	runBody, _ := json.Marshal(map[string]any{
		"categories": []string{"security"},
		"timeout":    10,
	})
	resp, err = http.Post(probex.URL+"/api/v1/run", "application/json", bytes.NewReader(runBody))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		t.Fatalf("run failed with %d: %v", resp.StatusCode, errBody)
	}

	var summary models.RunSummary
	_ = json.NewDecoder(resp.Body).Decode(&summary)

	t.Logf("Security-only run: %d tests", summary.TotalTests)

	// All results should be security category.
	for _, r := range summary.Results {
		if r.Category != models.CategorySecurity {
			t.Errorf("expected security category, got %s for %s", r.Category, r.TestName)
		}
	}
}

// TestE2EWithAIMockBrain tests the full pipeline with AI brain integration.
func TestE2EWithAIMockBrain(t *testing.T) {
	api := mockAPI()
	defer api.Close()

	// Mock AI brain.
	brain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/health":
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "version": "0.4.0", "ai_mode": "local", "model": "test"})
		case "/api/v1/scenarios/generate":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"scenarios": []map[string]any{
					{
						"name":        "AI: User CRUD flow",
						"description": "Create, read, and delete a user",
						"category":    "happy_path",
						"severity":    "high",
						"request":     map[string]any{"method": "GET", "url": fmt.Sprintf("%s/users", api.URL)},
						"assertions":  []map[string]any{{"type": "status_code", "target": "status_code", "operator": "eq", "expected": 200}},
						"tags":        []string{"ai-generated"},
					},
				},
				"model_used":  "test-model",
				"tokens_used": 150,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer brain.Close()

	orig := changeToTempDir(t)
	t.Cleanup(func() { _ = chdir(orig) })

	srv, err := server.New("127.0.0.1:0", server.WithAIURL(brain.URL))
	if err != nil {
		t.Fatal(err)
	}
	probex := httptest.NewServer(srv.Handler())
	defer probex.Close()

	// Scan.
	scanBody, _ := json.Marshal(map[string]any{"base_url": api.URL, "max_depth": 1})
	resp, err := http.Post(probex.URL+"/api/v1/scan", "application/json", bytes.NewReader(scanBody))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	// Run with AI enabled.
	runBody, _ := json.Marshal(map[string]any{
		"use_ai":  true,
		"timeout": 10,
	})
	resp, err = http.Post(probex.URL+"/api/v1/run", "application/json", bytes.NewReader(runBody))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		t.Fatalf("run with AI failed: %d: %v", resp.StatusCode, errBody)
	}

	var summary models.RunSummary
	_ = json.NewDecoder(resp.Body).Decode(&summary)

	t.Logf("AI-augmented run: %d total tests", summary.TotalTests)

	// Check that AI-generated tests were included.
	hasAITest := false
	for _, r := range summary.Results {
		if r.TestName == "AI: User CRUD flow" {
			hasAITest = true
			break
		}
	}
	if !hasAITest {
		t.Error("expected AI-generated test in results")
	}
}
