package ai

import (
	"fmt"
	"time"

	"github.com/probex/probex/internal/models"
)

// HealthResponse represents the Python brain health check response.
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	AIMode  string `json:"ai_mode"`
	Model   string `json:"model"`
}

// EndpointInfo is the AI-side representation of an API endpoint.
type EndpointInfo struct {
	Method      string          `json:"method"`
	Path        string          `json:"path"`
	BaseURL     string          `json:"base_url"`
	QueryParams []ParameterInfo `json:"query_params,omitempty"`
	PathParams  []ParameterInfo `json:"path_params,omitempty"`
	RequestBody *SchemaInfo     `json:"request_body,omitempty"`
	Auth        *AuthDetail     `json:"auth,omitempty"`
	Tags        []string        `json:"tags,omitempty"`
}

// ParameterInfo represents a request parameter for the AI service.
type ParameterInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
	Example  any    `json:"example,omitempty"`
}

// SchemaInfo represents a JSON schema for the AI service.
type SchemaInfo struct {
	Type       string                `json:"type"`
	Properties map[string]*SchemaInfo `json:"properties,omitempty"`
	Items      *SchemaInfo           `json:"items,omitempty"`
	Required   []string              `json:"required,omitempty"`
	Enum       []any                 `json:"enum,omitempty"`
	Pattern    string                `json:"pattern,omitempty"`
	Format     string                `json:"format,omitempty"`
}

// AuthDetail represents authentication info for the AI service.
type AuthDetail struct {
	Type     string `json:"type"`
	Location string `json:"location"`
	Key      string `json:"key"`
}

// --- Scenario Generation ---

// ScenarioRequest is the request body for POST /api/v1/scenarios/generate.
type ScenarioRequest struct {
	Endpoints      []EndpointInfo `json:"endpoints"`
	ProfileContext string         `json:"profile_context,omitempty"`
	MaxScenarios   int            `json:"max_scenarios"`
}

// ScenarioResponse is the response from POST /api/v1/scenarios/generate.
type ScenarioResponse struct {
	Scenarios  []GeneratedTestCase `json:"scenarios"`
	ModelUsed  string              `json:"model_used"`
	TokensUsed int                 `json:"tokens_used"`
}

// GeneratedTestCase represents an AI-generated test case.
type GeneratedTestCase struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Category    string          `json:"category"`
	Severity    string          `json:"severity"`
	Request     TestRequestInfo `json:"request"`
	Assertions  []AssertionInfo `json:"assertions"`
	Tags        []string        `json:"tags,omitempty"`
}

// TestRequestInfo describes the HTTP request for a generated test.
type TestRequestInfo struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
}

// AssertionInfo describes an assertion for a generated test.
type AssertionInfo struct {
	Type     string `json:"type"`
	Target   string `json:"target"`
	Operator string `json:"operator"`
	Expected any    `json:"expected"`
}

// --- Security Analysis ---

// SecurityAnalysisRequest is the request body for POST /api/v1/security/analyze.
type SecurityAnalysisRequest struct {
	Endpoints []EndpointInfo `json:"endpoints"`
	Depth     string         `json:"depth,omitempty"` // quick, standard, deep
}

// SecurityAnalysisResponse is the response from POST /api/v1/security/analyze.
type SecurityAnalysisResponse struct {
	Findings   []SecurityFinding `json:"findings"`
	ModelUsed  string            `json:"model_used"`
	TokensUsed int               `json:"tokens_used"`
}

// SecurityFinding represents a single security finding.
type SecurityFinding struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Category    string `json:"category"`
	Endpoint    string `json:"endpoint"`
	Evidence    string `json:"evidence,omitempty"`
	Remediation string `json:"remediation,omitempty"`
}

// --- NL to Test ---

// NLTestRequest is the request body for POST /api/v1/nl-to-test.
type NLTestRequest struct {
	Description string         `json:"description"`
	Endpoints   []EndpointInfo `json:"endpoints,omitempty"`
}

// NLTestResponse is the response from POST /api/v1/nl-to-test.
type NLTestResponse struct {
	TestCases  []GeneratedTestCase `json:"test_cases"`
	ModelUsed  string              `json:"model_used"`
	TokensUsed int                 `json:"tokens_used"`
}

// --- Anomaly Classification ---

