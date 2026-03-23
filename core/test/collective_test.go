package test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mkaganm/probex/internal/collective"
	"github.com/mkaganm/probex/internal/models"
)

func TestAnonymizerExtractPatterns(t *testing.T) {
	summary := &models.RunSummary{
		TotalTests: 3,
		Passed:     2,
		Failed:     1,
		Results: []models.TestResult{
			{
				TestCaseID: "tc-1",
				TestName:   "BOLA: access other user's resource",
				Status:     "failed",
				Category:   models.CategorySecurity,
				Severity:   models.SeverityHigh,
			},
			{
				TestCaseID: "tc-2",
				TestName:   "Happy path: GET /users returns 200",
				Status:     "passed",
				Category:   models.CategoryHappyPath,
				Severity:   models.SeverityLow,
			},
			{
				TestCaseID: "tc-3",
				TestName:   "Edge case: empty body",
				Status:     "passed",
				Category:   models.CategoryEdgeCase,
				Severity:   models.SeverityMedium,
			},
		},
	}

	anon := collective.NewAnonymizer()
	patterns := anon.ExtractPatterns(summary)

	if len(patterns) == 0 {
		t.Fatal("expected at least 1 pattern")
	}

	// Failed security test should produce a higher score pattern.
	foundSecurity := false
	for _, p := range patterns {
		if p.Category == "security" {
			foundSecurity = true
			if p.Score < 0.7 {
				t.Errorf("expected high score for failed security pattern, got %.1f", p.Score)
			}
		}
	}
	if !foundSecurity {
		t.Error("expected a security pattern")
	}
}

func TestCollectivePush(t *testing.T) {
	var received collective.Contribution

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/collective/push" && r.Method == "POST" {
			json.NewDecoder(r.Body).Decode(&received)
			w.WriteHeader(http.StatusCreated)
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	client := collective.NewClient(srv.URL, "test-instance-123")
	patterns := []collective.Pattern{
		{ID: "p1", Category: "security", TestType: "bola", Score: 0.9},
	}

	err := client.Push(context.Background(), patterns)
	if err != nil {
		t.Fatalf("push failed: %v", err)
	}

	if received.InstanceID != "test-instance-123" {
		t.Errorf("expected instance ID test-instance-123, got %s", received.InstanceID)
	}
	if len(received.Patterns) != 1 {
		t.Errorf("expected 1 pattern, got %d", len(received.Patterns))
	}
}

func TestCollectivePull(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/collective/pull" {
			resp := collective.PullResponse{
				Patterns: []collective.Pattern{
					{ID: "c1", Category: "security", TestType: "bola", Score: 0.95, UsageCount: 42},
					{ID: "c2", Category: "edge_case", TestType: "boundary", Score: 0.8, UsageCount: 15},
				},
				Total: 2,
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	client := collective.NewClient(srv.URL, "test-instance-123")
	resp, err := client.Pull(context.Background(), nil, 0.5)
	if err != nil {
		t.Fatalf("pull failed: %v", err)
	}

	if len(resp.Patterns) != 2 {
		t.Errorf("expected 2 patterns, got %d", len(resp.Patterns))
	}

	// Convert to test cases.
	tests := collective.PatternToTestCases(resp.Patterns, "http://localhost:8080")
	if len(tests) != 2 {
		t.Errorf("expected 2 test cases, got %d", len(tests))
	}

	for _, tc := range tests {
		found := false
		for _, tag := range tc.Tags {
			if tag == "collective" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("test %q missing 'collective' tag", tc.Name)
		}
	}
}

func TestGenerateInstanceID(t *testing.T) {
	id1 := collective.GenerateInstanceID("host-a")
	id2 := collective.GenerateInstanceID("host-b")
	id3 := collective.GenerateInstanceID("host-a")

	if id1 == id2 {
		t.Error("different seeds should produce different IDs")
	}
	if id1 != id3 {
		t.Error("same seed should produce same ID")
	}
	if len(id1) != 32 {
		t.Errorf("expected 32-char hex ID, got %d chars", len(id1))
	}
}
