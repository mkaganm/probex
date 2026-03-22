package generator

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/probex/probex/internal/models"
)

// GraphQL generates tests specific to GraphQL endpoints.
type GraphQL struct{}

// NewGraphQL creates a new GraphQL test generator.
func NewGraphQL() *GraphQL { return &GraphQL{} }

// Category returns the test category.
func (g *GraphQL) Category() models.TestCategory { return models.CategoryHappyPath }

// Generate creates GraphQL-specific test cases.
func (g *GraphQL) Generate(endpoint models.Endpoint) ([]models.TestCase, error) {
	// Only process GraphQL endpoints.
	if !isGraphQLEndpoint(endpoint) {
		return nil, nil
	}

	var tests []models.TestCase

	opName := endpoint.Headers["X-GraphQL-Operation"]
	opType := endpoint.Headers["X-GraphQL-Type"]

	if opName == "" || opType == "" {
		return nil, nil
	}

	// 1. Basic operation execution.
	query := buildGraphQLQuery(opName, opType, endpoint)
	tests = append(tests, models.TestCase{
		Name:        fmt.Sprintf("GraphQL %s %s executes successfully", opType, opName),
		Description: fmt.Sprintf("Verify GraphQL %s '%s' returns valid response", opType, opName),
		Category:    models.CategoryHappyPath,
		Severity:    models.SeverityHigh,
		Request:     buildGraphQLRequest(endpoint, query, nil),
		Assertions: []models.Assertion{
			{Type: models.AssertStatusCode, Target: "status_code", Operator: "eq", Expected: 200},
			{Type: models.AssertBody, Target: "@valid", Operator: "eq", Expected: true},
			{Type: models.AssertBody, Target: "data", Operator: "exists", Expected: true},
		},
	})

	// 2. Check for GraphQL errors field.
	tests = append(tests, models.TestCase{
		Name:        fmt.Sprintf("GraphQL %s %s has no errors", opType, opName),
		Description: "Verify no errors array in response",
		Category:    models.CategoryHappyPath,
		Severity:    models.SeverityHigh,
		Request:     buildGraphQLRequest(endpoint, query, nil),
		Assertions: []models.Assertion{
			{Type: models.AssertStatusCode, Target: "status_code", Operator: "eq", Expected: 200},
			{Type: models.AssertBody, Target: "errors", Operator: "not_exists", Expected: nil},
		},
	})

	// 3. Introspection disabled check (security).
	tests = append(tests, models.TestCase{
		Name:        "GraphQL introspection should be disabled in production",
		Description: "Introspection queries should be disabled to prevent schema exposure",
		Category:    models.CategorySecurity,
		Severity:    models.SeverityMedium,
		Request:     buildGraphQLRequest(endpoint, `{ __schema { types { name } } }`, nil),
		Assertions: []models.Assertion{
			{Type: models.AssertBody, Target: "data.__schema", Operator: "not_exists", Expected: nil},
		},
		Tags: []string{"graphql", "security", "introspection"},
	})

	// 4. Syntax error handling.
	tests = append(tests, models.TestCase{
		Name:        "GraphQL handles syntax errors gracefully",
		Description: "Send malformed query and verify proper error response",
		Category:    models.CategoryEdgeCase,
		Severity:    models.SeverityMedium,
		Request:     buildGraphQLRequest(endpoint, "{ invalid query {{{{", nil),
		Assertions: []models.Assertion{
			{Type: models.AssertStatusCode, Target: "status_code", Operator: "eq", Expected: 200},
			{Type: models.AssertBody, Target: "errors", Operator: "exists", Expected: true},
		},
		Tags: []string{"graphql", "error-handling"},
	})

	// 5. Query depth limit (security).
	deepQuery := buildDeepNestedQuery(opName, 10)
	tests = append(tests, models.TestCase{
		Name:        "GraphQL enforces query depth limit",
		Description: "Deeply nested queries should be rejected to prevent DoS",
		Category:    models.CategorySecurity,
		Severity:    models.SeverityHigh,
		Request:     buildGraphQLRequest(endpoint, deepQuery, nil),
		Assertions: []models.Assertion{
			{Type: models.AssertBody, Target: "errors", Operator: "exists", Expected: true},
		},
		Tags: []string{"graphql", "security", "depth-limit"},
	})

	// 6. Query complexity / batch attack.
	if opType == "query" {
		batchQuery := buildBatchQuery(opName, 100)
		tests = append(tests, models.TestCase{
			Name:        "GraphQL enforces query complexity limit",
			Description: "Batched aliases should be limited to prevent resource exhaustion",
			Category:    models.CategorySecurity,
			Severity:    models.SeverityHigh,
			Request:     buildGraphQLRequest(endpoint, batchQuery, nil),
			Assertions: []models.Assertion{
				{Type: models.AssertResponseTime, Target: "duration", Operator: "lt", Expected: float64(10 * time.Second)},
			},
			Tags: []string{"graphql", "security", "complexity"},
		})
	}

	// 7. Missing required variables.
	if endpoint.RequestBody != nil && len(endpoint.RequestBody.Required) > 0 {
		tests = append(tests, models.TestCase{
			Name:        fmt.Sprintf("GraphQL %s %s requires variables", opType, opName),
			Description: "Send query without required variables",
			Category:    models.CategoryEdgeCase,
			Severity:    models.SeverityMedium,
			Request:     buildGraphQLRequest(endpoint, query, nil), // no variables
			Assertions: []models.Assertion{
				{Type: models.AssertStatusCode, Target: "status_code", Operator: "eq", Expected: 200},
			},
			Tags: []string{"graphql", "validation"},
		})
	}

	return tests, nil
}

