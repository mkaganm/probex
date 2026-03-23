package generator

import (
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/mkaganm/probex/internal/models"
)

// Engine orchestrates test generation from an API profile.
type Engine struct {
	profile        *models.APIProfile
	generators     []Generator
	categoryFilter map[models.TestCategory]bool
}

// Generator is the interface for test generators.
type Generator interface {
	Generate(endpoint models.Endpoint) ([]models.TestCase, error)
	Category() models.TestCategory
}

// New creates a new test generation Engine.
func New(profile *models.APIProfile) *Engine {
	rel := NewRelationship()
	rel.SetEndpoints(profile.Endpoints)

	e := &Engine{profile: profile}
	e.generators = []Generator{
		NewHappyPath(),
		NewEdgeCase(),
		NewSecurity(),
		NewFuzzer(),
		rel,
		NewConcurrency(),
		NewGraphQL(),
		NewWebSocket(),
		NewGRPC(),
	}
	return e
}

// Generate produces test cases for all endpoints.
func (e *Engine) Generate() ([]models.TestCase, error) {
	var tests []models.TestCase
	now := time.Now()

	for _, endpoint := range e.profile.Endpoints {
		for _, gen := range e.generators {
			if len(e.categoryFilter) > 0 && !e.categoryFilter[gen.Category()] {
				continue
			}
			cases, err := gen.Generate(endpoint)
			if err != nil {
				return nil, fmt.Errorf("generator %s failed for endpoint %s %s: %w",
					gen.Category(), endpoint.Method, endpoint.Path, err)
			}
			for i := range cases {
				cases[i].ID = generateTestID(endpoint, gen.Category(), cases[i].Name)
				cases[i].GeneratedBy = string(gen.Category())
				cases[i].GeneratedAt = now
				cases[i].EndpointID = endpoint.ID
			}
			tests = append(tests, cases...)
		}
	}
	return tests, nil
}

// SetCategoryFilter limits test generation to only the specified categories.
func (e *Engine) SetCategoryFilter(filter map[models.TestCategory]bool) {
	e.categoryFilter = filter
}

// generateTestID creates a deterministic test ID from the endpoint, category, and test name.
func generateTestID(ep models.Endpoint, cat models.TestCategory, name string) string {
	data := fmt.Sprintf("%s:%s:%s:%s:%s", ep.Method, ep.Path, ep.BaseURL, cat, name)
	h := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", h[:8])
}
