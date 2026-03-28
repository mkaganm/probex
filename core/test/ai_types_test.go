package test

import (
	"testing"

	"github.com/mkaganm/probex/internal/ai"
	"github.com/mkaganm/probex/internal/models"
)

func TestEndpointToInfo(t *testing.T) {
	ep := models.Endpoint{
		Method:  "POST",
		Path:    "/users",
		BaseURL: "http://example.com",
		QueryParams: []models.Parameter{
			{Name: "page", Type: "integer", Required: false, Example: 1},
		},
		PathParams: []models.Parameter{
			{Name: "id", Type: "string", Required: true},
		},
		RequestBody: &models.Schema{
			Type: "object",
			Properties: map[string]*models.Schema{
				"name":  {Type: "string"},
				"email": {Type: "string", Format: "email"},
			},
			Required: []string{"name", "email"},
		},
		Auth: &models.AuthInfo{
			Type:     models.AuthBearer,
			Location: "header",
			Key:      "Authorization",
		},
		Tags: []string{"users"},
	}

	info := ai.EndpointToInfo(ep)

	if info.Method != "POST" {
		t.Errorf("Method: got %s, want POST", info.Method)
	}
	if info.Path != "/users" {
		t.Errorf("Path: got %s, want /users", info.Path)
	}
	if info.BaseURL != "http://example.com" {
		t.Errorf("BaseURL: got %s, want http://example.com", info.BaseURL)
	}
	if len(info.QueryParams) != 1 {
		t.Fatalf("QueryParams: got %d, want 1", len(info.QueryParams))
	}
	if info.QueryParams[0].Name != "page" {
		t.Errorf("QueryParam name: got %s, want page", info.QueryParams[0].Name)
	}
	if len(info.PathParams) != 1 {
		t.Fatalf("PathParams: got %d, want 1", len(info.PathParams))
	}
	if !info.PathParams[0].Required {
		t.Error("PathParam should be required")
	}
	if info.RequestBody == nil {
		t.Fatal("expected non-nil RequestBody")
	}
	if info.RequestBody.Type != "object" {
		t.Errorf("RequestBody.Type: got %s, want object", info.RequestBody.Type)
	}
	if len(info.RequestBody.Properties) != 2 {
		t.Errorf("RequestBody.Properties: got %d, want 2", len(info.RequestBody.Properties))
	}
	if info.Auth == nil {
		t.Fatal("expected non-nil Auth")
	}
	if info.Auth.Type != "bearer" {
		t.Errorf("Auth.Type: got %s, want bearer", info.Auth.Type)
	}
	if len(info.Tags) != 1 || info.Tags[0] != "users" {
		t.Errorf("Tags: got %v, want [users]", info.Tags)
	}
}

func TestEndpointToInfoMinimal(t *testing.T) {
	ep := models.Endpoint{
		Method:  "GET",
		Path:    "/health",
		BaseURL: "http://localhost",
	}

	info := ai.EndpointToInfo(ep)

	if info.RequestBody != nil {
		t.Error("expected nil RequestBody for minimal endpoint")
	}
	if info.Auth != nil {
		t.Error("expected nil Auth for minimal endpoint")
	}
	if len(info.QueryParams) != 0 {
		t.Errorf("expected 0 QueryParams, got %d", len(info.QueryParams))
	}
}

func TestEndpointsToInfo(t *testing.T) {
	eps := []models.Endpoint{
		{Method: "GET", Path: "/a", BaseURL: "http://example.com"},
		{Method: "POST", Path: "/b", BaseURL: "http://example.com"},
		{Method: "DELETE", Path: "/c", BaseURL: "http://example.com"},
	}

	infos := ai.EndpointsToInfo(eps)
	if len(infos) != 3 {
		t.Fatalf("expected 3 infos, got %d", len(infos))
	}
	if infos[2].Method != "DELETE" {
		t.Errorf("expected DELETE, got %s", infos[2].Method)
	}
}

func TestGeneratedTestToModelTest(t *testing.T) {
	gen := ai.GeneratedTestCase{
		Name:        "Login Test",
		Description: "Verify login flow",
		Category:    "security",
		Severity:    "high",
		Request: ai.TestRequestInfo{
			Method:  "POST",
			URL:     "http://example.com/login",
			Headers: map[string]string{"Content-Type": "application/json"},
			Body:    `{"username":"admin","password":"test"}`,
		},
		Assertions: []ai.AssertionInfo{
			{Type: "status_code", Target: "status_code", Operator: "eq", Expected: 200},
			{Type: "body_contains", Target: "body", Operator: "contains", Expected: "token"},
		},
		Tags: []string{"auth", "ai-generated"},
	}

	tc := ai.GeneratedTestToModelTest(gen, "ep-login")

	if tc.Name != "Login Test" {
		t.Errorf("Name: got %s, want Login Test", tc.Name)
	}
	if tc.Category != models.TestCategory("security") {
		t.Errorf("Category: got %s, want security", tc.Category)
	}
	if tc.Severity != models.SeverityHigh {
		t.Errorf("Severity: got %s, want high", tc.Severity)
	}
	if tc.EndpointID != "ep-login" {
		t.Errorf("EndpointID: got %s, want ep-login", tc.EndpointID)
	}
	if tc.GeneratedBy != "ai-brain" {
		t.Errorf("GeneratedBy: got %s, want ai-brain", tc.GeneratedBy)
	}
	if tc.Request.Method != "POST" {
		t.Errorf("Request.Method: got %s, want POST", tc.Request.Method)
	}
	if len(tc.Assertions) != 2 {
		t.Fatalf("Assertions: got %d, want 2", len(tc.Assertions))
	}
	if tc.Assertions[0].Type != models.AssertStatusCode {
		t.Errorf("Assertion[0].Type: got %s, want status_code", tc.Assertions[0].Type)
	}
	if len(tc.Tags) != 2 {
		t.Errorf("Tags: got %d, want 2", len(tc.Tags))
	}
}

func TestGeneratedTestsToModelTests(t *testing.T) {
	generated := []ai.GeneratedTestCase{
		{Name: "Test 1", Request: ai.TestRequestInfo{Method: "GET", URL: "/a"}},
		{Name: "Test 2", Request: ai.TestRequestInfo{Method: "POST", URL: "/b"}},
	}

	tests := ai.GeneratedTestsToModelTests(generated)
	if len(tests) != 2 {
		t.Fatalf("expected 2 tests, got %d", len(tests))
	}
	if tests[0].Name != "Test 1" {
		t.Errorf("first test name: got %s, want Test 1", tests[0].Name)
	}
	if tests[1].GeneratedBy != "ai-brain" {
		t.Errorf("GeneratedBy should be ai-brain")
	}
}

func TestNewClient(t *testing.T) {
	c := ai.NewClient("http://localhost:9711")
	if c == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNewBridge(t *testing.T) {
	b := ai.NewBridge(0)
	if b == nil {
		t.Fatal("expected non-nil bridge")
	}
	if b.Address() != "http://127.0.0.1:9711" {
		t.Errorf("Address: got %s, want http://127.0.0.1:9711", b.Address())
	}
	if b.IsReady() {
		t.Error("expected IsReady=false before Start")
	}
}

func TestNewBridgeCustomPort(t *testing.T) {
	b := ai.NewBridge(8888)
	if b.Address() != "http://127.0.0.1:8888" {
		t.Errorf("Address: got %s, want http://127.0.0.1:8888", b.Address())
	}
}
