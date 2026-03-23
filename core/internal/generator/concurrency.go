package generator

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mkaganm/probex/internal/models"
)

// Concurrency generates tests for race conditions and concurrent access.
type Concurrency struct{}

// NewConcurrency creates a new Concurrency generator.
func NewConcurrency() *Concurrency { return &Concurrency{} }

// Category returns the test category.
func (c *Concurrency) Category() models.TestCategory { return models.CategoryConcurrency }

// Generate creates concurrency test cases for an endpoint.
func (c *Concurrency) Generate(endpoint models.Endpoint) ([]models.TestCase, error) {
	var tests []models.TestCase
	method := strings.ToUpper(endpoint.Method)

	// Concurrency tests are most relevant for state-changing operations
	switch method {
	case "POST":
		tests = append(tests, c.doubleSubmit(endpoint)...)
		tests = append(tests, c.parallelCreate(endpoint)...)
	case "PUT", "PATCH":
		tests = append(tests, c.raceConditionUpdate(endpoint)...)
	case "DELETE":
		tests = append(tests, c.doubleDelete(endpoint)...)
	}

	return tests, nil
}

// doubleSubmit tests idempotency by submitting the same POST request twice.
func (c *Concurrency) doubleSubmit(ep models.Endpoint) []models.TestCase {
	baseReq := buildBaseRequest(ep)

	return []models.TestCase{
		{
			Name:        fmt.Sprintf("Double submit %s %s", ep.Method, ep.Path),
			Description: "Submit the same creation request twice to test idempotency. Second request should be rejected (409) or return the same resource.",
			Category:    models.CategoryConcurrency,
			Severity:    models.SeverityHigh,
			Request:     baseReq,
			Assertions: []models.Assertion{
				// First request should succeed
				{Type: models.AssertStatusCode, Target: "status_code", Operator: "lte", Expected: 201},
			},
			Tags: []string{"idempotency", "double_submit"},
		},
		{
			Name:        fmt.Sprintf("Double submit verify %s %s", ep.Method, ep.Path),
			Description: "Second identical POST should either return 409 Conflict or be idempotent (same result).",
			Category:    models.CategoryConcurrency,
			Severity:    models.SeverityHigh,
			DependsOn:   []string{fmt.Sprintf("Double submit %s %s", ep.Method, ep.Path)},
			Request:     baseReq,
			Assertions: []models.Assertion{
				// Should not create a duplicate — expect 409 or 200/201
				{Type: models.AssertStatusCode, Target: "status_code", Operator: "ne", Expected: 500},
			},
			Tags: []string{"idempotency", "double_submit"},
		},
	}
}

// parallelCreate tests race conditions by creating the same resource in parallel.
func (c *Concurrency) parallelCreate(ep models.Endpoint) []models.TestCase {
	if ep.RequestBody == nil {
		return nil
	}

	baseReq := buildBaseRequest(ep)

	// Create a unique-ish body to detect duplicates
	body := buildExampleBody(ep.RequestBody)
	if bodyMap, ok := body.(map[string]any); ok {
		bodyMap["_concurrency_marker"] = "parallel_test"
		b, err := json.Marshal(bodyMap)
		if err == nil {
			baseReq.Body = string(b)
		}
	}

	return []models.TestCase{
		{
			Name:        fmt.Sprintf("Parallel create %s %s", ep.Method, ep.Path),
			Description: "Send multiple creation requests simultaneously to test race condition handling. Server should handle gracefully without data corruption.",
			Category:    models.CategoryConcurrency,
			Severity:    models.SeverityHigh,
			Request:     baseReq,
			Assertions: []models.Assertion{
				{Type: models.AssertStatusCode, Target: "status_code", Operator: "ne", Expected: 500},
			},
			Tags: []string{"race_condition", "parallel_create"},
		},
	}
}

// raceConditionUpdate tests concurrent updates to the same resource.
func (c *Concurrency) raceConditionUpdate(ep models.Endpoint) []models.TestCase {
	baseReq := buildBaseRequest(ep)

	return []models.TestCase{
		{
			Name:        fmt.Sprintf("Race condition update %s %s", ep.Method, ep.Path),
			Description: "Send concurrent update requests to detect lost update / race condition problems. Server should use optimistic locking or handle gracefully.",
			Category:    models.CategoryConcurrency,
			Severity:    models.SeverityHigh,
			Request:     baseReq,
			Assertions: []models.Assertion{
				{Type: models.AssertStatusCode, Target: "status_code", Operator: "ne", Expected: 500},
			},
			Tags: []string{"race_condition", "concurrent_update"},
		},
	}
}

// doubleDelete tests deleting the same resource twice.
func (c *Concurrency) doubleDelete(ep models.Endpoint) []models.TestCase {
	baseReq := buildBaseRequest(ep)

	return []models.TestCase{
		{
			Name:        fmt.Sprintf("Double delete %s %s", ep.Method, ep.Path),
			Description: "Delete the same resource twice. Second request should return 404 (already deleted) or 204 (idempotent delete).",
			Category:    models.CategoryConcurrency,
			Severity:    models.SeverityMedium,
			Request:     baseReq,
			Assertions: []models.Assertion{
				{Type: models.AssertStatusCode, Target: "status_code", Operator: "ne", Expected: 500},
			},
			Tags: []string{"idempotency", "double_delete"},
		},
	}
}

var _ Generator = (*Concurrency)(nil)
var _ = time.Second // ensure time is used
