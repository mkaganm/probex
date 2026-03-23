package generator

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mkaganm/probex/internal/models"
)

// HappyPath generates tests for expected successful API behavior.
type HappyPath struct{}

// NewHappyPath creates a new HappyPath generator.
func NewHappyPath() *HappyPath { return &HappyPath{} }

// Category returns the test category.
func (h *HappyPath) Category() models.TestCategory { return models.CategoryHappyPath }

// Generate creates happy path test cases for an endpoint.
func (h *HappyPath) Generate(endpoint models.Endpoint) ([]models.TestCase, error) {
	var tests []models.TestCase

	baseReq := buildBaseRequest(endpoint)

	// 1. Status code check
	expectedStatus := expectedStatusForMethod(endpoint.Method)
	tests = append(tests, models.TestCase{
		Name:        fmt.Sprintf("%s %s returns %d", endpoint.Method, endpoint.Path, expectedStatus),
		Description: fmt.Sprintf("Verify %s %s returns expected status code %d", endpoint.Method, endpoint.Path, expectedStatus),
		Category:    models.CategoryHappyPath,
		Severity:    models.SeverityHigh,
		Request:     baseReq,
		Assertions: []models.Assertion{
			{
				Type:     models.AssertStatusCode,
				Target:   "status_code",
				Operator: "eq",
				Expected: expectedStatus,
			},
		},
	})

	// 2. Response body is valid JSON (for endpoints that return JSON)
	if producesJSON(endpoint) {
		tests = append(tests, models.TestCase{
			Name:        fmt.Sprintf("%s %s returns valid JSON", endpoint.Method, endpoint.Path),
			Description: "Verify the response body is valid JSON",
			Category:    models.CategoryHappyPath,
			Severity:    models.SeverityHigh,
			Request:     baseReq,
			Assertions: []models.Assertion{
				{
					Type:     models.AssertBody,
					Target:   "@valid",
					Operator: "eq",
					Expected: true,
				},
			},
		})
	}

	// 3. Content-Type header matches
	expectedCT := expectedContentType(endpoint)
	if expectedCT != "" {
		tests = append(tests, models.TestCase{
			Name:        fmt.Sprintf("%s %s has correct Content-Type", endpoint.Method, endpoint.Path),
			Description: fmt.Sprintf("Verify Content-Type header contains %s", expectedCT),
			Category:    models.CategoryHappyPath,
			Severity:    models.SeverityMedium,
			Request:     baseReq,
			Assertions: []models.Assertion{
				{
					Type:     models.AssertHeader,
					Target:   "Content-Type",
					Operator: "contains",
					Expected: expectedCT,
				},
			},
		})
	}

	// 4. Response time under threshold (5s)
	tests = append(tests, models.TestCase{
		Name:        fmt.Sprintf("%s %s responds within 5s", endpoint.Method, endpoint.Path),
		Description: "Verify the endpoint responds within a reasonable time threshold",
		Category:    models.CategoryHappyPath,
		Severity:    models.SeverityMedium,
		Request:     baseReq,
		Assertions: []models.Assertion{
			{
				Type:     models.AssertResponseTime,
				Target:   "duration",
				Operator: "lt",
				Expected: float64(5 * time.Second),
			},
		},
	})

	// 5. Schema validation if response schema is known
	if len(endpoint.Responses) > 0 {
		for _, resp := range endpoint.Responses {
			if resp.Schema != nil && resp.StatusCode == expectedStatus {
				tests = append(tests, models.TestCase{
					Name:        fmt.Sprintf("%s %s response matches schema", endpoint.Method, endpoint.Path),
					Description: "Verify the response body conforms to the expected schema",
					Category:    models.CategoryHappyPath,
					Severity:    models.SeverityHigh,
					Request:     baseReq,
					Assertions: []models.Assertion{
						{
							Type:     models.AssertSchema,
							Target:   "body",
							Operator: "matches",
							Expected: schemaToMap(resp.Schema),
						},
					},
				})
				break
			}
		}
	}

	return tests, nil
}

