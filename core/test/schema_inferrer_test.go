package test

import (
	"testing"

	"github.com/mkaganm/probex/internal/models"
	schemapkg "github.com/mkaganm/probex/internal/schema"
)

func TestInferFromJSONObjectTypes(t *testing.T) {
	inf := schemapkg.New()
	data := []byte(`{"name": "John", "age": 30, "active": true, "score": 9.5}`)

	s, err := inf.InferFromJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	if s.Type != "object" {
		t.Fatalf("expected object, got %s", s.Type)
	}
	if s.Properties["name"].Type != "string" {
		t.Errorf("name: expected string, got %s", s.Properties["name"].Type)
	}
	if s.Properties["age"].Type != "integer" {
		t.Errorf("age: expected integer, got %s", s.Properties["age"].Type)
	}
	if s.Properties["active"].Type != "boolean" {
		t.Errorf("active: expected boolean, got %s", s.Properties["active"].Type)
	}
	if s.Properties["score"].Type != "number" {
		t.Errorf("score: expected number, got %s", s.Properties["score"].Type)
	}
}

func TestInferFromJSONArrayItems(t *testing.T) {
	inf := schemapkg.New()
	data := []byte(`[{"id": 1}, {"id": 2}]`)

	s, err := inf.InferFromJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	if s.Type != "array" {
		t.Fatalf("expected array, got %s", s.Type)
	}
	if s.Items == nil {
		t.Fatal("expected non-nil Items")
	}
	if s.Items.Type != "object" {
		t.Errorf("items type: expected object, got %s", s.Items.Type)
	}
}

func TestInferFromJSONNullType(t *testing.T) {
	inf := schemapkg.New()
	s, err := inf.InferFromJSON([]byte(`null`))
	if err != nil {
		t.Fatal(err)
	}
	if s.Type != "null" {
		t.Errorf("expected null, got %s", s.Type)
	}
}

func TestInferStringFormatDetection(t *testing.T) {
	inf := schemapkg.New()
	data := []byte(`{
		"email": "user@example.com",
		"id": "550e8400-e29b-41d4-a716-446655440000",
		"created": "2024-01-15T10:30:00Z",
		"birth_date": "1990-05-20",
		"website": "https://example.com",
		"plain": "hello world"
	}`)

	s, err := inf.InferFromJSON(data)
	if err != nil {
		t.Fatal(err)
	}

	cases := map[string]string{
		"email":      "email",
		"id":         "uuid",
		"created":    "date-time",
		"birth_date": "date",
		"website":    "uri",
		"plain":      "",
	}

	for field, expectedFormat := range cases {
		prop := s.Properties[field]
		if prop == nil {
			t.Errorf("missing property %s", field)
			continue
		}
		if prop.Format != expectedFormat {
			t.Errorf("%s format: got %q, want %q", field, prop.Format, expectedFormat)
		}
	}
}

func TestInferFromJSONInvalidInput(t *testing.T) {
	inf := schemapkg.New()
	_, err := inf.InferFromJSON([]byte(`{invalid`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestMergeSchemasPropertyUnion(t *testing.T) {
	inf := schemapkg.New()

	s1 := &models.Schema{
		Type: "object",
		Properties: map[string]*models.Schema{
			"name": {Type: "string"},
			"age":  {Type: "integer"},
		},
		Required: []string{"name", "age"},
	}

	s2 := &models.Schema{
		Type: "object",
		Properties: map[string]*models.Schema{
			"name":  {Type: "string"},
			"email": {Type: "string", Format: "email"},
		},
		Required: []string{"name", "email"},
	}

	merged := inf.Merge([]*models.Schema{s1, s2})

	if merged.Type != "object" {
		t.Fatalf("expected object, got %s", merged.Type)
	}
	// Union of properties.
	if len(merged.Properties) != 3 {
		t.Errorf("expected 3 properties, got %d", len(merged.Properties))
	}
	// Intersection of required.
	found := false
	for _, r := range merged.Required {
		if r == "name" {
			found = true
		}
	}
	if !found {
		t.Error("name should be in required (present in both)")
	}
	// "age" and "email" should NOT be required (only in one).
	for _, r := range merged.Required {
		if r == "age" || r == "email" {
			t.Errorf("%s should not be required (only in one schema)", r)
		}
	}
}

func TestMergeTypeWidening(t *testing.T) {
	inf := schemapkg.New()

	s1 := &models.Schema{Type: "integer"}
	s2 := &models.Schema{Type: "number"}

	merged := inf.Merge([]*models.Schema{s1, s2})
	if merged.Type != "number" {
		t.Errorf("integer + number should widen to number, got %s", merged.Type)
	}
}

func TestMergeNilInput(t *testing.T) {
	inf := schemapkg.New()
	if inf.Merge(nil) != nil {
		t.Error("expected nil for nil input")
	}
	if inf.Merge([]*models.Schema{}) != nil {
		t.Error("expected nil for empty input")
	}
}

func TestMergeSingleSchema(t *testing.T) {
	inf := schemapkg.New()
	s := &models.Schema{Type: "string", Format: "email"}
	result := inf.Merge([]*models.Schema{s})
	if result.Type != "string" || result.Format != "email" {
		t.Errorf("single schema should pass through unchanged")
	}
}
