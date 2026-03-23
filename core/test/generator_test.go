package test

import (
	"testing"

	"github.com/mkaganm/probex/internal/generator"
	"github.com/mkaganm/probex/internal/models"
)

func TestNewEngine(t *testing.T) {
	profile := &models.APIProfile{
		BaseURL: "https://api.example.com",
	}
	e := generator.New(profile)
	if e == nil {
		t.Fatal("expected non-nil engine")
	}
}

func TestGenerateEmptyProfile(t *testing.T) {
	profile := &models.APIProfile{}
	e := generator.New(profile)
	tests, err := e.Generate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tests) != 0 {
		t.Errorf("expected 0 tests for empty profile, got %d", len(tests))
	}
}

func TestGenerateHappyPathTests(t *testing.T) {
	profile := &models.APIProfile{
		BaseURL: "https://api.example.com",
		Endpoints: []models.Endpoint{
			{
				ID:      "test1",
				Method:  "GET",
				Path:    "/users",
				BaseURL: "https://api.example.com",
				Responses: []models.Response{
					{StatusCode: 200, ContentType: "application/json"},
				},
			},
		},
	}

	e := generator.New(profile)
	filter := map[models.TestCategory]bool{models.CategoryHappyPath: true}
	e.SetCategoryFilter(filter)

	tests, err := e.Generate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tests) == 0 {
		t.Fatal("expected at least 1 happy path test")
	}

	for _, tc := range tests {
		if tc.Category != models.CategoryHappyPath {
			t.Errorf("expected category happy_path, got %s", tc.Category)
		}
		if tc.ID == "" {
			t.Error("test case should have an ID")
		}
		if tc.GeneratedBy == "" {
			t.Error("test case should have GeneratedBy set")
		}
	}
}

func TestGenerateEdgeCaseTests(t *testing.T) {
	profile := &models.APIProfile{
		BaseURL: "https://api.example.com",
		Endpoints: []models.Endpoint{
			{
				ID:      "test2",
				Method:  "POST",
				Path:    "/users",
				BaseURL: "https://api.example.com",
				RequestBody: &models.Schema{
					Type: "object",
					Properties: map[string]*models.Schema{
						"name":  {Type: "string"},
						"email": {Type: "string"},
					},
					Required: []string{"name", "email"},
				},
			},
		},
	}

	e := generator.New(profile)
	filter := map[models.TestCategory]bool{models.CategoryEdgeCase: true}
	e.SetCategoryFilter(filter)

	tests, err := e.Generate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should include: empty body, missing name, missing email, wrong types,
	// boundaries, null values, extra fields
	if len(tests) < 5 {
		t.Errorf("expected at least 5 edge case tests, got %d", len(tests))
	}
}

func TestGenerateSecurityTests(t *testing.T) {
	profile := &models.APIProfile{
		BaseURL: "https://api.example.com",
		Endpoints: []models.Endpoint{
			{
				ID:      "test3",
				Method:  "POST",
				Path:    "/search",
				BaseURL: "https://api.example.com",
				QueryParams: []models.Parameter{
					{Name: "q", Type: "string"},
				},
				RequestBody: &models.Schema{
					Type: "object",
					Properties: map[string]*models.Schema{
						"query": {Type: "string"},
					},
				},
				Auth: &models.AuthInfo{
					Type:     models.AuthBearer,
					Location: "header",
					Key:      "Authorization",
				},
			},
		},
	}

	e := generator.New(profile)
	filter := map[models.TestCategory]bool{models.CategorySecurity: true}
	e.SetCategoryFilter(filter)

	tests, err := e.Generate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should include SQLi, XSS, missing auth, large payload
	if len(tests) < 10 {
		t.Errorf("expected at least 10 security tests, got %d", len(tests))
	}

	// Check we have different types
	hasSQLi := false
	hasXSS := false
	hasMissingAuth := false
	for _, tc := range tests {
		if contains(tc.Name, "SQLi") {
			hasSQLi = true
		}
		if contains(tc.Name, "XSS") {
			hasXSS = true
		}
		if contains(tc.Name, "MissingAuth") {
			hasMissingAuth = true
		}
	}

	if !hasSQLi {
		t.Error("expected SQL injection tests")
	}
	if !hasXSS {
		t.Error("expected XSS tests")
	}
	if !hasMissingAuth {
		t.Error("expected missing auth test")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