// AnomalyClassifyRequest is the request body for POST /api/v1/anomaly/classify.
type AnomalyClassifyRequest struct {
	EndpointID     string `json:"endpoint_id"`
	ObservedStatus int    `json:"observed_status"`
	ExpectedStatus int    `json:"expected_status"`
	ResponseBody   string `json:"response_body,omitempty"`
	ResponseTimeMs int    `json:"response_time_ms,omitempty"`
	BaselineTimeMs int    `json:"baseline_time_ms,omitempty"`
}

// AnomalyClassifyResponse is the response from POST /api/v1/anomaly/classify.
type AnomalyClassifyResponse struct {
	Classification string  `json:"classification"`
	Confidence     float64 `json:"confidence"`
	Explanation    string  `json:"explanation"`
	Severity       string  `json:"severity"`
	ModelUsed      string  `json:"model_used"`
}

// --- Conversion helpers ---

// EndpointToInfo converts a models.Endpoint to an EndpointInfo for the AI service.
func EndpointToInfo(ep models.Endpoint) EndpointInfo {
	info := EndpointInfo{
		Method:  ep.Method,
		Path:    ep.Path,
		BaseURL: ep.BaseURL,
		Tags:    ep.Tags,
	}

	for _, p := range ep.QueryParams {
		info.QueryParams = append(info.QueryParams, ParameterInfo{
			Name:     p.Name,
			Type:     p.Type,
			Required: p.Required,
			Example:  p.Example,
		})
	}

	for _, p := range ep.PathParams {
		info.PathParams = append(info.PathParams, ParameterInfo{
			Name:     p.Name,
			Type:     p.Type,
			Required: p.Required,
			Example:  p.Example,
		})
	}

	if ep.RequestBody != nil {
		info.RequestBody = schemaToInfo(ep.RequestBody)
	}

	if ep.Auth != nil {
		info.Auth = &AuthDetail{
			Type:     string(ep.Auth.Type),
			Location: ep.Auth.Location,
			Key:      ep.Auth.Key,
		}
	}

	return info
}

// schemaToInfo converts a models.Schema to a SchemaInfo.
func schemaToInfo(s *models.Schema) *SchemaInfo {
	if s == nil {
		return nil
	}
	info := &SchemaInfo{
		Type:     s.Type,
		Required: s.Required,
		Enum:     s.Enum,
		Pattern:  s.Pattern,
		Format:   s.Format,
	}
	if s.Properties != nil {
		info.Properties = make(map[string]*SchemaInfo, len(s.Properties))
		for k, v := range s.Properties {
			info.Properties[k] = schemaToInfo(v)
		}
	}
	if s.Items != nil {
		info.Items = schemaToInfo(s.Items)
	}
	return info
}

// EndpointsToInfo converts a slice of models.Endpoint to a slice of EndpointInfo.
func EndpointsToInfo(endpoints []models.Endpoint) []EndpointInfo {
	infos := make([]EndpointInfo, 0, len(endpoints))
	for _, ep := range endpoints {
		infos = append(infos, EndpointToInfo(ep))
	}
	return infos
}

// GeneratedTestToModelTest converts an AI-generated test case to a models.TestCase.
func GeneratedTestToModelTest(g GeneratedTestCase, endpointID string) models.TestCase {
	tc := models.TestCase{
		ID:          fmt.Sprintf("ai-%s-%d", endpointID, time.Now().UnixNano()),
		Name:        g.Name,
		Description: g.Description,
		Category:    models.TestCategory(g.Category),
		Severity:    models.Severity(g.Severity),
		EndpointID:  endpointID,
		Request: models.TestRequest{
			Method:  g.Request.Method,
			URL:     g.Request.URL,
			Headers: g.Request.Headers,
			Body:    g.Request.Body,
		},
		Tags:        g.Tags,
		GeneratedBy: "ai-brain",
		GeneratedAt: time.Now(),
	}

	for _, a := range g.Assertions {
		tc.Assertions = append(tc.Assertions, models.Assertion{
			Type:     models.AssertionType(a.Type),
			Target:   a.Target,
			Operator: a.Operator,
			Expected: a.Expected,
		})
	}

	return tc
}

// GeneratedTestsToModelTests converts a slice of AI-generated test cases to models.TestCase slice.
func GeneratedTestsToModelTests(generated []GeneratedTestCase) []models.TestCase {
	tests := make([]models.TestCase, 0, len(generated))
	for i, g := range generated {
		tc := GeneratedTestToModelTest(g, fmt.Sprintf("ai-gen-%d", i))
		tests = append(tests, tc)
	}
	return tests
}
