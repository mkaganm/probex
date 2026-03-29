package test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mkaganm/probex/internal/models"
	"github.com/mkaganm/probex/internal/runner"
)

// ---------------------------------------------------------------------------
// AssertionEngine.Evaluate — status_code assertions
// ---------------------------------------------------------------------------

func TestRunnerAssertionEngine_StatusCode(t *testing.T) {
	ae := runner.NewAssertionEngine()

	tests := []struct {
		name     string
		op       string
		expected any
		actual   int
		want     bool
	}{
		{"eq pass", "eq", 200, 200, true},
		{"eq fail", "eq", 200, 404, false},
		{"ne pass", "ne", 200, 404, true},
		{"ne fail", "ne", 200, 200, false},
		{"gt pass", "gt", 200, 201, true},
		{"gt fail equal", "gt", 200, 200, false},
		{"gt fail less", "gt", 200, 199, false},
		{"gte pass equal", "gte", 200, 200, true},
		{"gte pass greater", "gte", 200, 201, true},
		{"gte fail", "gte", 200, 199, false},
		{"lt pass", "lt", 400, 200, true},
		{"lt fail equal", "lt", 200, 200, false},
		{"lt fail greater", "lt", 200, 201, false},
		{"lte pass equal", "lte", 200, 200, true},
		{"lte pass less", "lte", 200, 199, true},
		{"lte fail", "lte", 200, 201, false},
		{"float64 expected", "eq", float64(200), 200, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := models.Assertion{
				Type:     models.AssertStatusCode,
				Target:   "status_code",
				Operator: tt.op,
				Expected: tt.expected,
			}
			resp := &models.TestResponse{StatusCode: tt.actual}
			result := ae.Evaluate(a, resp)
			if result.Passed != tt.want {
				t.Errorf("Evaluate() passed=%v, want %v (op=%s expected=%v actual=%d)",
					result.Passed, tt.want, tt.op, tt.expected, tt.actual)
			}
			if !result.Passed && result.Message == "" {
				t.Error("failed assertion should include a message")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// AssertionEngine.Evaluate — body assertions
// ---------------------------------------------------------------------------

func TestRunnerAssertionEngine_Body_JSONPath(t *testing.T) {
	ae := runner.NewAssertionEngine()

	jsonBody := `{"name":"Alice","age":30,"nested":{"city":"NYC"},"tags":["go","rust"]}`

	tests := []struct {
		name     string
		target   string
		op       string
		expected any
		want     bool
	}{
		{"string eq pass", "name", "eq", "Alice", true},
		{"string eq fail", "name", "eq", "Bob", false},
		{"string ne pass", "name", "ne", "Bob", true},
		{"string ne fail", "name", "ne", "Alice", false},
		{"number eq pass", "age", "eq", float64(30), true},
		{"number eq fail", "age", "eq", float64(25), false},
		{"number gt pass", "age", "gt", float64(20), true},
		{"number gt fail", "age", "gt", float64(40), false},
		{"number lt pass", "age", "lt", float64(40), true},
		{"number lt fail", "age", "lt", float64(20), false},
		{"nested path eq", "nested.city", "eq", "NYC", true},
		{"nested path ne", "nested.city", "ne", "NYC", false},
		{"array element", "tags.0", "eq", "go", true},
		{"array element 1", "tags.1", "eq", "rust", true},
		{"contains in string", "name", "contains", "lic", true},
		{"not_contains pass", "name", "not_contains", "xyz", true},
		{"not_contains fail", "name", "not_contains", "Ali", false},
		{"path not found", "nonexistent", "eq", "x", false},
		{"path not found not_exists", "nonexistent", "not_exists", nil, true},
		{"exists operator", "name", "exists", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := models.Assertion{
				Type:     models.AssertBody,
				Target:   tt.target,
				Operator: tt.op,
				Expected: tt.expected,
			}
			resp := &models.TestResponse{StatusCode: 200, Body: jsonBody}
			result := ae.Evaluate(a, resp)
			if result.Passed != tt.want {
				t.Errorf("Evaluate() passed=%v, want %v (target=%s op=%s expected=%v)",
					result.Passed, tt.want, tt.target, tt.op, tt.expected)
			}
		})
	}
}

func TestRunnerAssertionEngine_Body_ValidJSON(t *testing.T) {
	ae := runner.NewAssertionEngine()

	tests := []struct {
		name     string
		body     string
		expected any
		want     bool
	}{
		{"valid json true", `{"a":1}`, true, true},
		{"valid json array", `[1,2,3]`, true, true},
		{"invalid json", `{broken`, true, false},
		{"empty body valid false", ``, true, false},
		{"expect false on valid json", `{"a":1}`, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := models.Assertion{
				Type:     models.AssertBody,
				Target:   "@valid",
				Operator: "eq",
				Expected: tt.expected,
			}
			resp := &models.TestResponse{StatusCode: 200, Body: tt.body}
			result := ae.Evaluate(a, resp)
			if result.Passed != tt.want {
				t.Errorf("Evaluate() passed=%v, want %v (body=%q expected=%v)",
					result.Passed, tt.want, tt.body, tt.expected)
			}
		})
	}
}

func TestRunnerAssertionEngine_Body_Raw(t *testing.T) {
	ae := runner.NewAssertionEngine()

	body := "Hello World! This is raw text."

	tests := []struct {
		name     string
		op       string
		expected any
		want     bool
	}{
		{"contains pass", "contains", "World", true},
		{"contains fail", "contains", "missing", false},
		{"not_contains pass", "not_contains", "missing", true},
		{"not_contains fail", "not_contains", "World", false},
		{"eq pass", "eq", body, true},
		{"eq fail", "eq", "wrong", false},
		{"unknown op fails", "gt", "x", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := models.Assertion{
				Type:     models.AssertBody,
				Target:   "@raw",
				Operator: tt.op,
				Expected: tt.expected,
			}
			resp := &models.TestResponse{StatusCode: 200, Body: body}
			result := ae.Evaluate(a, resp)
			if result.Passed != tt.want {
				t.Errorf("Evaluate() passed=%v, want %v (op=%s expected=%v)",
					result.Passed, tt.want, tt.op, tt.expected)
			}
		})
	}
}

func TestRunnerAssertionEngine_Body_EmptyBody(t *testing.T) {
	ae := runner.NewAssertionEngine()

	a := models.Assertion{
		Type:     models.AssertBody,
		Target:   "name",
		Operator: "eq",
		Expected: "Alice",
	}
	resp := &models.TestResponse{StatusCode: 200, Body: ""}
	result := ae.Evaluate(a, resp)
	if result.Passed {
		t.Error("expected assertion to fail on empty body")
	}
}

// ---------------------------------------------------------------------------
// AssertionEngine.Evaluate — header assertions
// ---------------------------------------------------------------------------

func TestRunnerAssertionEngine_Header(t *testing.T) {
	ae := runner.NewAssertionEngine()

	headers := map[string]string{
		"Content-Type":    "application/json",
		"X-Custom-Header": "test-value",
		"X-Request-Id":    "abc-123",
	}

	tests := []struct {
		name     string
		target   string
		op       string
		expected any
		want     bool
	}{
		{"eq pass", "Content-Type", "eq", "application/json", true},
		{"eq fail", "Content-Type", "eq", "text/html", false},
		{"ne pass", "Content-Type", "ne", "text/html", true},
		{"ne fail", "Content-Type", "ne", "application/json", false},
		{"contains pass", "Content-Type", "contains", "json", true},
		{"contains fail", "Content-Type", "contains", "xml", false},
		{"not_contains pass", "Content-Type", "not_contains", "xml", true},
		{"not_contains fail", "Content-Type", "not_contains", "json", false},
		{"exists pass", "X-Custom-Header", "exists", "", true},
		{"not_exists pass on missing", "X-Missing", "not_exists", "", true},
		{"missing header eq fails", "X-Missing", "eq", "val", false},
		// case-insensitive lookup
		{"case insensitive eq", "content-type", "eq", "application/json", true},
		{"case insensitive custom", "x-custom-header", "eq", "test-value", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := models.Assertion{
				Type:     models.AssertHeader,
				Target:   tt.target,
				Operator: tt.op,
				Expected: tt.expected,
			}
			resp := &models.TestResponse{StatusCode: 200, Headers: headers}
			result := ae.Evaluate(a, resp)
			if result.Passed != tt.want {
				t.Errorf("Evaluate() passed=%v, want %v (target=%s op=%s expected=%v)",
					result.Passed, tt.want, tt.target, tt.op, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// AssertionEngine.Evaluate — response_time assertions
// ---------------------------------------------------------------------------

func TestRunnerAssertionEngine_ResponseTime(t *testing.T) {
	ae := runner.NewAssertionEngine()

	tests := []struct {
		name     string
		op       string
		expected any
		duration time.Duration
		want     bool
	}{
		{"lt pass", "lt", float64(2 * time.Second), 1 * time.Second, true},
		{"lt fail", "lt", float64(500 * time.Millisecond), 1 * time.Second, false},
		{"gt pass", "gt", float64(100 * time.Millisecond), 1 * time.Second, true},
		{"gt fail", "gt", float64(2 * time.Second), 1 * time.Second, false},
		{"lte pass equal", "lte", float64(1 * time.Second), 1 * time.Second, true},
		{"gte pass equal", "gte", float64(1 * time.Second), 1 * time.Second, true},
		{"eq pass", "eq", float64(1 * time.Second), 1 * time.Second, true},
		{"eq fail", "eq", float64(2 * time.Second), 1 * time.Second, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := models.Assertion{
				Type:     models.AssertResponseTime,
				Operator: tt.op,
				Expected: tt.expected,
			}
			resp := &models.TestResponse{StatusCode: 200, Duration: tt.duration}
			result := ae.Evaluate(a, resp)
			if result.Passed != tt.want {
				t.Errorf("Evaluate() passed=%v, want %v (op=%s expected=%v duration=%v)",
					result.Passed, tt.want, tt.op, tt.expected, tt.duration)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// AssertionEngine.Evaluate — schema assertions
// ---------------------------------------------------------------------------

func TestRunnerAssertionEngine_Schema(t *testing.T) {
	ae := runner.NewAssertionEngine()

	t.Run("object with matching properties", func(t *testing.T) {
		body := `{"name":"Alice","age":30}`
		schema := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{"type": "string"},
				"age":  map[string]any{"type": "integer"},
			},
		}
		a := models.Assertion{
			Type:     models.AssertSchema,
			Operator: "eq",
			Expected: schema,
		}
		resp := &models.TestResponse{StatusCode: 200, Body: body}
		result := ae.Evaluate(a, resp)
		if !result.Passed {
			t.Errorf("expected schema validation to pass, msg=%s", result.Message)
		}
	})

	t.Run("type mismatch fails", func(t *testing.T) {
		body := `"just a string"`
		schema := map[string]any{
			"type": "object",
		}
		a := models.Assertion{
			Type:     models.AssertSchema,
			Operator: "eq",
			Expected: schema,
		}
		resp := &models.TestResponse{StatusCode: 200, Body: body}
		result := ae.Evaluate(a, resp)
		if result.Passed {
			t.Error("expected schema validation to fail for type mismatch")
		}
	})

	t.Run("array type passes", func(t *testing.T) {
		body := `[1,2,3]`
		schema := map[string]any{
			"type": "array",
		}
		a := models.Assertion{
			Type:     models.AssertSchema,
			Operator: "eq",
			Expected: schema,
		}
		resp := &models.TestResponse{StatusCode: 200, Body: body}
		result := ae.Evaluate(a, resp)
		if !result.Passed {
			t.Errorf("expected schema validation to pass, msg=%s", result.Message)
		}
	})

	t.Run("integer compatible with number", func(t *testing.T) {
		body := `42`
		schema := map[string]any{
			"type": "number",
		}
		a := models.Assertion{
			Type:     models.AssertSchema,
			Operator: "eq",
			Expected: schema,
		}
		resp := &models.TestResponse{StatusCode: 200, Body: body}
		result := ae.Evaluate(a, resp)
		if !result.Passed {
			t.Errorf("expected integer to be compatible with number, msg=%s", result.Message)
		}
	})

	t.Run("nested property type mismatch", func(t *testing.T) {
		body := `{"name":123}`
		schema := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{"type": "string"},
			},
		}
		a := models.Assertion{
			Type:     models.AssertSchema,
			Operator: "eq",
			Expected: schema,
		}
		resp := &models.TestResponse{StatusCode: 200, Body: body}
		result := ae.Evaluate(a, resp)
		if result.Passed {
			t.Error("expected schema validation to fail for nested type mismatch")
		}
	})

	t.Run("invalid JSON body", func(t *testing.T) {
		a := models.Assertion{
			Type:     models.AssertSchema,
			Operator: "eq",
			Expected: map[string]any{"type": "object"},
		}
		resp := &models.TestResponse{StatusCode: 200, Body: `{broken`}
		result := ae.Evaluate(a, resp)
		if result.Passed {
			t.Error("expected schema validation to fail for invalid JSON")
		}
	})

	t.Run("non-map expected skips gracefully", func(t *testing.T) {
		a := models.Assertion{
			Type:     models.AssertSchema,
			Operator: "eq",
			Expected: "not a map",
		}
		resp := &models.TestResponse{StatusCode: 200, Body: `{"a":1}`}
		result := ae.Evaluate(a, resp)
		// schema assertion is skipped (passed=true) when expected is not a map
		if !result.Passed {
			t.Errorf("expected schema to be skipped when expected is not a map, msg=%s", result.Message)
		}
	})
}

// ---------------------------------------------------------------------------
// AssertionEngine.Evaluate — edge cases
// ---------------------------------------------------------------------------

func TestRunnerAssertionEngine_NilResponse(t *testing.T) {
	ae := runner.NewAssertionEngine()

	types := []models.AssertionType{
		models.AssertStatusCode,
		models.AssertBody,
		models.AssertHeader,
		models.AssertResponseTime,
		models.AssertSchema,
	}

	for _, at := range types {
		t.Run(string(at), func(t *testing.T) {
			a := models.Assertion{
				Type:     at,
				Operator: "eq",
				Expected: 200,
			}
			result := ae.Evaluate(a, nil)
			if result.Passed {
				t.Error("expected assertion to fail with nil response")
			}
			if result.Message != "no response received" {
				t.Errorf("expected 'no response received' message, got %q", result.Message)
			}
		})
	}
}

func TestRunnerAssertionEngine_UnknownType(t *testing.T) {
	ae := runner.NewAssertionEngine()

	a := models.Assertion{
		Type:     models.AssertionType("unknown_type"),
		Operator: "eq",
		Expected: 200,
	}
	resp := &models.TestResponse{StatusCode: 200}
	result := ae.Evaluate(a, resp)
	if result.Passed {
		t.Error("expected assertion to fail for unknown type")
	}
	if result.Message == "" {
		t.Error("expected a message for unknown assertion type")
	}
}

// ---------------------------------------------------------------------------
// VarContext — basic operations
// ---------------------------------------------------------------------------

func TestRunnerVarContext_SetGet(t *testing.T) {
	vc := runner.NewVarContext()

	tests := []struct {
		key   string
		value any
	}{
		{"string_var", "hello"},
		{"int_var", 42},
		{"float_var", 3.14},
		{"bool_var", true},
		{"nil_var", nil},
		{"map_var", map[string]string{"a": "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			vc.Set(tt.key, tt.value)
			got, ok := vc.Get(tt.key)
			if !ok {
				t.Fatalf("Get(%q) returned ok=false, want true", tt.key)
			}
			// nil comparison needs special handling
			if tt.value == nil {
				if got != nil {
					t.Errorf("Get(%q) = %v, want nil", tt.key, got)
				}
				return
			}
			gotStr, wantStr := stringify(got), stringify(tt.value)
			if gotStr != wantStr {
				t.Errorf("Get(%q) = %v, want %v", tt.key, got, tt.value)
			}
		})
	}
}

func TestRunnerVarContext_GetNonExistent(t *testing.T) {
	vc := runner.NewVarContext()

	val, ok := vc.Get("does_not_exist")
	if ok {
		t.Error("Get for non-existent key should return ok=false")
	}
	if val != nil {
		t.Errorf("Get for non-existent key should return nil, got %v", val)
	}
}

func TestRunnerVarContext_Overwrite(t *testing.T) {
	vc := runner.NewVarContext()

	vc.Set("key", "original")
	vc.Set("key", "updated")

	val, ok := vc.Get("key")
	if !ok {
		t.Fatal("Get returned ok=false after overwrite")
	}
	if val != "updated" {
		t.Errorf("Get after overwrite = %v, want 'updated'", val)
	}
}

// ---------------------------------------------------------------------------
// VarContext — concurrent safety
// ---------------------------------------------------------------------------

func TestRunnerVarContext_ConcurrentAccess(t *testing.T) {
	vc := runner.NewVarContext()

	const goroutines = 100
	const iterations = 50
	var wg sync.WaitGroup

	// Concurrent writers
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				key := stringify(id)
				vc.Set(key, i)
			}
		}(g)
	}

	// Concurrent readers
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				key := stringify(id)
				vc.Get(key)
			}
		}(g)
	}

	wg.Wait()

	// Verify all goroutine keys exist with their final value
	for g := 0; g < goroutines; g++ {
		key := stringify(g)
		val, ok := vc.Get(key)
		if !ok {
			t.Errorf("Get(%q) returned ok=false after concurrent writes", key)
			continue
		}
		if val != iterations-1 {
			t.Errorf("Get(%q) = %v, want %d", key, val, iterations-1)
		}
	}
}

