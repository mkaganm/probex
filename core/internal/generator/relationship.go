package generator

import (
	"fmt"
	"strings"

	"github.com/mkaganm/probex/internal/models"
)

// Relationship generates tests for inter-endpoint dependencies.
type Relationship struct {
	allEndpoints []models.Endpoint
}

// NewRelationship creates a new Relationship generator.
func NewRelationship() *Relationship { return &Relationship{} }

// Category returns the test category.
func (r *Relationship) Category() models.TestCategory { return models.CategoryRelation }

// SetEndpoints provides the full endpoint list for relationship analysis.
func (r *Relationship) SetEndpoints(eps []models.Endpoint) {
	r.allEndpoints = eps
}

// Generate creates relationship test cases for an endpoint.
func (r *Relationship) Generate(endpoint models.Endpoint) ([]models.TestCase, error) {
	var tests []models.TestCase
	method := strings.ToUpper(endpoint.Method)

	// Only generate relationship tests for resource-creating or deleting endpoints
	switch method {
	case "POST":
		tests = append(tests, r.generateCRUDCycle(endpoint)...)
	case "DELETE":
		tests = append(tests, r.generateDeleteVerify(endpoint)...)
	}

	return tests, nil
}

// generateCRUDCycle generates a create → read → update → delete → verify-deleted test chain.
func (r *Relationship) generateCRUDCycle(createEp models.Endpoint) []models.TestCase {
	var tests []models.TestCase

	// Find corresponding GET, PUT, DELETE endpoints for the same resource
	resource := extractResource(createEp.Path)
	if resource == "" {
		return tests
	}

	var getEp, putEp, deleteEp *models.Endpoint
	for i := range r.allEndpoints {
		ep := &r.allEndpoints[i]
		epResource := extractResource(ep.Path)
		if epResource != resource {
			continue
		}
		switch strings.ToUpper(ep.Method) {
		case "GET":
			if hasIDParam(ep.Path) {
				getEp = ep
			}
		case "PUT", "PATCH":
			putEp = ep
		case "DELETE":
			deleteEp = ep
		}
	}

	baseReq := buildBaseRequest(createEp)

	// Step 1: Create
	createTest := models.TestCase{
		Name:        fmt.Sprintf("CRUD create %s", resource),
		Description: fmt.Sprintf("Create a new %s resource", resource),
		Category:    models.CategoryRelation,
		Severity:    models.SeverityHigh,
		Request:     baseReq,
		Assertions: []models.Assertion{
			{Type: models.AssertStatusCode, Target: "status_code", Operator: "eq", Expected: 201},
		},
		Tags: []string{"crud", "create", resource},
	}
	tests = append(tests, createTest)

	// Step 2: Read (if GET endpoint exists)
	if getEp != nil {
		readReq := buildBaseRequest(*getEp)
		tests = append(tests, models.TestCase{
			Name:        fmt.Sprintf("CRUD read %s after create", resource),
			Description: fmt.Sprintf("Read the created %s resource to verify it exists", resource),
			Category:    models.CategoryRelation,
			Severity:    models.SeverityHigh,
			DependsOn:   []string{createTest.Name},
			Request:     readReq,
			Assertions: []models.Assertion{
				{Type: models.AssertStatusCode, Target: "status_code", Operator: "eq", Expected: 200},
			},
			Tags: []string{"crud", "read", resource},
		})
	}

	// Step 3: Update (if PUT/PATCH endpoint exists)
	if putEp != nil {
		updateReq := buildBaseRequest(*putEp)
		tests = append(tests, models.TestCase{
			Name:        fmt.Sprintf("CRUD update %s", resource),
			Description: fmt.Sprintf("Update the created %s resource", resource),
			Category:    models.CategoryRelation,
			Severity:    models.SeverityMedium,
			DependsOn:   []string{createTest.Name},
			Request:     updateReq,
			Assertions: []models.Assertion{
				{Type: models.AssertStatusCode, Target: "status_code", Operator: "lte", Expected: 204},
			},
			Tags: []string{"crud", "update", resource},
		})
	}

	// Step 4: Delete (if DELETE endpoint exists)
	if deleteEp != nil {
		deleteReq := buildBaseRequest(*deleteEp)
		deleteTestName := fmt.Sprintf("CRUD delete %s", resource)
		tests = append(tests, models.TestCase{
			Name:        deleteTestName,
			Description: fmt.Sprintf("Delete the created %s resource", resource),
			Category:    models.CategoryRelation,
			Severity:    models.SeverityHigh,
			DependsOn:   []string{createTest.Name},
			Request:     deleteReq,
			Assertions: []models.Assertion{
				{Type: models.AssertStatusCode, Target: "status_code", Operator: "lte", Expected: 204},
			},
			Tags: []string{"crud", "delete", resource},
		})

		// Step 5: Verify deleted (GET should 404)
		if getEp != nil {
			verifyReq := buildBaseRequest(*getEp)
			tests = append(tests, models.TestCase{
				Name:        fmt.Sprintf("CRUD verify deleted %s", resource),
				Description: fmt.Sprintf("Verify the deleted %s returns 404", resource),
				Category:    models.CategoryRelation,
				Severity:    models.SeverityHigh,
				DependsOn:   []string{deleteTestName},
				Request:     verifyReq,
				Assertions: []models.Assertion{
					{Type: models.AssertStatusCode, Target: "status_code", Operator: "eq", Expected: 404},
				},
				Tags: []string{"crud", "verify_deleted", resource},
			})
		}
	}

	return tests
}

