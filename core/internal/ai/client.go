package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client communicates with the Python AI brain FastAPI service.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new AI service client.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// Health calls GET /health on the Python brain.
func (c *Client) Health(ctx context.Context) (*HealthResponse, error) {
	var resp HealthResponse
	if err := c.doGet(ctx, "/health", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GenerateScenarios calls POST /api/v1/scenarios/generate.
func (c *Client) GenerateScenarios(ctx context.Context, req *ScenarioRequest) (*ScenarioResponse, error) {
	var resp ScenarioResponse
	if err := c.doPost(ctx, "/api/v1/scenarios/generate", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// AnalyzeSecurity calls POST /api/v1/security/analyze.
func (c *Client) AnalyzeSecurity(ctx context.Context, req *SecurityAnalysisRequest) (*SecurityAnalysisResponse, error) {
	var resp SecurityAnalysisResponse
	if err := c.doPost(ctx, "/api/v1/security/analyze", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// NLToTest calls POST /api/v1/nl-to-test.
func (c *Client) NLToTest(ctx context.Context, req *NLTestRequest) (*NLTestResponse, error) {
	var resp NLTestResponse
	if err := c.doPost(ctx, "/api/v1/nl-to-test", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ClassifyAnomaly calls POST /api/v1/anomaly/classify.
func (c *Client) ClassifyAnomaly(ctx context.Context, req *AnomalyClassifyRequest) (*AnomalyClassifyResponse, error) {
	var resp AnomalyClassifyResponse
	if err := c.doPost(ctx, "/api/v1/anomaly/classify", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// doGet performs a GET request and decodes the JSON response.
func (c *Client) doGet(ctx context.Context, path string, result any) error {
	url := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GET %s returned %d: %s", path, resp.StatusCode, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decoding response from GET %s: %w", path, err)
	}
	return nil
}

// doPost performs a POST request with a JSON body and decodes the JSON response.
func (c *Client) doPost(ctx context.Context, path string, body any, result any) error {
	url := c.baseURL + path

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encoding request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("POST %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("POST %s returned %d: %s", path, resp.StatusCode, string(respBody))
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decoding response from POST %s: %w", path, err)
	}
	return nil
}
