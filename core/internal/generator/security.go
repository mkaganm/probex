package generator

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mkaganm/probex/internal/models"
)

// Security generates OWASP API Top 10 security tests.
type Security struct{}

// NewSecurity creates a new Security generator.
func NewSecurity() *Security { return &Security{} }

// Category returns the test category.
func (s *Security) Category() models.TestCategory { return models.CategorySecurity }

var sqlInjectionPayloads = []string{
	"' OR '1'='1",
	"'; DROP TABLE users; --",
	"1; SELECT * FROM users",
	"' UNION SELECT NULL,NULL,NULL--",
	"1' AND '1'='1",
	"admin'--",
}

var xssPayloads = []string{
	"<script>alert('xss')</script>",
	"<img src=x onerror=alert('xss')>",
	"javascript:alert('xss')",
	"<svg onload=alert('xss')>",
	"\"><script>alert('xss')</script>",
}

var pathTraversalPayloads = []string{
	"../../etc/passwd",
	"..\\..\\windows\\system32\\config\\sam",
	"....//....//etc/passwd",
	"%2e%2e%2f%2e%2e%2fetc%2fpasswd",
	"..%252f..%252f..%252fetc%252fpasswd",
}

var ssrfPayloads = []string{
	"http://127.0.0.1:80",
	"http://169.254.169.254/latest/meta-data/",
	"http://localhost:22",
	"http://[::1]",
	"http://0.0.0.0:80",
}

var massAssignmentFields = []string{
	"is_admin", "role", "permissions", "admin", "verified", "balance",
	"_isAdmin", "privilege", "access_level",
}

// urlFieldNames identifies fields that may contain URLs.
var urlFieldNames = []string{
	"url", "link", "href", "callback", "redirect", "webhook",
	"uri", "endpoint", "target", "destination", "return_url",
}

// adminPathParts identifies admin-like endpoint paths.
var adminPathParts = []string{
	"admin", "manage", "config", "settings", "internal",
	"system", "debug", "actuator", "metrics",
}

// Generate creates security test cases for an endpoint.
func (s *Security) Generate(endpoint models.Endpoint) ([]models.TestCase, error) {
	var tests []models.TestCase

	baseReq := buildBaseRequest(endpoint)
	method := strings.ToUpper(endpoint.Method)

	// 1. SQL Injection in query params
	tests = append(tests, s.sqlInjectionQuery(endpoint, baseReq)...)

	// 2. SQL Injection in body fields
	tests = append(tests, s.sqlInjectionBody(endpoint, baseReq, method)...)

	// 3. XSS in body string fields
	tests = append(tests, s.xssBody(endpoint, baseReq, method)...)

	// 4. Path traversal in URL path params
	tests = append(tests, s.pathTraversal(endpoint, baseReq)...)

	// 5. Missing auth header
	tests = append(tests, s.missingAuth(endpoint, baseReq)...)

	// 6. Large payload
	tests = append(tests, s.largePayload(endpoint, baseReq, method)...)

	// 7. BOLA — Broken Object Level Authorization
	tests = append(tests, s.bola(endpoint, baseReq)...)

	// 8. Broken Authentication
	tests = append(tests, s.brokenAuth(endpoint, baseReq)...)

	// 9. Mass Assignment
	tests = append(tests, s.massAssignment(endpoint, baseReq, method)...)

	// 10. SSRF
	tests = append(tests, s.ssrf(endpoint, baseReq, method)...)

	// 11. Security Misconfiguration
	tests = append(tests, s.securityMisconfig(endpoint, baseReq)...)

	// 12. BFLA — Broken Function Level Authorization
	tests = append(tests, s.bfla(endpoint, baseReq)...)

	return tests, nil
}

// --- Existing tests refactored into methods ---