// ---------------------------------------------------------------------------
// Executor — integration tests using httptest (no real network)
// ---------------------------------------------------------------------------

func TestRunnerNewExecutor(t *testing.T) {
	e := runner.New(models.RunOptions{Concurrency: 5})
	if e == nil {
		t.Fatal("expected non-nil executor")
	}
}

func TestRunnerExecuteEmptyTests(t *testing.T) {
	e := runner.New(models.RunOptions{Concurrency: 5})
	summary, err := e.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.TotalTests != 0 {
		t.Errorf("expected 0 total tests, got %d", summary.TotalTests)
	}
}

func TestRunnerExecuteStatusCodeAssert(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

func TestRunnerExecuteBodyAssert(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

func TestRunnerExecuteHeaderAssert(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

func TestRunnerExecuteConcurrent(t *testing.T) {
	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

func TestRunnerExecuteMultipleAssertions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-Id", "req-001")
		w.WriteHeader(200)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
			"count":  5,
		})
	}))
	defer server.Close()

	tests := []models.TestCase{
		{
			ID:   "multi",
			Name: "Multiple assertions",
			Request: models.TestRequest{
				Method: "GET",
				URL:    server.URL,
			},
			Assertions: []models.Assertion{
				{Type: models.AssertStatusCode, Operator: "eq", Expected: 200},
				{Type: models.AssertBody, Target: "status", Operator: "eq", Expected: "ok"},
				{Type: models.AssertHeader, Target: "Content-Type", Operator: "contains", Expected: "json"},
			},
		},
	}

	e := runner.New(models.RunOptions{Concurrency: 1})
	summary, err := e.Execute(context.Background(), tests)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.Passed != 1 {
		t.Errorf("expected 1 passed, got %d (failed=%d errors=%d)", summary.Passed, summary.Failed, summary.Errors)
	}
	if len(summary.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(summary.Results))
	}
	if len(summary.Results[0].Assertions) != 3 {
		t.Errorf("expected 3 assertion results, got %d", len(summary.Results[0].Assertions))
	}
}

