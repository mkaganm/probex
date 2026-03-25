package runner

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/tidwall/gjson"

	"github.com/mkaganm/probex/internal/models"
)

// AssertionEngine evaluates assertions against actual HTTP responses.
type AssertionEngine struct{}

// NewAssertionEngine creates a new AssertionEngine.
func NewAssertionEngine() *AssertionEngine {
	return &AssertionEngine{}
}

// Evaluate checks a single assertion against an actual response.
func (ae *AssertionEngine) Evaluate(assertion models.Assertion, response *models.TestResponse) models.AssertionResult {
	if response == nil {
		return models.AssertionResult{
			Assertion: assertion,
			Passed:    false,
			Message:   "no response received",
		}
	}

	switch assertion.Type {
	case models.AssertStatusCode:
		return ae.evaluateStatusCode(assertion, response)
	case models.AssertBody:
		return ae.evaluateBody(assertion, response)
	case models.AssertHeader:
		return ae.evaluateHeader(assertion, response)
	case models.AssertResponseTime:
		return ae.evaluateResponseTime(assertion, response)
	case models.AssertSchema:
		return ae.evaluateSchema(assertion, response)
	default:
		return models.AssertionResult{
			Assertion: assertion,
			Passed:    false,
			Message:   fmt.Sprintf("unknown assertion type: %s", assertion.Type),
		}
	}
}

func (ae *AssertionEngine) evaluateStatusCode(a models.Assertion, r *models.TestResponse) models.AssertionResult {
	actual := r.StatusCode
	expected := toInt(a.Expected)
	passed := compareInt(actual, expected, a.Operator)
	msg := ""
	if !passed {
		msg = fmt.Sprintf("expected status_code %s %d, got %d", a.Operator, expected, actual)
	}
	return models.AssertionResult{Assertion: a, Passed: passed, Actual: actual, Message: msg}
}

func (ae *AssertionEngine) evaluateBody(a models.Assertion, r *models.TestResponse) models.AssertionResult {
	body := r.Body

	// Special targets
	if a.Target == "@valid" {
		valid := json.Valid([]byte(body))
		passed := compareBool(valid, a.Expected, a.Operator)
		msg := ""
		if !passed {
			msg = "response body is not valid JSON"
		}
		return models.AssertionResult{Assertion: a, Passed: passed, Actual: valid, Message: msg}
	}

	if a.Target == "@raw" {
		return ae.evaluateRawBody(a, body)
	}

	// JSONPath extraction using gjson
	result := gjson.Get(body, a.Target)
	if !result.Exists() {
		return models.AssertionResult{
			Assertion: a,
			Passed:    a.Operator == "not_exists",
			Actual:    nil,
			Message:   fmt.Sprintf("path '%s' not found in response body", a.Target),
		}
	}

	actual := result.Value()
	passed := compareAny(actual, a.Expected, a.Operator)
	msg := ""
	if !passed {
		msg = fmt.Sprintf("body path '%s': expected %s %v, got %v", a.Target, a.Operator, a.Expected, actual)
	}
	return models.AssertionResult{Assertion: a, Passed: passed, Actual: actual, Message: msg}
}

func (ae *AssertionEngine) evaluateRawBody(a models.Assertion, body string) models.AssertionResult {
	expected := fmt.Sprintf("%v", a.Expected)
	var passed bool
	switch a.Operator {
	case "contains":
		passed = strings.Contains(body, expected)
	case "not_contains":
		passed = !strings.Contains(body, expected)
	case "eq":
		passed = body == expected
	default:
		passed = false
	}
	msg := ""
	if !passed {
		msg = fmt.Sprintf("body %s check failed for '%s'", a.Operator, truncate(expected, 50))
	}
	return models.AssertionResult{Assertion: a, Passed: passed, Actual: truncate(body, 100), Message: msg}
}

func (ae *AssertionEngine) evaluateHeader(a models.Assertion, r *models.TestResponse) models.AssertionResult {
	actual, ok := r.Headers[a.Target]
	if !ok {
		// Try case-insensitive lookup
		for k, v := range r.Headers {
			if strings.EqualFold(k, a.Target) {
				actual = v
				ok = true
				break
			}
		}
	}

	if !ok {
		if a.Operator == "not_exists" {
			return models.AssertionResult{Assertion: a, Passed: true, Actual: nil}
		}
		return models.AssertionResult{
			Assertion: a, Passed: false, Actual: nil,
			Message: fmt.Sprintf("header '%s' not found", a.Target),
		}
	}

	expected := fmt.Sprintf("%v", a.Expected)
	passed := compareString(actual, expected, a.Operator)
	msg := ""
	if !passed {
		msg = fmt.Sprintf("header '%s': expected %s '%s', got '%s'", a.Target, a.Operator, expected, actual)
	}
	return models.AssertionResult{Assertion: a, Passed: passed, Actual: actual, Message: msg}
}

