package schema

import (
	"encoding/json"
	"regexp"

	"github.com/mkaganm/probex/internal/models"
)

// Pattern regexes for string format detection.
var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	uuidRegex  = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	// ISO 8601 date-time (e.g., 2024-01-15T10:30:00Z or with offset).
	isoDateTimeRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`)
	// ISO 8601 date only.
	isoDateRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	urlRegex     = regexp.MustCompile(`^https?://`)
)

// Inferrer analyzes JSON responses to infer their schema.
type Inferrer struct{}

// New creates a new schema Inferrer.
func New() *Inferrer {
	return &Inferrer{}
}

// InferFromJSON infers a JSON schema from a raw JSON response body.
func (i *Inferrer) InferFromJSON(data []byte) (*models.Schema, error) {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	return inferValue(value), nil
}

// inferValue recursively infers the schema of an arbitrary JSON value.
func inferValue(v any) *models.Schema {
	if v == nil {
		return &models.Schema{Type: "null"}
	}

	switch val := v.(type) {
	case bool:
		return &models.Schema{Type: "boolean"}

	case float64:
		// json.Unmarshal decodes all numbers as float64.
		// Check if it's actually an integer.
		if val == float64(int64(val)) {
			return &models.Schema{Type: "integer"}
		}
		return &models.Schema{Type: "number"}

	case string:
		s := &models.Schema{Type: "string"}
		s.Format = detectStringFormat(val)
		return s

	case []any:
		s := &models.Schema{Type: "array"}
		if len(val) > 0 {
			s.Items = inferValue(val[0])
		}
		return s

	case map[string]any:
		s := &models.Schema{
			Type:       "object",
			Properties: make(map[string]*models.Schema),
			Required:   make([]string, 0, len(val)),
		}
		for key, child := range val {
			s.Properties[key] = inferValue(child)
			// In a single observation, all present fields are "observed".
			s.Required = append(s.Required, key)
		}
		return s

	default:
		return &models.Schema{Type: "string"}
	}
}

// detectStringFormat checks if a string matches known formats.
func detectStringFormat(s string) string {
	switch {
	case uuidRegex.MatchString(s):
		return "uuid"
	case emailRegex.MatchString(s):
		return "email"
	case isoDateTimeRegex.MatchString(s):
		return "date-time"
	case isoDateRegex.MatchString(s):
		return "date"
	case urlRegex.MatchString(s):
		return "uri"
	default:
		return ""
	}
}

// Merge merges multiple observed schemas into a unified schema.
// Properties are unioned, and required is set to fields present in ALL schemas.
// Conflicting types are widened (e.g., integer+number -> number, anything+string -> string).
func (i *Inferrer) Merge(schemas []*models.Schema) *models.Schema {
	if len(schemas) == 0 {
		return nil
	}
	if len(schemas) == 1 {
		return schemas[0]
	}

	result := deepCopySchema(schemas[0])
	for _, s := range schemas[1:] {
		result = mergeTwo(result, s)
	}
	return result
}

// mergeTwo merges two schemas together.
func mergeTwo(a, b *models.Schema) *models.Schema {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}

	merged := &models.Schema{}

	// Widen types if they conflict.
	merged.Type = widenType(a.Type, b.Type)

	// Merge format: keep only if both agree.
	if a.Format == b.Format {
		merged.Format = a.Format
	}

	// Merge object properties.
	if a.Properties != nil || b.Properties != nil {
		merged.Properties = make(map[string]*models.Schema)

		// Collect all property names.
		allProps := make(map[string]bool)
		for k := range a.Properties {
			allProps[k] = true
		}
		for k := range b.Properties {
			allProps[k] = true
		}

		for prop := range allProps {
			aProp := a.Properties[prop]
			bProp := b.Properties[prop]
			if aProp != nil && bProp != nil {
				merged.Properties[prop] = mergeTwo(aProp, bProp)
			} else if aProp != nil {
				merged.Properties[prop] = deepCopySchema(aProp)
			} else {
				merged.Properties[prop] = deepCopySchema(bProp)
			}
		}

		// Required: only fields present in BOTH schemas' required lists.
		aReq := toSet(a.Required)
		bReq := toSet(b.Required)
		for field := range aReq {
			if bReq[field] {
				merged.Required = append(merged.Required, field)
			}
		}
	}

	// Merge array items.
	if a.Items != nil || b.Items != nil {
		merged.Items = mergeTwo(a.Items, b.Items)
	}

	// Merge pattern: keep only if both agree.
	if a.Pattern == b.Pattern {
		merged.Pattern = a.Pattern
	}

	return merged
}

// widenType returns the wider of two types.
func widenType(a, b string) string {
	if a == b {
		return a
	}
	if a == "null" {
		return b
	}
	if b == "null" {
		return a
	}
	// integer + number -> number
	if (a == "integer" && b == "number") || (a == "number" && b == "integer") {
		return "number"
	}
	// Any other conflict -> string (the widest type).
	return "string"
}

// toSet converts a string slice to a set.
func toSet(items []string) map[string]bool {
	s := make(map[string]bool, len(items))
	for _, item := range items {
		s[item] = true
	}
	return s
}

// deepCopySchema makes a deep copy of a schema via JSON round-trip.
func deepCopySchema(s *models.Schema) *models.Schema {
	if s == nil {
		return nil
	}
	data, err := json.Marshal(s)
	if err != nil {
		return s
	}
	var copy models.Schema
	if err := json.Unmarshal(data, &copy); err != nil {
		return s
	}
	return &copy
}