// expectedStatusForMethod returns the expected successful HTTP status code for a method.
func expectedStatusForMethod(method string) int {
	switch strings.ToUpper(method) {
	case "POST":
		return 201
	case "DELETE":
		return 204
	default:
		return 200
	}
}

// producesJSON returns true if the endpoint is expected to return JSON.
func producesJSON(ep models.Endpoint) bool {
	for _, r := range ep.Responses {
		if strings.Contains(r.ContentType, "json") {
			return true
		}
	}
	// Default assumption: most APIs return JSON
	if len(ep.Responses) == 0 && strings.ToUpper(ep.Method) != "DELETE" {
		return true
	}
	return false
}

// expectedContentType determines the expected content type for an endpoint.
func expectedContentType(ep models.Endpoint) string {
	for _, r := range ep.Responses {
		if r.ContentType != "" {
			return r.ContentType
		}
	}
	if strings.ToUpper(ep.Method) != "DELETE" {
		return "application/json"
	}
	return ""
}

// buildBaseRequest creates a standard request from an endpoint, using example values.
func buildBaseRequest(ep models.Endpoint) models.TestRequest {
	req := models.TestRequest{
		Method:  ep.Method,
		URL:     ep.FullURL(),
		Headers: make(map[string]string),
		Timeout: 30 * time.Second,
	}

	// Copy endpoint headers
	for k, v := range ep.Headers {
		req.Headers[k] = v
	}

	// Build request body from schema if available
	if ep.RequestBody != nil && (strings.ToUpper(ep.Method) == "POST" || strings.ToUpper(ep.Method) == "PUT" || strings.ToUpper(ep.Method) == "PATCH") {
		body := buildExampleBody(ep.RequestBody)
		if body != nil {
			b, err := json.Marshal(body)
			if err == nil {
				req.Body = string(b)
				req.Headers["Content-Type"] = "application/json"
			}
		}
	}

	// Add auth header if specified
	if ep.Auth != nil && ep.Auth.Type != models.AuthNone {
		if ep.Auth.Location == "header" && ep.Auth.Key != "" {
			req.Headers[ep.Auth.Key] = "{{auth_token}}"
		}
	}

	return req
}

// buildExampleBody creates an example JSON body from a schema.
func buildExampleBody(schema *models.Schema) any {
	if schema == nil {
		return nil
	}
	switch schema.Type {
	case "object":
		obj := make(map[string]any)
		for name, prop := range schema.Properties {
			obj[name] = buildExampleValue(prop)
		}
		return obj
	case "array":
		return []any{buildExampleValue(schema.Items)}
	default:
		return buildExampleValue(schema)
	}
}

// buildExampleValue creates an example value for a schema property.
func buildExampleValue(schema *models.Schema) any {
	if schema == nil {
		return nil
	}
	if len(schema.Enum) > 0 {
		return schema.Enum[0]
	}
	switch schema.Type {
	case "string":
		if schema.Format == "email" {
			return "test@example.com"
		}
		if schema.Format == "date-time" {
			return "2024-01-01T00:00:00Z"
		}
		if schema.Format == "uuid" {
			return "550e8400-e29b-41d4-a716-446655440000"
		}
		return "test_value"
	case "integer", "number":
		return 1
	case "boolean":
		return true
	case "object":
		return buildExampleBody(schema)
	case "array":
		item := buildExampleValue(schema.Items)
		if item != nil {
			return []any{item}
		}
		return []any{}
	default:
		return "test_value"
	}
}

// schemaToMap converts a Schema to a map representation for assertion comparison.
func schemaToMap(schema *models.Schema) map[string]any {
	if schema == nil {
		return nil
	}
	m := map[string]any{
		"type": schema.Type,
	}
	if len(schema.Required) > 0 {
		m["required"] = schema.Required
	}
	if schema.Properties != nil {
		props := make(map[string]any)
		for k, v := range schema.Properties {
			props[k] = schemaToMap(v)
		}
		m["properties"] = props
	}
	if schema.Items != nil {
		m["items"] = schemaToMap(schema.Items)
	}
	return m
}