// generateDeleteVerify generates cascade/referential integrity tests for DELETE.
func (r *Relationship) generateDeleteVerify(deleteEp models.Endpoint) []models.TestCase {
	var tests []models.TestCase

	resource := extractResource(deleteEp.Path)
	if resource == "" {
		return tests
	}

	// Find child resources that reference this resource
	for _, ep := range r.allEndpoints {
		if strings.ToUpper(ep.Method) != "GET" {
			continue
		}
		// Check if endpoint path contains the parent resource ID pattern
		// e.g., /users/{userId}/orders
		if strings.Contains(ep.Path, resource) && ep.Path != deleteEp.Path && hasIDParam(ep.Path) {
			childResource := extractResource(ep.Path)
			readReq := buildBaseRequest(ep)
			tests = append(tests, models.TestCase{
				Name:        fmt.Sprintf("Cascade check: %s after %s delete", childResource, resource),
				Description: fmt.Sprintf("Check what happens to %s when parent %s is deleted", childResource, resource),
				Category:    models.CategoryRelation,
				Severity:    models.SeverityMedium,
				DependsOn:   []string{fmt.Sprintf("CRUD delete %s", resource)},
				Request:     readReq,
				Assertions: []models.Assertion{
					// Either 404 (cascade delete) or 200 (orphaned) — both are valid behaviors
					// We just verify no 500
					{Type: models.AssertStatusCode, Target: "status_code", Operator: "ne", Expected: 500},
				},
				Tags: []string{"cascade", childResource, resource},
			})
		}
	}

	return tests
}

// extractResource extracts the resource name from a path (e.g., /users/{id} → users).
func extractResource(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for _, p := range parts {
		if p != "" && !strings.HasPrefix(p, "{") && !strings.HasPrefix(p, ":") {
			return p
		}
	}
	return ""
}

// hasIDParam checks if a path contains an ID parameter.
func hasIDParam(path string) bool {
	return strings.Contains(path, "{") || strings.Contains(path, "/:") ||
		// Patterns like /users/1 — paths ending with a non-word segment
		endsWithIDSegment(path)
}

func endsWithIDSegment(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 {
		return false
	}
	last := parts[len(parts)-1]
	return strings.HasPrefix(last, "{") || strings.HasPrefix(last, ":")
}

// init is not needed; relationship generator is added to Engine separately
// since it needs all endpoints set via SetEndpoints
var _ Generator = (*Relationship)(nil)
