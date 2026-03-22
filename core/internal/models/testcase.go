package models

import "time"

// TestCase represents a single generated test case.
type TestCase struct {
	ID          string       `json:"id" yaml:"id"`
	Name        string       `json:"name" yaml:"name"`
	Description string       `json:"description,omitempty" yaml:"description,omitempty"`
	Category    TestCategory `json:"category" yaml:"category"`
	Severity    Severity     `json:"severity" yaml:"severity"`
	EndpointID  string       `json:"endpoint_id" yaml:"endpoint_id"`
	Request     TestRequest  `json:"request" yaml:"request"`
	Assertions  []Assertion  `json:"assertions" yaml:"assertions"`
	DependsOn   []string     `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
	Tags        []string     `json:"tags,omitempty" yaml:"tags,omitempty"`
	GeneratedBy string       `json:"generated_by" yaml:"generated_by"`
	GeneratedAt time.Time    `json:"generated_at" yaml:"generated_at"`
}

// TestRequest defines the HTTP request for a test case.
type TestRequest struct {
	Method  string            `json:"method" yaml:"method"`
	URL     string            `json:"url" yaml:"url"`
	Headers map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	Body    string            `json:"body,omitempty" yaml:"body,omitempty"`
	Timeout time.Duration     `json:"timeout,omitempty" yaml:"timeout,omitempty"`
}

// Assertion defines an expected condition on the response.
type Assertion struct {
	Type     AssertionType `json:"type" yaml:"type"`
	Target   string        `json:"target" yaml:"target"`     // JSONPath, header name, etc.
	Operator string        `json:"operator" yaml:"operator"` // eq, ne, gt, lt, contains, matches, exists
	Expected any           `json:"expected" yaml:"expected"`
}

// TestCategory classifies the type of test.
type TestCategory string

const (
	CategoryHappyPath   TestCategory = "happy_path"
	CategoryEdgeCase    TestCategory = "edge_case"
	CategorySecurity    TestCategory = "security"
	CategoryFuzz        TestCategory = "fuzz"
	CategoryRelation    TestCategory = "relationship"
	CategoryConcurrency TestCategory = "concurrency"
	CategoryPerformance TestCategory = "performance"
)

// Severity indicates the importance level of a test finding.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
)

// AssertionType defines what part of the response to assert on.
type AssertionType string

const (
	AssertStatusCode   AssertionType = "status_code"
	AssertBody         AssertionType = "body"
	AssertHeader       AssertionType = "header"
	AssertSchema       AssertionType = "schema"
	AssertResponseTime AssertionType = "response_time"
)
