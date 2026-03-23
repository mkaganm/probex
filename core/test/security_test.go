package test

import (
	"testing"

	"github.com/mkaganm/probex/internal/generator"
	"github.com/mkaganm/probex/internal/models"
)

func TestSecurityGeneratorBOLA(t *testing.T) {
	ep := models.Endpoint{
		Method:  "GET",
		Path:    "/users/{id}",
		BaseURL: "http://localhost",
		PathParams: []models.Parameter{
			{Name: "id", Type: "string"},
		},
	}

	gen := generator.NewSecurity()
	tests, err := gen.Generate(ep)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	bolaCount := 0
	for _, tc := range tests {
		if hasTag(tc.Tags, "bola") {
			bolaCount++
		}
	}
	if bolaCount == 0 {
		t.Error("Expected BOLA tests for endpoint with path param 'id'")
	}
	if bolaCount != 4 {
		t.Errorf("Expected 4 BOLA tests, got %d", bolaCount)
	}
}

func TestSecurityGeneratorBrokenAuth(t *testing.T) {
	ep := models.Endpoint{
		Method:  "GET",
		Path:    "/users",
		BaseURL: "http://localhost",
	}

	gen := generator.NewSecurity()
	tests, err := gen.Generate(ep)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	authCount := 0
	for _, tc := range tests {
		if hasTag(tc.Tags, "broken_auth") {
			authCount++
		}
	}
	if authCount != 5 {
		t.Errorf("Expected 5 broken auth tests, got %d", authCount)
	}
}

func TestSecurityGeneratorMassAssignment(t *testing.T) {
	ep := models.Endpoint{
		Method:  "POST",
		Path:    "/users",
		BaseURL: "http://localhost",
		RequestBody: &models.Schema{
			Type: "object",
			Properties: map[string]*models.Schema{
				"name":  {Type: "string"},
				"email": {Type: "string"},
			},
		},
	}

	gen := generator.NewSecurity()
	tests, err := gen.Generate(ep)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	maCount := 0
	for _, tc := range tests {
		if hasTag(tc.Tags, "mass_assignment") {
			maCount++
		}
	}
	if maCount != 1 {
		t.Errorf("Expected 1 mass assignment test, got %d", maCount)
	}
}

func TestSecurityGeneratorSSRF(t *testing.T) {
	ep := models.Endpoint{
		Method:  "POST",
		Path:    "/hooks",
		BaseURL: "http://localhost",
		RequestBody: &models.Schema{
			Type: "object",
			Properties: map[string]*models.Schema{
				"name":         {Type: "string"},
				"callback_url": {Type: "string"},
			},
		},
	}

	gen := generator.NewSecurity()
	tests, err := gen.Generate(ep)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	ssrfCount := 0
	for _, tc := range tests {
		if hasTag(tc.Tags, "ssrf") {
			ssrfCount++
		}
	}
	if ssrfCount != 5 {
		t.Errorf("Expected 5 SSRF tests for 'callback_url' field, got %d", ssrfCount)
	}
}

func TestSecurityGeneratorBFLA(t *testing.T) {
	ep := models.Endpoint{
		Method:  "GET",
		Path:    "/admin/users",
		BaseURL: "http://localhost",
	}

	gen := generator.NewSecurity()
	tests, err := gen.Generate(ep)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	bflaCount := 0
	for _, tc := range tests {
		if hasTag(tc.Tags, "bfla") {
			bflaCount++
		}
	}
	if bflaCount != 1 {
		t.Errorf("Expected 1 BFLA test for admin endpoint, got %d", bflaCount)
	}
}

func TestSecurityGeneratorSecurityMisconfig(t *testing.T) {
	ep := models.Endpoint{
		Method:  "POST",
		Path:    "/users",
		BaseURL: "http://localhost",
		RequestBody: &models.Schema{
			Type: "object",
			Properties: map[string]*models.Schema{
				"name": {Type: "string"},
			},
		},
	}

	gen := generator.NewSecurity()
	tests, err := gen.Generate(ep)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	corsCount := 0
	verboseCount := 0
	headerCount := 0
	for _, tc := range tests {
		if hasTag(tc.Tags, "cors") {
			corsCount++
		}
		if hasTag(tc.Tags, "verbose_error") {
			verboseCount++
		}
		if hasTag(tc.Tags, "security_headers") {
			headerCount++
		}
	}
	if corsCount != 1 {
		t.Errorf("Expected 1 CORS test, got %d", corsCount)
	}
	if verboseCount != 1 {
		t.Errorf("Expected 1 verbose error test, got %d", verboseCount)
	}
	if headerCount != 1 {
		t.Errorf("Expected 1 security headers test, got %d", headerCount)
	}
}

func hasTag(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
