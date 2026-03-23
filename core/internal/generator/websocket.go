package generator

import (
	"fmt"
	"time"

	"github.com/mkaganm/probex/internal/models"
)

// WebSocket generates tests specific to WebSocket endpoints.
type WebSocket struct{}

// NewWebSocket creates a new WebSocket test generator.
func NewWebSocket() *WebSocket { return &WebSocket{} }

// Category returns the test category.
func (w *WebSocket) Category() models.TestCategory { return models.CategoryHappyPath }

// Generate creates WebSocket-specific test cases.
func (w *WebSocket) Generate(endpoint models.Endpoint) ([]models.TestCase, error) {
	if !isWebSocketEndpoint(endpoint) {
		return nil, nil
	}

	var tests []models.TestCase

	wsURL := toWSURL(endpoint.FullURL())

	// 1. Connection handshake test — verify HTTP upgrade works.
	tests = append(tests, models.TestCase{
		Name:        fmt.Sprintf("WS %s upgrade handshake succeeds", endpoint.Path),
		Description: "Verify WebSocket upgrade returns 101 Switching Protocols",
		Category:    models.CategoryHappyPath,
		Severity:    models.SeverityHigh,
		Request: models.TestRequest{
			Method: "GET",
			URL:    endpoint.FullURL(),
			Headers: map[string]string{
				"Upgrade":               "websocket",
				"Connection":            "Upgrade",
				"Sec-WebSocket-Key":     "dGhlIHNhbXBsZSBub25jZQ==",
				"Sec-WebSocket-Version": "13",
			},
			Timeout: 10 * time.Second,
		},
		Assertions: []models.Assertion{
			{Type: models.AssertStatusCode, Target: "status_code", Operator: "eq", Expected: 101},
		},
		Tags: []string{"websocket", "handshake"},
	})

	// 2. Reject non-upgrade request.
	tests = append(tests, models.TestCase{
		Name:        fmt.Sprintf("WS %s rejects non-upgrade request", endpoint.Path),
		Description: "Normal GET without Upgrade header should not succeed as WebSocket",
		Category:    models.CategoryEdgeCase,
		Severity:    models.SeverityMedium,
		Request: models.TestRequest{
			Method: "GET",
			URL:    endpoint.FullURL(),
			Headers: map[string]string{
				"Accept": "application/json",
			},
			Timeout: 10 * time.Second,
		},
		Assertions: []models.Assertion{
			{Type: models.AssertStatusCode, Target: "status_code", Operator: "ne", Expected: 101},
		},
		Tags: []string{"websocket", "edge-case"},
	})

	// 3. Invalid WebSocket version.
	tests = append(tests, models.TestCase{
		Name:        fmt.Sprintf("WS %s rejects invalid protocol version", endpoint.Path),
		Description: "WebSocket endpoint should reject unsupported protocol versions",
		Category:    models.CategoryEdgeCase,
		Severity:    models.SeverityLow,
		Request: models.TestRequest{
			Method: "GET",
			URL:    endpoint.FullURL(),
			Headers: map[string]string{
				"Upgrade":               "websocket",
				"Connection":            "Upgrade",
				"Sec-WebSocket-Key":     "dGhlIHNhbXBsZSBub25jZQ==",
				"Sec-WebSocket-Version": "99",
			},
			Timeout: 10 * time.Second,
		},
		Assertions: []models.Assertion{
			{Type: models.AssertStatusCode, Target: "status_code", Operator: "ne", Expected: 101},
		},
		Tags: []string{"websocket", "edge-case"},
	})

	// 4. Missing Sec-WebSocket-Key.
	tests = append(tests, models.TestCase{
		Name:        fmt.Sprintf("WS %s requires Sec-WebSocket-Key", endpoint.Path),
		Description: "WebSocket upgrade should fail without required key header",
		Category:    models.CategorySecurity,
		Severity:    models.SeverityMedium,
		Request: models.TestRequest{
			Method: "GET",
			URL:    endpoint.FullURL(),
			Headers: map[string]string{
				"Upgrade":               "websocket",
				"Connection":            "Upgrade",
				"Sec-WebSocket-Version": "13",
			},
			Timeout: 10 * time.Second,
		},
		Assertions: []models.Assertion{
			{Type: models.AssertStatusCode, Target: "status_code", Operator: "ne", Expected: 101},
		},
		Tags: []string{"websocket", "security"},
	})

	// 5. Origin validation (CSWSH — Cross-Site WebSocket Hijacking).
	tests = append(tests, models.TestCase{
		Name:        fmt.Sprintf("WS %s validates Origin header", endpoint.Path),
		Description: "WebSocket endpoint should validate Origin to prevent CSWSH attacks",
		Category:    models.CategorySecurity,
		Severity:    models.SeverityCritical,
		Request: models.TestRequest{
			Method: "GET",
			URL:    endpoint.FullURL(),
			Headers: map[string]string{
				"Upgrade":               "websocket",
				"Connection":            "Upgrade",
				"Sec-WebSocket-Key":     "dGhlIHNhbXBsZSBub25jZQ==",
				"Sec-WebSocket-Version": "13",
				"Origin":                "https://evil-attacker.example.com",
			},
			Timeout: 10 * time.Second,
		},
		Assertions: []models.Assertion{
			{Type: models.AssertStatusCode, Target: "status_code", Operator: "ne", Expected: 101},
		},
		Tags: []string{"websocket", "security", "cswsh", "owasp"},
	})

	// 6. Auth required for WS (if endpoint has auth).
	if endpoint.Auth != nil && endpoint.Auth.Type != models.AuthNone {
		tests = append(tests, models.TestCase{
			Name:        fmt.Sprintf("WS %s requires authentication", endpoint.Path),
			Description: "WebSocket endpoint should reject unauthenticated connections",
			Category:    models.CategorySecurity,
			Severity:    models.SeverityHigh,
			Request: models.TestRequest{
				Method: "GET",
				URL:    endpoint.FullURL(),
				Headers: map[string]string{
					"Upgrade":               "websocket",
					"Connection":            "Upgrade",
					"Sec-WebSocket-Key":     "dGhlIHNhbXBsZSBub25jZQ==",
					"Sec-WebSocket-Version": "13",
				},
				Timeout: 10 * time.Second,
			},
			Assertions: []models.Assertion{
				{Type: models.AssertStatusCode, Target: "status_code", Operator: "ne", Expected: 101},
			},
			Tags: []string{"websocket", "security", "auth"},
		})
	}

	_ = wsURL // Used for documentation/reference.

	return tests, nil
}

func isWebSocketEndpoint(ep models.Endpoint) bool {
	if ep.Method == "WS" || ep.Method == "WSS" {
		return true
	}
	for _, tag := range ep.Tags {
		if tag == "websocket" {
			return true
		}
	}
	return false
}

func toWSURL(httpURL string) string {
	if len(httpURL) > 5 && httpURL[:5] == "https" {
		return "wss" + httpURL[5:]
	}
	if len(httpURL) > 4 && httpURL[:4] == "http" {
		return "ws" + httpURL[4:]
	}
	return httpURL
}
