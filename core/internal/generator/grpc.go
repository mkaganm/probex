package generator

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/probex/probex/internal/models"
)

// GRPC generates tests specific to gRPC endpoints.
type GRPC struct{}

// NewGRPC creates a new gRPC test generator.
func NewGRPC() *GRPC { return &GRPC{} }

// Category returns the test category.
func (g *GRPC) Category() models.TestCategory { return models.CategoryHappyPath }

// Generate creates gRPC-specific test cases.
func (g *GRPC) Generate(endpoint models.Endpoint) ([]models.TestCase, error) {
	if !isGRPCEndpoint(endpoint) {
		return nil, nil
	}

	var tests []models.TestCase

	serviceName := endpoint.Headers["X-GRPC-Service"]
	methodName := endpoint.Headers["X-GRPC-Method"]
	streamType := endpoint.Headers["X-GRPC-StreamType"]

	if serviceName == "" {
		serviceName = "UnknownService"
	}
	if methodName == "" {
		methodName = endpoint.Path
	}

	// 1. Basic unary call with empty request (JSON transcoding).
	tests = append(tests, models.TestCase{
		Name:        fmt.Sprintf("gRPC %s.%s responds to empty request", serviceName, methodName),
		Description: "Verify gRPC method handles empty request body via JSON transcoding",
		Category:    models.CategoryHappyPath,
		Severity:    models.SeverityHigh,
		Request:     buildGRPCRequest(endpoint, map[string]any{}),
		Assertions: []models.Assertion{
			{Type: models.AssertStatusCode, Target: "status_code", Operator: "ne", Expected: 404},
			{Type: models.AssertStatusCode, Target: "status_code", Operator: "ne", Expected: 405},
		},
		Tags: []string{"grpc", "unary"},
	})

	// 2. Content-Type check — gRPC should accept application/json (transcoding) or application/grpc.
	tests = append(tests, models.TestCase{
		Name:        fmt.Sprintf("gRPC %s.%s accepts JSON transcoding", serviceName, methodName),
		Description: "Verify gRPC endpoint accepts application/json content type",
		Category:    models.CategoryHappyPath,
		Severity:    models.SeverityMedium,
		Request:     buildGRPCRequest(endpoint, map[string]any{}),
		Assertions: []models.Assertion{
			{Type: models.AssertStatusCode, Target: "status_code", Operator: "lt", Expected: 500},
		},
		Tags: []string{"grpc", "transcoding"},
	})

	// 3. Response time check.
	tests = append(tests, models.TestCase{
		Name:        fmt.Sprintf("gRPC %s.%s responds within timeout", serviceName, methodName),
		Description: "Verify gRPC method responds within 5 seconds",
		Category:    models.CategoryPerformance,
		Severity:    models.SeverityMedium,
		Request:     buildGRPCRequest(endpoint, map[string]any{}),
		Assertions: []models.Assertion{
			{Type: models.AssertResponseTime, Target: "duration", Operator: "lt", Expected: float64(5 * time.Second)},
		},
		Tags: []string{"grpc", "performance"},
	})

	// 4. Invalid content type rejection.
	tests = append(tests, models.TestCase{
		Name:        fmt.Sprintf("gRPC %s.%s rejects invalid content type", serviceName, methodName),
		Description: "Verify gRPC endpoint rejects text/plain content type",
		Category:    models.CategoryEdgeCase,
		Severity:    models.SeverityLow,
		Request: models.TestRequest{
			Method: "POST",
			URL:    endpoint.FullURL(),
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
			Body:    "not json",
			Timeout: 10 * time.Second,
		},
		Assertions: []models.Assertion{
			{Type: models.AssertStatusCode, Target: "status_code", Operator: "ne", Expected: 200},
		},
		Tags: []string{"grpc", "edge-case"},
	})

	// 5. Large payload handling.
	largePayload := map[string]any{
		"data": generateLargeString(100000), // 100KB
	}
	tests = append(tests, models.TestCase{
		Name:        fmt.Sprintf("gRPC %s.%s handles large payload", serviceName, methodName),
		Description: "Verify gRPC method handles large request payloads gracefully",
		Category:    models.CategoryEdgeCase,
		Severity:    models.SeverityMedium,
		Request:     buildGRPCRequest(endpoint, largePayload),
		Assertions: []models.Assertion{
			{Type: models.AssertStatusCode, Target: "status_code", Operator: "lt", Expected: 500},
		},
		Tags: []string{"grpc", "edge-case", "payload"},
	})

	// 6. Auth check — gRPC without credentials.
	if endpoint.Auth != nil && endpoint.Auth.Type != models.AuthNone {
		tests = append(tests, models.TestCase{
			Name:        fmt.Sprintf("gRPC %s.%s requires authentication", serviceName, methodName),
			Description: "Verify gRPC method rejects unauthenticated requests",
			Category:    models.CategorySecurity,
			Severity:    models.SeverityHigh,
			Request: models.TestRequest{
				Method: "POST",
				URL:    endpoint.FullURL(),
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Body:    "{}",
				Timeout: 10 * time.Second,
			},
			Assertions: []models.Assertion{
				{Type: models.AssertStatusCode, Target: "status_code", Operator: "eq", Expected: 401},
			},
			Tags: []string{"grpc", "security", "auth"},
		})
	}

	// 7. Reflection disabled check (security).
	tests = append(tests, models.TestCase{
		Name:        "gRPC server reflection should be disabled in production",
		Description: "Server reflection exposes service definitions — should be disabled",
		Category:    models.CategorySecurity,
		Severity:    models.SeverityMedium,
		Request: models.TestRequest{
			Method: "POST",
			URL:    endpoint.BaseURL + "/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body:    `{"list_services":""}`,
			Timeout: 10 * time.Second,
		},
		Assertions: []models.Assertion{
			{Type: models.AssertStatusCode, Target: "status_code", Operator: "ne", Expected: 200},
		},
		Tags: []string{"grpc", "security", "reflection"},
	})

	// 8. Streaming-specific tests.
	if streamType == "server-stream" || streamType == "bidi-stream" {
		tests = append(tests, models.TestCase{
			Name:        fmt.Sprintf("gRPC %s.%s streaming endpoint accessible", serviceName, methodName),
			Description: "Verify streaming gRPC endpoint is reachable",
			Category:    models.CategoryHappyPath,
			Severity:    models.SeverityMedium,
			Request:     buildGRPCRequest(endpoint, map[string]any{}),
			Assertions: []models.Assertion{
				{Type: models.AssertStatusCode, Target: "status_code", Operator: "lt", Expected: 500},
			},
			Tags: []string{"grpc", "streaming", streamType},
		})
	}

	return tests, nil
}

func isGRPCEndpoint(ep models.Endpoint) bool {
	if ep.Method == "GRPC" {
		return true
	}
	for _, tag := range ep.Tags {
		if tag == "grpc" {
			return true
		}
	}
	return false
}

func buildGRPCRequest(ep models.Endpoint, payload map[string]any) models.TestRequest {
	body, _ := json.Marshal(payload)
	headers := map[string]string{
		"Content-Type": "application/json",
	}

	// Copy auth headers if present.
	if ep.Auth != nil && ep.Auth.Location == "header" && ep.Auth.Key != "" {
		headers[ep.Auth.Key] = "{{auth_token}}"
	}

	return models.TestRequest{
		Method:  "POST",
		URL:     ep.FullURL(),
		Headers: headers,
		Body:    string(body),
		Timeout: 10 * time.Second,
	}
}

func generateLargeString(size int) string {
	b := make([]byte, size)
	for i := range b {
		b[i] = 'A' + byte(i%26)
	}
	return string(b)
}