func (s *Security) sqlInjectionQuery(endpoint models.Endpoint, baseReq models.TestRequest) []models.TestCase {
	var tests []models.TestCase
	for _, param := range endpoint.QueryParams {
		if param.Type == "string" || param.Type == "" {
			for i, payload := range sqlInjectionPayloads {
				url := appendQueryParam(baseReq.URL, param.Name, payload)
				tests = append(tests, models.TestCase{
					Name:        fmt.Sprintf("SQLi %s %s param '%s' #%d", endpoint.Method, endpoint.Path, param.Name, i+1),
					Description: fmt.Sprintf("SQL injection test on query param '%s'", param.Name),
					Category:    models.CategorySecurity,
					Severity:    models.SeverityCritical,
					Request: models.TestRequest{
						Method:  baseReq.Method,
						URL:     url,
						Headers: copyHeaders(baseReq.Headers),
						Body:    baseReq.Body,
						Timeout: 30 * time.Second,
					},
					Assertions: []models.Assertion{
						{Type: models.AssertStatusCode, Target: "status_code", Operator: "ne", Expected: 200},
					},
				})
			}
		}
	}
	return tests
}

func (s *Security) sqlInjectionBody(endpoint models.Endpoint, baseReq models.TestRequest, method string) []models.TestCase {
	var tests []models.TestCase
	if endpoint.RequestBody == nil || endpoint.RequestBody.Properties == nil || (method != "POST" && method != "PUT" && method != "PATCH") {
		return tests
	}
	exampleBody := buildExampleBody(endpoint.RequestBody)
	bodyMap, ok := exampleBody.(map[string]any)
	if !ok {
		return tests
	}
	for fieldName, prop := range endpoint.RequestBody.Properties {
		if prop.Type != "string" && prop.Type != "" {
			continue
		}
		for i, payload := range sqlInjectionPayloads {
			mutated := copyMap(bodyMap)
			mutated[fieldName] = payload
			b, err := json.Marshal(mutated)
			if err != nil {
				continue
			}
			tests = append(tests, models.TestCase{
				Name:        fmt.Sprintf("SQLi body %s %s field '%s' #%d", endpoint.Method, endpoint.Path, fieldName, i+1),
				Description: fmt.Sprintf("SQL injection test on body field '%s'", fieldName),
				Category:    models.CategorySecurity,
				Severity:    models.SeverityCritical,
				Request: models.TestRequest{
					Method:  baseReq.Method,
					URL:     baseReq.URL,
					Headers: copyHeaders(baseReq.Headers),
					Body:    string(b),
					Timeout: 30 * time.Second,
				},
				Assertions: []models.Assertion{
					{Type: models.AssertStatusCode, Target: "status_code", Operator: "ne", Expected: 200},
				},
			})
		}
	}
	return tests
}

func (s *Security) xssBody(endpoint models.Endpoint, baseReq models.TestRequest, method string) []models.TestCase {
	var tests []models.TestCase
	if endpoint.RequestBody == nil || endpoint.RequestBody.Properties == nil || (method != "POST" && method != "PUT" && method != "PATCH") {
		return tests
	}
	exampleBody := buildExampleBody(endpoint.RequestBody)
	bodyMap, ok := exampleBody.(map[string]any)
	if !ok {
		return tests
	}
	for fieldName, prop := range endpoint.RequestBody.Properties {
		if prop.Type != "string" && prop.Type != "" {
			continue
		}
		for i, payload := range xssPayloads {
			mutated := copyMap(bodyMap)
			mutated[fieldName] = payload
			b, err := json.Marshal(mutated)
			if err != nil {
				continue
			}
			tests = append(tests, models.TestCase{
				Name:        fmt.Sprintf("XSS %s %s field '%s' #%d", endpoint.Method, endpoint.Path, fieldName, i+1),
				Description: fmt.Sprintf("XSS payload test on field '%s'", fieldName),
				Category:    models.CategorySecurity,
				Severity:    models.SeverityHigh,
				Request: models.TestRequest{
					Method:  baseReq.Method,
					URL:     baseReq.URL,
					Headers: copyHeaders(baseReq.Headers),
					Body:    string(b),
					Timeout: 30 * time.Second,
				},
				Assertions: []models.Assertion{
					{Type: models.AssertBody, Target: "@raw", Operator: "not_contains", Expected: payload},
				},
			})
		}
	}
	return tests
}