func TestRunnerExecuteFailedAssertion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"error":"not found"}`))
	}))
	defer server.Close()

	tests := []models.TestCase{
		{
			ID:   "fail",
			Name: "Expected to fail",
			Request: models.TestRequest{
				Method: "GET",
				URL:    server.URL,
			},
			Assertions: []models.Assertion{
				{Type: models.AssertStatusCode, Operator: "eq", Expected: 200},
			},
		},
	}

	e := runner.New(models.RunOptions{Concurrency: 1})
	summary, err := e.Execute(context.Background(), tests)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", summary.Failed)
	}
	if summary.Passed != 0 {
		t.Errorf("expected 0 passed, got %d", summary.Passed)
	}
}

func TestRunnerExecuteSummaryCounters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	tests := []models.TestCase{
		{
			ID: "pass1", Name: "pass", Category: models.CategoryHappyPath, Severity: models.SeverityHigh,
			Request:    models.TestRequest{Method: "GET", URL: server.URL},
			Assertions: []models.Assertion{{Type: models.AssertStatusCode, Operator: "eq", Expected: 200}},
		},
		{
			ID: "pass2", Name: "pass2", Category: models.CategoryEdgeCase, Severity: models.SeverityMedium,
			Request:    models.TestRequest{Method: "GET", URL: server.URL},
			Assertions: []models.Assertion{{Type: models.AssertStatusCode, Operator: "eq", Expected: 200}},
		},
		{
			ID: "fail1", Name: "fail", Category: models.CategorySecurity, Severity: models.SeverityCritical,
			Request:    models.TestRequest{Method: "GET", URL: server.URL},
			Assertions: []models.Assertion{{Type: models.AssertStatusCode, Operator: "eq", Expected: 404}},
		},
	}

	e := runner.New(models.RunOptions{Concurrency: 1})
	summary, err := e.Execute(context.Background(), tests)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.TotalTests != 3 {
		t.Errorf("TotalTests: got %d, want 3", summary.TotalTests)
	}
	if summary.Passed != 2 {
		t.Errorf("Passed: got %d, want 2", summary.Passed)
	}
	if summary.Failed != 1 {
		t.Errorf("Failed: got %d, want 1", summary.Failed)
	}
	if summary.Duration <= 0 {
		t.Error("Duration should be positive")
	}
	if summary.BySeverity == nil {
		t.Fatal("BySeverity should not be nil")
	}
	if summary.ByCategory == nil {
		t.Fatal("ByCategory should not be nil")
	}
}

// ---------------------------------------------------------------------------
// VarContext — original integration test preserved
// ---------------------------------------------------------------------------

func TestRunnerVarContext(t *testing.T) {
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

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func stringify(v any) string {
	switch val := v.(type) {
	case string:
		return val
	default:
		b, _ := json.Marshal(val)
		return string(b)
	}
}