func isGraphQLEndpoint(ep models.Endpoint) bool {
	for _, tag := range ep.Tags {
		if tag == "graphql" {
			return true
		}
	}
	return ep.Method == "QUERY" || ep.Method == "MUTATION" || ep.Method == "SUBSCRIPTION"
}

func buildGraphQLQuery(opName, opType string, ep models.Endpoint) string {
	// Build a minimal query with arguments.
	var args []string
	var params []string
	for _, p := range ep.QueryParams {
		gqlType := toGraphQLType(p.Type)
		if p.Required {
			gqlType += "!"
		}
		params = append(params, fmt.Sprintf("$%s: %s", p.Name, gqlType))
		args = append(args, fmt.Sprintf("%s: $%s", p.Name, p.Name))
	}

	var paramStr, argStr string
	if len(params) > 0 {
		paramStr = "(" + strings.Join(params, ", ") + ")"
		argStr = "(" + strings.Join(args, ", ") + ")"
	}

	return fmt.Sprintf("%s %s%s { %s%s { __typename } }", opType, opName, paramStr, opName, argStr)
}

func buildGraphQLRequest(ep models.Endpoint, query string, variables map[string]any) models.TestRequest {
	body := map[string]any{
		"query": query,
	}
	if variables != nil {
		body["variables"] = variables
	}

	bodyJSON, _ := json.Marshal(body)

	return models.TestRequest{
		Method: "POST",
		URL:    ep.BaseURL + ep.Path,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body:    string(bodyJSON),
		Timeout: 30 * time.Second,
	}
}

func buildDeepNestedQuery(field string, depth int) string {
	var sb strings.Builder
	sb.WriteString("{ ")
	for i := 0; i < depth; i++ {
		sb.WriteString(field)
		sb.WriteString(" { ")
	}
	sb.WriteString("__typename")
	for i := 0; i < depth; i++ {
		sb.WriteString(" }")
	}
	sb.WriteString(" }")
	return sb.String()
}

func buildBatchQuery(field string, count int) string {
	var sb strings.Builder
	sb.WriteString("{ ")
	for i := 0; i < count; i++ {
		sb.WriteString(fmt.Sprintf("a%d: %s { __typename } ", i, field))
	}
	sb.WriteString("}")
	return sb.String()
}

func toGraphQLType(jsonType string) string {
	switch jsonType {
	case "string", "String", "ID":
		return "String"
	case "integer", "Int":
		return "Int"
	case "number", "Float":
		return "Float"
	case "boolean", "Boolean":
		return "Boolean"
	default:
		return "String"
	}
}
