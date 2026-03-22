package generator

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/probex/probex/internal/models"
)

// EdgeCase generates tests for boundary conditions and unusual inputs.
type EdgeCase struct{}

// NewEdgeCase creates a new EdgeCase generator.
func NewEdgeCase() *EdgeCase { return &EdgeCase{} }

// Category returns the test category.
func (e *EdgeCase) Category() models.TestCategory { return models.CategoryEdgeCase }

// Generate creates edge case test cases for an endpoint.
func (e *EdgeCase) Generate(endpoint models.Endpoint) ([]models.TestCase, error) {
	var tests []models.TestCase

	method := strings.ToUpper(endpoint.Method)
	hasBody := method == "POST" || method == "PUT" || method == "PATCH"

	if !hasBody {
		return tests, nil
	}

	baseReq := buildBaseRequest(endpoint)
	errorAssertions := []models.Assertion{
		{
			Type:     models.AssertStatusCode,
			Target:   "status_code",
			Operator: "gte",
			Expected: 400,
		},
		{
			Type:     models.AssertStatusCode,
			Target:   "status_code",
			Operator: "lte",
			Expected: 422,
		},
	}

	// 1. Empty body
	tests = append(tests, models.TestCase{
		Name:        fmt.Sprintf("%s %s with empty body", endpoint.Method, endpoint.Path),
		Description: "Send request with empty body to verify proper error handling",
		Category:    models.CategoryEdgeCase,
		Severity:    models.SeverityMedium,
		Request: models.TestRequest{
			Method:  baseReq.Method,
			URL:     baseReq.URL,
			Headers: copyHeaders(baseReq.Headers),
			Body:    "",
			Timeout: 30 * time.Second,
		},
		Assertions: errorAssertions,
	})

	// Generate field-level edge cases only if we have a request body schema
	if endpoint.RequestBody != nil && endpoint.RequestBody.Type == "object" && endpoint.RequestBody.Properties != nil {
		exampleBody := buildExampleBody(endpoint.RequestBody)
		bodyMap, ok := exampleBody.(map[string]any)
		if !ok {
			return tests, nil
		}

		// 2. Missing required fields (one at a time)
		for _, reqField := range endpoint.RequestBody.Required {
			mutated := copyMap(bodyMap)
			delete(mutated, reqField)
			b, err := json.Marshal(mutated)
			if err != nil {
				continue
			}
			tests = append(tests, models.TestCase{
				Name:        fmt.Sprintf("%s %s missing required field '%s'", endpoint.Method, endpoint.Path, reqField),
				Description: fmt.Sprintf("Omit required field '%s' to verify validation", reqField),
				Category:    models.CategoryEdgeCase,
				Severity:    models.SeverityMedium,
				Request: models.TestRequest{
					Method:  baseReq.Method,
					URL:     baseReq.URL,
					Headers: copyHeaders(baseReq.Headers),
					Body:    string(b),
					Timeout: 30 * time.Second,
				},
				Assertions: errorAssertions,
			})
		}

		// 3. Wrong type values
		for fieldName, prop := range endpoint.RequestBody.Properties {
			wrongVal := wrongTypeValue(prop.Type)
			if wrongVal == nil {
				continue
			}
			mutated := copyMap(bodyMap)
			mutated[fieldName] = wrongVal
			b, err := json.Marshal(mutated)
			if err != nil {
				continue
			}
			tests = append(tests, models.TestCase{
				Name:        fmt.Sprintf("%s %s wrong type for field '%s'", endpoint.Method, endpoint.Path, fieldName),
				Description: fmt.Sprintf("Send wrong type value for field '%s' (expected %s)", fieldName, prop.Type),
				Category:    models.CategoryEdgeCase,
				Severity:    models.SeverityMedium,
				Request: models.TestRequest{
					Method:  baseReq.Method,
					URL:     baseReq.URL,
					Headers: copyHeaders(baseReq.Headers),
					Body:    string(b),
					Timeout: 30 * time.Second,
				},
				Assertions: errorAssertions,
			})
		}

		// 4. Boundary values for each field
		for fieldName, prop := range endpoint.RequestBody.Properties {
			boundaryTests := boundaryValues(fieldName, prop)
			for _, bt := range boundaryTests {
				mutated := copyMap(bodyMap)
				mutated[fieldName] = bt.value
				b, err := json.Marshal(mutated)
				if err != nil {
					continue
				}
				tests = append(tests, models.TestCase{
					Name:        fmt.Sprintf("%s %s boundary '%s' for field '%s'", endpoint.Method, endpoint.Path, bt.label, fieldName),
					Description: fmt.Sprintf("Test boundary condition '%s' on field '%s'", bt.label, fieldName),
					Category:    models.CategoryEdgeCase,
					Severity:    models.SeverityLow,
					Request: models.TestRequest{
						Method:  baseReq.Method,
						URL:     baseReq.URL,
						Headers: copyHeaders(baseReq.Headers),
						Body:    string(b),
						Timeout: 30 * time.Second,
					},
					Assertions: errorAssertions,
				})
			}
		}

		// 5. Null values for each field
		for fieldName := range endpoint.RequestBody.Properties {
			mutated := copyMap(bodyMap)
			mutated[fieldName] = nil
			b, err := json.Marshal(mutated)
			if err != nil {
				continue
			}
			tests = append(tests, models.TestCase{
				Name:        fmt.Sprintf("%s %s null value for field '%s'", endpoint.Method, endpoint.Path, fieldName),
				Description: fmt.Sprintf("Send null value for field '%s' to verify null handling", fieldName),
				Category:    models.CategoryEdgeCase,
				Severity:    models.SeverityLow,
				Request: models.TestRequest{
					Method:  baseReq.Method,
					URL:     baseReq.URL,
					Headers: copyHeaders(baseReq.Headers),
					Body:    string(b),
					Timeout: 30 * time.Second,
				},
				Assertions: errorAssertions,
			})
		}

		// 6. Extra unexpected fields
		mutated := copyMap(bodyMap)
		mutated["__unexpected_field"] = "unexpected_value"
		mutated["admin"] = true
		b, err := json.Marshal(mutated)
		if err == nil {
			tests = append(tests, models.TestCase{
				Name:        fmt.Sprintf("%s %s with extra unexpected fields", endpoint.Method, endpoint.Path),
				Description: "Send request with extra unexpected fields to verify strict parsing",
				Category:    models.CategoryEdgeCase,
				Severity:    models.SeverityLow,
				Request: models.TestRequest{
					Method:  baseReq.Method,
					URL:     baseReq.URL,
					Headers: copyHeaders(baseReq.Headers),
					Body:    string(b),
					Timeout: 30 * time.Second,
				},
				Assertions: []models.Assertion{
					{
						Type:     models.AssertStatusCode,
						Target:   "status_code",
						Operator: "ne",
						Expected: 500,
					},
				},
			})
		}
	}

	return tests, nil
}

