package test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mkaganm/probex/internal/models"
	"github.com/mkaganm/probex/internal/runner"
)

func TestNewExecutor(t *testing.T) {
	e := runner.New(models.RunOptions{Concurrency: 5})
	if e == nil {
		t.Fatal("expected non-nil executor")
	}
}

func TestExecuteEmptyTests(t *testing.T) {
	e := runner.New(models.RunOptions{Concurrency: 5})
	summary, err := e.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.TotalTests != 0 {
		t.Errorf("expected 0 total tests, got %d", summary.TotalTests)
	}
}

func TestExecuteStatusCodeAssert(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	tests := []models.TestCase{
		{
			ID:       "t1",
			Name:     "Status 200 check",
			Category: models.CategoryHappyPath,
			Severity: models.SeverityHigh,
			Request: models.TestRequest{
				Method:  "GET",
				URL:     server.URL + "/test",
				Timeout: 5 * time.Second,
			},
			Assertions: []models.Assertion{
				{
					Type:     models.AssertStatusCode,
					Target:   "status_code",
					Operator: "eq",
					Expected: 200,
				},
			},
		},
	}

	e := runner.New(models.RunOptions{Concurrency: 5})
	summary, err := e.Execute(context.Background(), tests)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if summary.Passed != 1 {
		t.Errorf("expected 1 passed, got %d", summary.Passed)
	}
	if summary.Failed != 0 {
		t.Errorf("expected 0 failed, got %d", summary.Failed)
	}
}

func TestExecuteBodyAssert(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name": "Alice",
			"age":  30,
		})
	}))
	defer server.Close()

	tests := []models.TestCase{
		{
			ID:       "t2",
			Name:     "Body JSONPath check",
			Category: models.CategoryHappyPath,
			Severity: models.SeverityHigh,
			Request: models.TestRequest{
				Method: "GET",
				URL:    server.URL + "/user",
			},
			Assertions: []models.Assertion{
				{
					Type:     models.AssertBody,
					Target:   "name",
					Operator: "eq",
					Expected: "Alice",
				},
			},
		},
	}

	e := runner.New(models.RunOptions{Concurrency: 5})
	summary, err := e.Execute(context.Background(), tests)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if summary.Passed != 1 {
		t.Errorf("expected 1 passed, got %d", summary.Passed)
	}
}

func TestExecuteHeaderAssert(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Custom", "test-value")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	tests := []models.TestCase{
		{
			ID:   "t3",
			Name: "Header check",
			Request: models.TestRequest{
				Method: "GET",
				URL:    server.URL + "/test",
			},
			Assertions: []models.Assertion{
				{
					Type:     models.AssertHeader,
					Target:   "X-Custom",
					Operator: "eq",
					Expected: "test-value",
				},
			},
		},
	}

	e := runner.New(models.RunOptions{Concurrency: 5})
	summary, err := e.Execute(context.Background(), tests)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if summary.Passed != 1 {
		t.Errorf("expected 1 passed, got %d", summary.Passed)
	}
}

func TestExecuteConcurrent(t *testing.T) {
	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	var tests []models.TestCase
	for i := 0; i < 10; i++ {
		tests = append(tests, models.TestCase{
			ID:   "concurrent",
			Name: "Concurrent test",
			Request: models.TestRequest{
				Method: "GET",
				URL:    server.URL,
			},
			Assertions: []models.Assertion{
				{Type: models.AssertStatusCode, Operator: "eq", Expected: 200},
			},
		})
	}

	e := runner.New(models.RunOptions{Concurrency: 5})
	summary, err := e.Execute(context.Background(), tests)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if summary.TotalTests != 10 {
		t.Errorf("expected 10 tests, got %d", summary.TotalTests)
	}
	if summary.Passed != 10 {
		t.Errorf("expected 10 passed, got %d", summary.Passed)
	}
}

func TestVarContext(t *testing.T) {
	vc := runner.NewVarContext()
	vc.Set("user_id", "123")
	val, ok := vc.Get("user_id")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if val != "123" {
		t.Errorf("expected '123', got %v", val)
	}

	_, ok = vc.Get("nonexistent")
	if ok {
		t.Error("expected key to not exist")
	}
}
