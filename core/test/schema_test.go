package test

import (
	"testing"

	"github.com/mkaganm/probex/internal/models"
	"github.com/mkaganm/probex/internal/schema"
)

func TestInferFromJSON(t *testing.T) {
	inf := schema.New()

	tests := []struct {
		name     string
		json     string
		wantType string
	}{
		{"object", `{"name":"Alice","age":30}`, "object"},
		{"array", `[1, 2, 3]`, "array"},
		{"string", `"hello"`, "string"},
		{"number", `3.14`, "number"},
		{"integer", `42`, "integer"},
		{"boolean", `true`, "boolean"},
		{"null", `null`, "null"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := inf.InferFromJSON([]byte(tt.json))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if s.Type != tt.wantType {
				t.Errorf("expected type %s, got %s", tt.wantType, s.Type)
			}
		})
	}
}

func TestInferObjectProperties(t *testing.T) {
	inf := schema.New()
	s, err := inf.InferFromJSON([]byte(`{
		"id": 1,
		"name": "Alice",
		"email": "alice@example.com",
		"created": "2024-01-15T10:30:00Z"
	}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if s.Type != "object" {
		t.Fatalf("expected object type, got %s", s.Type)
	}
	if len(s.Properties) != 4 {
		t.Errorf("expected 4 properties, got %d", len(s.Properties))
	}
	if s.Properties["id"].Type != "integer" {
		t.Errorf("expected id to be integer, got %s", s.Properties["id"].Type)
	}
	if s.Properties["email"].Format != "email" {
		t.Errorf("expected email format, got %s", s.Properties["email"].Format)
	}
	if s.Properties["created"].Format != "date-time" {
		t.Errorf("expected date-time format, got %s", s.Properties["created"].Format)
	}
}

func TestMergeSchemas(t *testing.T) {
	inf := schema.New()

	s1, _ := inf.InferFromJSON([]byte(`{"name":"Alice","age":30}`))
	s2, _ := inf.InferFromJSON([]byte(`{"name":"Bob","email":"bob@test.com"}`))

	merged := inf.Merge([]*models.Schema{s1, s2})
	if merged == nil {
		t.Fatal("expected non-nil merged schema")
	}
	if merged.Type != "object" {
		t.Errorf("expected object type, got %s", merged.Type)
	}
	// Should have all three properties (name, age, email)
	if len(merged.Properties) != 3 {
		t.Errorf("expected 3 properties in merged schema, got %d", len(merged.Properties))
	}
	// Only "name" is required (present in both)
	if len(merged.Required) != 1 || merged.Required[0] != "name" {
		t.Errorf("expected only 'name' required, got %v", merged.Required)
	}
}