type boundaryTest struct {
	label string
	value any
}

// boundaryValues returns boundary test values for a field based on its type.
func boundaryValues(fieldName string, schema *models.Schema) []boundaryTest {
	var tests []boundaryTest

	switch schema.Type {
	case "string":
		tests = append(tests,
			boundaryTest{"empty_string", ""},
			boundaryTest{"very_long_string", strings.Repeat("a", 10000)},
		)
	case "integer", "number":
		tests = append(tests,
			boundaryTest{"zero", 0},
			boundaryTest{"negative", -1},
			boundaryTest{"max_int", math.MaxInt32},
		)
	}
	_ = fieldName
	return tests
}

// wrongTypeValue returns a value of the wrong type for a given expected type.
func wrongTypeValue(expectedType string) any {
	switch expectedType {
	case "string":
		return 12345
	case "integer", "number":
		return "not_a_number"
	case "boolean":
		return "not_a_bool"
	case "array":
		return "not_an_array"
	case "object":
		return "not_an_object"
	default:
		return nil
	}
}

// copyMap creates a shallow copy of a map.
func copyMap(m map[string]any) map[string]any {
	cp := make(map[string]any, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return cp
}

// copyHeaders creates a copy of a headers map.
func copyHeaders(h map[string]string) map[string]string {
	if h == nil {
		return make(map[string]string)
	}
	cp := make(map[string]string, len(h))
	for k, v := range h {
		cp[k] = v
	}
	return cp
}
