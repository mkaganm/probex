package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mkaganm/probex/internal/models"
)

// ExternalPlugin communicates with an external plugin process via HTTP JSON-RPC.
// External plugins run as separate processes and expose a simple HTTP API.
// This allows plugins written in any language (Python, Node, Go, etc.).
type ExternalPlugin struct {
	meta       Metadata
	addr       string // e.g., "http://localhost:9720"
	httpClient *http.Client
}

// ExternalPluginConfig configures an external plugin connection.
type ExternalPluginConfig struct {
	Name    string `json:"name" yaml:"name"`
	Address string `json:"address" yaml:"address"`
}

// NewExternalPlugin connects to an external plugin at the given address.
func NewExternalPlugin(addr string) (*ExternalPlugin, error) {
	ep := &ExternalPlugin{
		addr: addr,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Fetch metadata.
	meta, err := ep.fetchMeta()
	if err != nil {
		return nil, fmt.Errorf("connect to plugin at %s: %w", addr, err)
	}
	ep.meta = *meta
	return ep, nil
}

func (ep *ExternalPlugin) fetchMeta() (*Metadata, error) {
	resp, err := ep.httpClient.Get(ep.addr + "/meta")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var meta Metadata
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

// Meta returns plugin metadata.
func (ep *ExternalPlugin) Meta() Metadata {
	return ep.meta
}

// Generate calls the external plugin's generate endpoint.
func (ep *ExternalPlugin) Generate(ctx context.Context, endpoint models.Endpoint) ([]models.TestCase, error) {
	body, err := json.Marshal(endpoint)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ep.addr+"/generate", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := ep.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("plugin returned status %d", resp.StatusCode)
	}

	var tests []models.TestCase
	if err := json.NewDecoder(resp.Body).Decode(&tests); err != nil {
		return nil, err
	}
	return tests, nil
}

// Report calls the external plugin's report endpoint.
func (ep *ExternalPlugin) Report(ctx context.Context, summary *models.RunSummary) ([]byte, error) {
	body, err := json.Marshal(summary)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ep.addr+"/report", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := ep.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("plugin returned status %d", resp.StatusCode)
	}

	var result json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

// FileExtension returns the file extension for external reporter plugins.
func (ep *ExternalPlugin) FileExtension() string {
	return ".json"
}

// Hook method stubs — external plugins support hooks via POST endpoints.

// BeforeScan calls the plugin's before-scan hook.
func (ep *ExternalPlugin) BeforeScan(ctx context.Context, baseURL string) error {
	return ep.callHook(ctx, "/hooks/before-scan", map[string]string{"base_url": baseURL})
}

// AfterScan calls the plugin's after-scan hook.
func (ep *ExternalPlugin) AfterScan(ctx context.Context, profile *models.APIProfile) error {
	return ep.callHook(ctx, "/hooks/after-scan", profile)
}

// BeforeRun calls the plugin's before-run hook.
func (ep *ExternalPlugin) BeforeRun(ctx context.Context, tests []models.TestCase) ([]models.TestCase, error) {
	body, err := json.Marshal(tests)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ep.addr+"/hooks/before-run", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := ep.httpClient.Do(req)
	if err != nil {
		// Non-fatal: return original tests.
		return tests, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		return tests, nil
	}

	var modified []models.TestCase
	if err := json.NewDecoder(resp.Body).Decode(&modified); err != nil {
		return tests, nil
	}
	return modified, nil
}

// AfterRun calls the plugin's after-run hook.
func (ep *ExternalPlugin) AfterRun(ctx context.Context, summary *models.RunSummary) error {
	return ep.callHook(ctx, "/hooks/after-run", summary)
}

func (ep *ExternalPlugin) callHook(ctx context.Context, path string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ep.addr+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := ep.httpClient.Do(req)
	if err != nil {
		// Non-fatal for hooks.
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("hook %s returned status %d", path, resp.StatusCode)
	}
	return nil
}
