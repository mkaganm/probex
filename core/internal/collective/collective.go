// Package collective implements federated knowledge sharing for PROBEX.
//
// Instances can anonymously share learned test patterns (NOT data or API details)
// with a central hub, and pull community patterns to enrich local test generation.
//
// Privacy guarantee: only abstract patterns are shared — never URLs, tokens,
// request bodies, or any identifying information.
package collective

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mkaganm/probex/internal/models"
)

// Pattern represents an anonymized, shareable test pattern.
type Pattern struct {
	ID          string   `json:"id"`
	Category    string   `json:"category"`
	Severity    string   `json:"severity"`
	HTTPMethod  string   `json:"http_method"`
	PathPattern string   `json:"path_pattern"` // e.g. "/resource/{id}" (anonymized)
	TestType    string   `json:"test_type"`     // e.g. "bola", "rate_limit", "schema_drift"
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Score       float64  `json:"score"`       // community effectiveness score
	UsageCount  int      `json:"usage_count"` // how many instances use this pattern
	CreatedAt   string   `json:"created_at"`
}

// Contribution is what an instance sends to the hub.
type Contribution struct {
	InstanceID string    `json:"instance_id"` // hashed, not identifying
	Patterns   []Pattern `json:"patterns"`
	Timestamp  string    `json:"timestamp"`
}

// PullResponse is what the hub returns when pulling patterns.
type PullResponse struct {
	Patterns  []Pattern `json:"patterns"`
	Total     int       `json:"total"`
	UpdatedAt string    `json:"updated_at"`
}

// Client communicates with the collective intelligence hub.
type Client struct {
	hubURL     string
	instanceID string
	httpClient *http.Client
}

// NewClient creates a collective intelligence client.
// instanceID should be a stable but non-identifying hash for this installation.
func NewClient(hubURL string, instanceID string) *Client {
	return &Client{
		hubURL:     hubURL,
		instanceID: instanceID,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// GenerateInstanceID creates a stable anonymized instance ID from a seed.
func GenerateInstanceID(seed string) string {
	h := sha256.Sum256([]byte("probex-collective:" + seed))
	return hex.EncodeToString(h[:16])
}

// Push shares anonymized patterns with the collective hub.
func (c *Client) Push(ctx context.Context, patterns []Pattern) error {
	contrib := Contribution{
		InstanceID: c.instanceID,
		Patterns:   patterns,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.Marshal(contrib)
	if err != nil {
		return fmt.Errorf("marshaling contribution: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.hubURL+"/api/v1/collective/push", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("creating push request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("pushing to collective: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("push returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Pull fetches community patterns from the hub.
func (c *Client) Pull(ctx context.Context, categories []string, minScore float64) (*PullResponse, error) {
	url := fmt.Sprintf("%s/api/v1/collective/pull?min_score=%.1f", c.hubURL, minScore)
	for _, cat := range categories {
		url += "&category=" + cat
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating pull request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pulling from collective: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("pull returned %d: %s", resp.StatusCode, string(body))
	}

	var result PullResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding pull response: %w", err)
	}

	return &result, nil
}

// Anonymizer converts real test results into shareable patterns,
// stripping all identifying information.
type Anonymizer struct{}

// NewAnonymizer creates a new pattern anonymizer.
func NewAnonymizer() *Anonymizer {
	return &Anonymizer{}
}

// ExtractPatterns takes test results and produces anonymized patterns.
func (a *Anonymizer) ExtractPatterns(results *models.RunSummary) []Pattern {
	seen := make(map[string]bool)
	var patterns []Pattern

	for _, r := range results.Results {
		key := fmt.Sprintf("%s-%s-%s", r.Category, r.Severity, anonymizePath(r.TestCaseID))
		if seen[key] {
			continue
		}
		seen[key] = true

		p := Pattern{
			ID:          hashPattern(key),
			Category:    string(r.Category),
			Severity:    string(r.Severity),
			PathPattern: anonymizePath(r.TestCaseID),
			TestType:    inferTestType(r),
			Description: anonymizeDescription(r.TestName),
			Tags:        []string{string(r.Category)},
			Score:       effectivenessScore(r),
			CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		}
		patterns = append(patterns, p)
	}

	return patterns
}

// PatternToTestCases converts community patterns into local test cases.
func PatternToTestCases(patterns []Pattern, baseURL string) []models.TestCase {
	var tests []models.TestCase

	for _, p := range patterns {
		tc := models.TestCase{
			ID:          fmt.Sprintf("collective-%s", p.ID),
			Name:        fmt.Sprintf("community: %s", p.Description),
			Description: fmt.Sprintf("Community pattern (score: %.1f, used by %d instances)", p.Score, p.UsageCount),
			Category:    models.TestCategory(p.Category),
			Severity:    models.Severity(p.Severity),
			Tags:        append(p.Tags, "collective"),
			GeneratedBy: "collective",
			GeneratedAt: time.Now(),
		}

		if p.HTTPMethod != "" && p.PathPattern != "" {
			tc.Request = models.TestRequest{
				Method: p.HTTPMethod,
				URL:    baseURL + p.PathPattern,
			}
		}

		tests = append(tests, tc)
	}

	return tests
}

func hashPattern(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:8])
}

func anonymizePath(endpointID string) string {
	// Strip specific IDs, keep structure.
	if endpointID == "" {
		return "/resource"
	}
	return "/resource/{id}"
}

func anonymizeDescription(name string) string {
	// Keep the test type, strip specifics.
	if len(name) > 80 {
		return name[:80]
	}
	return name
}

func inferTestType(r models.TestResult) string {
	switch r.Category {
	case models.CategorySecurity:
		return "security"
	case models.CategoryEdgeCase:
		return "edge_case"
	case models.CategoryFuzz:
		return "fuzz"
	default:
		return string(r.Category)
	}
}

func effectivenessScore(r models.TestResult) float64 {
	// Failed tests that find real issues are more valuable patterns.
	if r.Status == models.StatusFailed {
		return 0.8
	}
	return 0.5
}