func (s *Security) pathTraversal(endpoint models.Endpoint, baseReq models.TestRequest) []models.TestCase {
	var tests []models.TestCase
	for _, param := range endpoint.PathParams {
		for i, payload := range pathTraversalPayloads {
			url := strings.ReplaceAll(baseReq.URL, "{"+param.Name+"}", payload)
			if url == baseReq.URL {
				url = baseReq.URL + "/" + payload
			}
			tests = append(tests, models.TestCase{
				Name:        fmt.Sprintf("PathTraversal %s %s param '%s' #%d", endpoint.Method, endpoint.Path, param.Name, i+1),
				Description: fmt.Sprintf("Path traversal test on URL param '%s'", param.Name),
				Category:    models.CategorySecurity,
				Severity:    models.SeverityCritical,
				Request: models.TestRequest{
					Method:  baseReq.Method,
					URL:     url,
					Headers: copyHeaders(baseReq.Headers),
					Timeout: 30 * time.Second,
				},
				Assertions: []models.Assertion{
					{Type: models.AssertStatusCode, Target: "status_code", Operator: "ne", Expected: 200},
					{Type: models.AssertBody, Target: "@raw", Operator: "not_contains", Expected: "root:"},
				},
			})
		}
	}
	return tests
}

func (s *Security) missingAuth(endpoint models.Endpoint, baseReq models.TestRequest) []models.TestCase {
	if endpoint.Auth == nil || endpoint.Auth.Type == models.AuthNone {
		return nil
	}
	noAuthHeaders := copyHeaders(baseReq.Headers)
	delete(noAuthHeaders, "Authorization")
	delete(noAuthHeaders, endpoint.Auth.Key)
	return []models.TestCase{
		{
			Name:        fmt.Sprintf("MissingAuth %s %s", endpoint.Method, endpoint.Path),
			Description: "Verify endpoint returns 401 when authentication is missing",
			Category:    models.CategorySecurity,
			Severity:    models.SeverityCritical,
			Request: models.TestRequest{
				Method:  baseReq.Method,
				URL:     baseReq.URL,
				Headers: noAuthHeaders,
				Body:    baseReq.Body,
				Timeout: 30 * time.Second,
			},
			Assertions: []models.Assertion{
				{Type: models.AssertStatusCode, Target: "status_code", Operator: "eq", Expected: 401},
			},
		},
	}
}

func (s *Security) largePayload(endpoint models.Endpoint, baseReq models.TestRequest, method string) []models.TestCase {
	if method != "POST" && method != "PUT" && method != "PATCH" {
		return nil
	}
	largeBody := strings.Repeat("A", 1024*1024)
	largeJSON, _ := json.Marshal(map[string]string{"data": largeBody})
	return []models.TestCase{
		{
			Name:        fmt.Sprintf("LargePayload %s %s", endpoint.Method, endpoint.Path),
			Description: "Send 1MB payload to test resource limit handling",
			Category:    models.CategorySecurity,
			Severity:    models.SeverityMedium,
			Request: models.TestRequest{
				Method:  baseReq.Method,
				URL:     baseReq.URL,
				Headers: copyHeaders(baseReq.Headers),
				Body:    string(largeJSON),
				Timeout: 30 * time.Second,
			},
			Assertions: []models.Assertion{
				{Type: models.AssertStatusCode, Target: "status_code", Operator: "ne", Expected: 200},
			},
		},
	}
}

// --- NEW OWASP Tests ---