func (ae *AssertionEngine) evaluateResponseTime(a models.Assertion, r *models.TestResponse) models.AssertionResult {
	actual := float64(r.Duration)
	expected := toFloat64(a.Expected)
	passed := compareFloat(actual, expected, a.Operator)
	msg := ""
	if !passed {
		msg = fmt.Sprintf("response time %s: expected %s %s, got %s",
			a.Operator, a.Operator, time.Duration(int64(expected)), time.Duration(int64(actual)))
	}
	return models.AssertionResult{Assertion: a, Passed: passed, Actual: actual, Message: msg}
}

func (ae *AssertionEngine) evaluateSchema(a models.Assertion, r *models.TestResponse) models.AssertionResult {
	if !json.Valid([]byte(r.Body)) {
		return models.AssertionResult{Assertion: a, Passed: false, Message: "response body is not valid JSON"}
	}

	// Basic type checking
	var body any
	if err := json.Unmarshal([]byte(r.Body), &body); err != nil {
		return models.AssertionResult{Assertion: a, Passed: false, Message: "failed to parse response body"}
	}

	expectedMap, ok := a.Expected.(map[string]any)
	if !ok {
		return models.AssertionResult{Assertion: a, Passed: true, Message: "schema assertion skipped: expected not a schema map"}
	}

	err := validateAgainstSchema(body, expectedMap)
	if err != nil {
		return models.AssertionResult{Assertion: a, Passed: false, Message: err.Error()}
	}
	return models.AssertionResult{Assertion: a, Passed: true}
}

func validateAgainstSchema(value any, schema map[string]any) error {
	expectedType, _ := schema["type"].(string)
	if expectedType == "" {
		return nil
	}

	actualType := jsonType(value)
	if actualType != expectedType {
		// integer is compatible with number
		if expectedType == "number" && actualType == "integer" {
			return nil
		}
		return fmt.Errorf("expected type %s, got %s", expectedType, actualType)
	}

	// Check object properties
	if expectedType == "object" {
		obj, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("expected object, got %T", value)
		}
		if propsRaw, ok := schema["properties"]; ok {
			if props, ok := propsRaw.(map[string]any); ok {
				for key, propSchema := range props {
					if pSchema, ok := propSchema.(map[string]any); ok {
						if val, exists := obj[key]; exists {
							if err := validateAgainstSchema(val, pSchema); err != nil {
								return fmt.Errorf("property '%s': %w", key, err)
							}
						}
					}
				}
			}
		}
	}

	return nil
}

func jsonType(v any) string {
	switch v := v.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case float64:
		if v == float64(int64(v)) {
			return "integer"
		}
		return "number"
	case string:
		return "string"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	default:
		_ = v
		return "unknown"
	}
}

func compareInt(actual, expected int, op string) bool {
	switch op {
	case "eq":
		return actual == expected
	case "ne":
		return actual != expected
	case "gt":
		return actual > expected
	case "gte":
		return actual >= expected
	case "lt":
		return actual < expected
	case "lte":
		return actual <= expected
	default:
		return actual == expected
	}
}

func compareFloat(actual, expected float64, op string) bool {
	switch op {
	case "eq":
		return actual == expected
	case "ne":
		return actual != expected
	case "gt":
		return actual > expected
	case "gte":
		return actual >= expected
	case "lt":
		return actual < expected
	case "lte":
		return actual <= expected
	default:
		return actual == expected
	}
}

func compareString(actual, expected, op string) bool {
	switch op {
	case "eq":
		return actual == expected
	case "ne":
		return actual != expected
	case "contains":
		return strings.Contains(actual, expected)
	case "not_contains":
		return !strings.Contains(actual, expected)
	case "exists":
		return actual != ""
	default:
		return actual == expected
	}
}

func compareBool(actual any, expected any, op string) bool {
	a := fmt.Sprintf("%v", actual)
	e := fmt.Sprintf("%v", expected)
	return a == e
}

func compareAny(actual, expected any, op string) bool {
	aStr := fmt.Sprintf("%v", actual)
	eStr := fmt.Sprintf("%v", expected)
	switch op {
	case "eq":
		return aStr == eStr
	case "ne":
		return aStr != eStr
	case "contains":
		return strings.Contains(aStr, eStr)
	case "not_contains":
		return !strings.Contains(aStr, eStr)
	case "exists":
		return actual != nil
	case "not_exists":
		return actual == nil
	case "gt", "gte", "lt", "lte":
		af := toFloat64(actual)
		ef := toFloat64(expected)
		return compareFloat(af, ef, op)
	default:
		return aStr == eStr
	}
}

func toInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	case int64:
		return int(n)
	default:
		return 0
	}
}

func toFloat64(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return 0
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