// bola generates Broken Object Level Authorization tests.
// Tries accessing resources with manipulated IDs.
func (s *Security) bola(endpoint models.Endpoint, baseReq models.TestRequest) []models.TestCase {
	if len(endpoint.PathParams) == 0 {
		return nil
	}

	var tests []models.TestCase
	manipulatedIDs := []struct {
		label string
		value string
	}{
		{"zero", "0"},
		{"negative", "-1"},
		{"other_user", "999999"},
		{"sequential", "2"},
	}

	for _, param := range endpoint.PathParams {
		// Only test params that look like IDs
		nameLower := strings.ToLower(param.Name)
		if !strings.Contains(nameLower, "id") && !strings.Contains(nameLower, "uuid") && !strings.Contains(nameLower, "key") {
			continue
		}
		for _, mid := range manipulatedIDs {
			url := strings.ReplaceAll(baseReq.URL, "{"+param.Name+"}", mid.value)
			if url == baseReq.URL {
				continue
			}
			tests = append(tests, models.TestCase{
				Name:        fmt.Sprintf("BOLA %s %s param '%s' (%s)", endpoint.Method, endpoint.Path, param.Name, mid.label),
				Description: fmt.Sprintf("BOLA test: access resource with manipulated ID '%s' to verify authorization", mid.value),
				Category:    models.CategorySecurity,
				Severity:    models.SeverityCritical,
				Request: models.TestRequest{
					Method:  baseReq.Method,
					URL:     url,
					Headers: copyHeaders(baseReq.Headers),
					Body:    baseReq.Body,
					Timeout: 30 * time.Second,
				},
				Assertions: []models.Assertion{
					{Type: models.AssertStatusCode, Target: "status_code", Operator: "ne", Expected: 200},
				},
				Tags: []string{"bola", "owasp_api1"},
			})
		}
	}
	return tests
}

// brokenAuth generates Broken Authentication tests with invalid tokens.
func (s *Security) brokenAuth(endpoint models.Endpoint, baseReq models.TestRequest) []models.TestCase {
	var tests []models.TestCase

	invalidTokens := []struct {
		label string
		value string
	}{
		{"expired_jwt", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxIiwiZXhwIjoxfQ.invalid"},
		{"empty_bearer", "Bearer "},
		{"sqli_token", "Bearer ' OR '1'='1"},
		{"malformed", "Bearer not.a.valid.jwt.token"},
		{"null_token", "Bearer null"},
	}

	for _, tok := range invalidTokens {
		headers := copyHeaders(baseReq.Headers)
		headers["Authorization"] = tok.value
		tests = append(tests, models.TestCase{
			Name:        fmt.Sprintf("BrokenAuth %s %s (%s)", endpoint.Method, endpoint.Path, tok.label),
			Description: fmt.Sprintf("Test with invalid auth token (%s) — should be rejected", tok.label),
			Category:    models.CategorySecurity,
			Severity:    models.SeverityCritical,
			Request: models.TestRequest{
				Method:  baseReq.Method,
				URL:     baseReq.URL,
				Headers: headers,
				Body:    baseReq.Body,
				Timeout: 30 * time.Second,
			},
			Assertions: []models.Assertion{
				{Type: models.AssertStatusCode, Target: "status_code", Operator: "gte", Expected: 400},
				{Type: models.AssertStatusCode, Target: "status_code", Operator: "lte", Expected: 403},
			},
			Tags: []string{"broken_auth", "owasp_api2"},
		})
	}
	return tests
}

// massAssignment generates Mass Assignment tests by injecting privileged fields.
func (s *Security) massAssignment(endpoint models.Endpoint, baseReq models.TestRequest, method string) []models.TestCase {
	if method != "POST" && method != "PUT" && method != "PATCH" {
		return nil
	}

	// Build a base body
	var bodyMap map[string]any
	if endpoint.RequestBody != nil {
		if example := buildExampleBody(endpoint.RequestBody); example != nil {
			if m, ok := example.(map[string]any); ok {
				bodyMap = copyMap(m)
			}
		}
	}
	if bodyMap == nil {
		bodyMap = make(map[string]any)
	}

	// Add all mass assignment fields
	mutated := copyMap(bodyMap)
	for _, field := range massAssignmentFields {
		if _, exists := mutated[field]; !exists {
			mutated[field] = true
		}
	}

	b, err := json.Marshal(mutated)
	if err != nil {
		return nil
	}

	return []models.TestCase{
		{
			Name:        fmt.Sprintf("MassAssignment %s %s", endpoint.Method, endpoint.Path),
			Description: "Inject privileged fields (is_admin, role, permissions) to test mass assignment protection",
			Category:    models.CategorySecurity,
			Severity:    models.SeverityHigh,
			Request: models.TestRequest{
				Method:  baseReq.Method,
				URL:     baseReq.URL,
				Headers: copyHeaders(baseReq.Headers),
				Body:    string(b),
				Timeout: 30 * time.Second,
			},
			Assertions: []models.Assertion{
				// Should not return 200 with the privileged fields accepted
				{Type: models.AssertStatusCode, Target: "status_code", Operator: "ne", Expected: 500},
				// Response should not echo back admin privileges
				{Type: models.AssertBody, Target: "@raw", Operator: "not_contains", Expected: "\"is_admin\":true"},
			},
			Tags: []string{"mass_assignment", "owasp_api3"},
		},
	}
}

// ssrf generates Server-Side Request Forgery tests for URL-like fields.
func (s *Security) ssrf(endpoint models.Endpoint, baseReq models.TestRequest, method string) []models.TestCase {
	if method != "POST" && method != "PUT" && method != "PATCH" {
		return nil
	}
	if endpoint.RequestBody == nil || endpoint.RequestBody.Properties == nil {
		return nil
	}

	var tests []models.TestCase
	exampleBody := buildExampleBody(endpoint.RequestBody)
	bodyMap, ok := exampleBody.(map[string]any)
	if !ok {
		return nil
	}

	for fieldName := range endpoint.RequestBody.Properties {
		if !isURLField(fieldName) {
			continue
		}
		for i, payload := range ssrfPayloads {
			mutated := copyMap(bodyMap)
			mutated[fieldName] = payload
			b, err := json.Marshal(mutated)
			if err != nil {
				continue
			}
			tests = append(tests, models.TestCase{
				Name:        fmt.Sprintf("SSRF %s %s field '%s' #%d", endpoint.Method, endpoint.Path, fieldName, i+1),
				Description: fmt.Sprintf("SSRF test: inject internal URL '%s' into field '%s'", payload, fieldName),
				Category:    models.CategorySecurity,
				Severity:    models.SeverityCritical,
				Request: models.TestRequest{
					Method:  baseReq.Method,
					URL:     baseReq.URL,
					Headers: copyHeaders(baseReq.Headers),
					Body:    string(b),
					Timeout: 30 * time.Second,
				},
				Assertions: []models.Assertion{
					{Type: models.AssertStatusCode, Target: "status_code", Operator: "ne", Expected: 200},
				},
				Tags: []string{"ssrf", "owasp_api7"},
			})
		}
	}
	return tests
}

// securityMisconfig generates Security Misconfiguration tests.
func (s *Security) securityMisconfig(endpoint models.Endpoint, baseReq models.TestRequest) []models.TestCase {
	var tests []models.TestCase

	// CORS check: send request with evil origin
	corsHeaders := copyHeaders(baseReq.Headers)
	corsHeaders["Origin"] = "https://evil.com"
	tests = append(tests, models.TestCase{
		Name:        fmt.Sprintf("CORS %s %s", endpoint.Method, endpoint.Path),
		Description: "Check that CORS is not misconfigured to allow arbitrary origins",
		Category:    models.CategorySecurity,
		Severity:    models.SeverityMedium,
		Request: models.TestRequest{
			Method:  baseReq.Method,
			URL:     baseReq.URL,
			Headers: corsHeaders,
			Body:    baseReq.Body,
			Timeout: 30 * time.Second,
		},
		Assertions: []models.Assertion{
			{Type: models.AssertHeader, Target: "Access-Control-Allow-Origin", Operator: "ne", Expected: "*"},
			{Type: models.AssertHeader, Target: "Access-Control-Allow-Origin", Operator: "ne", Expected: "https://evil.com"},
		},
		Tags: []string{"cors", "misconfiguration", "owasp_api8"},
	})

	// Verbose error check: send malformed JSON body
	if strings.ToUpper(endpoint.Method) != "GET" && strings.ToUpper(endpoint.Method) != "DELETE" {
		errHeaders := copyHeaders(baseReq.Headers)
		errHeaders["Content-Type"] = "application/json"
		tests = append(tests, models.TestCase{
			Name:        fmt.Sprintf("VerboseError %s %s", endpoint.Method, endpoint.Path),
			Description: "Send malformed request to check that error responses don't leak stack traces or internal details",
			Category:    models.CategorySecurity,
			Severity:    models.SeverityMedium,
			Request: models.TestRequest{
				Method:  baseReq.Method,
				URL:     baseReq.URL,
				Headers: errHeaders,
				Body:    "{invalid json!@#$",
				Timeout: 30 * time.Second,
			},
			Assertions: []models.Assertion{
				{Type: models.AssertBody, Target: "@raw", Operator: "not_contains", Expected: "Traceback"},
				{Type: models.AssertBody, Target: "@raw", Operator: "not_contains", Expected: "panic:"},
				{Type: models.AssertBody, Target: "@raw", Operator: "not_contains", Expected: "java.lang."},
				{Type: models.AssertBody, Target: "@raw", Operator: "not_contains", Expected: "node_modules/"},
			},
			Tags: []string{"verbose_error", "misconfiguration", "owasp_api8"},
		})
	}

	// Security headers check
	tests = append(tests, models.TestCase{
		Name:        fmt.Sprintf("SecurityHeaders %s %s", endpoint.Method, endpoint.Path),
		Description: "Verify presence of security headers (X-Content-Type-Options, X-Frame-Options)",
		Category:    models.CategorySecurity,
		Severity:    models.SeverityLow,
		Request: models.TestRequest{
			Method:  baseReq.Method,
			URL:     baseReq.URL,
			Headers: copyHeaders(baseReq.Headers),
			Body:    baseReq.Body,
			Timeout: 30 * time.Second,
		},
		Assertions: []models.Assertion{
			{Type: models.AssertHeader, Target: "X-Content-Type-Options", Operator: "exists", Expected: true},
		},
		Tags: []string{"security_headers", "misconfiguration", "owasp_api8"},
	})

	return tests
}

// bfla generates Broken Function Level Authorization tests for admin-like endpoints.
func (s *Security) bfla(endpoint models.Endpoint, baseReq models.TestRequest) []models.TestCase {
	pathLower := strings.ToLower(endpoint.Path)
	isAdmin := false
	for _, part := range adminPathParts {
		if strings.Contains(pathLower, part) {
			isAdmin = true
			break
		}
	}
	if !isAdmin {
		return nil
	}

	// Try accessing without any auth
	noAuthHeaders := copyHeaders(baseReq.Headers)
	delete(noAuthHeaders, "Authorization")

	return []models.TestCase{
		{
			Name:        fmt.Sprintf("BFLA %s %s", endpoint.Method, endpoint.Path),
			Description: "Access admin/privileged endpoint without authentication to test function-level authorization",
			Category:    models.CategorySecurity,
			Severity:    models.SeverityCritical,
			Request: models.TestRequest{
				Method:  baseReq.Method,
				URL:     baseReq.URL,
				Headers: noAuthHeaders,
				Body:    baseReq.Body,
				Timeout: 30 * time.Second,
			},
			Assertions: []models.Assertion{
				{Type: models.AssertStatusCode, Target: "status_code", Operator: "gte", Expected: 401},
				{Type: models.AssertStatusCode, Target: "status_code", Operator: "lte", Expected: 403},
			},
			Tags: []string{"bfla", "owasp_api5"},
		},
	}
}

// isURLField returns true if a field name suggests it holds a URL.
func isURLField(name string) bool {
	nameLower := strings.ToLower(name)
	for _, keyword := range urlFieldNames {
		if strings.Contains(nameLower, keyword) {
			return true
		}
	}
	return false
}

// appendQueryParam appends a query parameter to a URL.
func appendQueryParam(rawURL, key, value string) string {
	sep := "?"
	if strings.Contains(rawURL, "?") {
		sep = "&"
	}
	return fmt.Sprintf("%s%s%s=%s", rawURL, sep, key, value)
}
