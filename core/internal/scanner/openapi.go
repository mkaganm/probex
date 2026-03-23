package scanner

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/mkaganm/probex/internal/models"
	"gopkg.in/yaml.v3"
)

// OpenAPIParser attempts to discover and parse OpenAPI/Swagger specs.
type OpenAPIParser struct {
	baseURL    string
	client     *http.Client
	authHeader string
}

// NewOpenAPIParser creates a new OpenAPIParser.
func NewOpenAPIParser(baseURL string) *OpenAPIParser {
	return &OpenAPIParser{
		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SetAuth sets the authorization header for requests.
func (p *OpenAPIParser) SetAuth(header string) {
	p.authHeader = header
}

// commonSpecPaths are well-known paths where OpenAPI specs are often served.
var commonSpecPaths = []string{
	"/openapi.json",
	"/openapi.yaml",
	"/swagger.json",
	"/swagger.yaml",
	"/api-docs",
	"/api/docs",
	"/v1/openapi.json",
	"/v2/openapi.json",
	"/v3/openapi.json",
	"/docs/openapi.json",
	"/.well-known/openapi.json",
}

// specResult holds the result of probing a single spec path.
type specResult struct {
	body []byte
	err  error
}

// Discover attempts to find and parse an OpenAPI spec at the base URL.
// It returns the parsed endpoints or nil if no spec is found.
func (p *OpenAPIParser) Discover(ctx context.Context) ([]models.Endpoint, error) {
	specBody, err := p.findSpec(ctx)
	if err != nil {
		return nil, err
	}
	if specBody == nil {
		return nil, nil
	}

	return p.parseSpec(specBody)
}

// findSpec probes commonSpecPaths concurrently and returns the first valid spec body.
func (p *OpenAPIParser) findSpec(ctx context.Context) ([]byte, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type probeResult struct {
		body []byte
		err  error
	}

	results := make(chan probeResult, len(commonSpecPaths))
	var wg sync.WaitGroup

	for _, path := range commonSpecPaths {
		wg.Add(1)
		go func(specPath string) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			default:
			}

			url := p.baseURL + specPath
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				return
			}
			if p.authHeader != "" {
				req.Header.Set("Authorization", p.authHeader)
			}
			req.Header.Set("Accept", "application/json, application/yaml, text/yaml")

			resp, err := p.client.Do(req)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return
			}

			body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024)) // 10MB limit
			if err != nil {
				return
			}

			// Quick validation: check if it looks like an OpenAPI/Swagger doc
			if !looksLikeSpec(body) {
				return
			}

			results <- probeResult{body: body}
		}(path)
	}

	// Close results channel when all goroutines finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Return the first valid result
	for r := range results {
		if r.body != nil {
			cancel() // Cancel remaining probes
			return r.body, nil
		}
	}

	return nil, nil
}

// looksLikeSpec checks if the body looks like an OpenAPI or Swagger spec.
func looksLikeSpec(body []byte) bool {
	s := string(body)
	return strings.Contains(s, "openapi") ||
		strings.Contains(s, "swagger") ||
		strings.Contains(s, "paths")
}

// genericSpec is a flexible structure for parsing both OpenAPI 3.x and Swagger 2.x.
type genericSpec struct {
	OpenAPI  string                          `json:"openapi" yaml:"openapi"`
	Swagger  string                          `json:"swagger" yaml:"swagger"`
	Paths    map[string]map[string]*specOp   `json:"paths" yaml:"paths"`
	Servers  []specServer                    `json:"servers" yaml:"servers"`
	Host     string                          `json:"host" yaml:"host"`     // Swagger 2.x
	BasePath string                          `json:"basePath" yaml:"basePath"` // Swagger 2.x
	Security []map[string][]string           `json:"security" yaml:"security"`
	SecurityDefs map[string]securityScheme   `json:"securityDefinitions" yaml:"securityDefinitions"` // Swagger 2.x
	Components   *specComponents             `json:"components" yaml:"components"` // OpenAPI 3.x
}

type specServer struct {
	URL string `json:"url" yaml:"url"`
}

type specComponents struct {
	SecuritySchemes map[string]securityScheme `json:"securitySchemes" yaml:"securitySchemes"`
}

type securityScheme struct {
	Type string `json:"type" yaml:"type"`
	In   string `json:"in" yaml:"in"`
	Name string `json:"name" yaml:"name"`
}

type specOp struct {
	Summary     string              `json:"summary" yaml:"summary"`
	OperationID string              `json:"operationId" yaml:"operationId"`
	Tags        []string            `json:"tags" yaml:"tags"`
	Parameters  []specParam         `json:"parameters" yaml:"parameters"`
	RequestBody *specRequestBody    `json:"requestBody" yaml:"requestBody"`
	Responses   map[string]*specResp `json:"responses" yaml:"responses"`
	Security    []map[string][]string `json:"security" yaml:"security"`
}

type specParam struct {
	Name     string      `json:"name" yaml:"name"`
	In       string      `json:"in" yaml:"in"` // query, path, header, cookie
	Required bool        `json:"required" yaml:"required"`
	Schema   *specSchema `json:"schema" yaml:"schema"`
	Type     string      `json:"type" yaml:"type"` // Swagger 2.x
}

type specRequestBody struct {
	Required bool                     `json:"required" yaml:"required"`
	Content  map[string]*specMediaType `json:"content" yaml:"content"`
}

type specMediaType struct {
	Schema *specSchema `json:"schema" yaml:"schema"`
}

type specResp struct {
	Description string                    `json:"description" yaml:"description"`
	Content     map[string]*specMediaType `json:"content" yaml:"content"`
	Schema      *specSchema               `json:"schema" yaml:"schema"` // Swagger 2.x
}

type specSchema struct {
	Type       string                 `json:"type" yaml:"type"`
	Format     string                 `json:"format" yaml:"format"`
	Properties map[string]*specSchema `json:"properties" yaml:"properties"`
	Items      *specSchema            `json:"items" yaml:"items"`
	Required   []string               `json:"required" yaml:"required"`
	Enum       []any                  `json:"enum" yaml:"enum"`
}

// parseSpec parses an OpenAPI 3.x or Swagger 2.x spec and extracts endpoints.
func (p *OpenAPIParser) parseSpec(body []byte) ([]models.Endpoint, error) {
	var spec genericSpec

	// Try JSON first, then YAML
	if err := json.Unmarshal(body, &spec); err != nil {
		if err2 := yaml.Unmarshal(body, &spec); err2 != nil {
			return nil, fmt.Errorf("failed to parse spec as JSON or YAML: %w", err2)
		}
	}

	// Determine global auth
	globalAuth := p.extractGlobalAuth(&spec)

	var endpoints []models.Endpoint
	now := time.Now()

	for path, methods := range spec.Paths {
		for method, op := range methods {
			method = strings.ToUpper(method)
			// Skip non-HTTP method keys like "parameters", "summary", etc.
			if !isHTTPMethod(method) {
				continue
			}
			if op == nil {
				continue
			}

			ep := models.Endpoint{
				ID:           endpointID(method, path),
				Method:       method,
				Path:         path,
				BaseURL:      p.baseURL,
				Tags:         op.Tags,
				DiscoveredAt: now,
				Source:       models.SourceOpenAPI,
			}

			// Extract parameters
			for _, param := range op.Parameters {
				mp := models.Parameter{
					Name:     param.Name,
					Required: param.Required,
					Type:     paramType(param),
				}
				switch param.In {
				case "query":
					ep.QueryParams = append(ep.QueryParams, mp)
				case "path":
					ep.PathParams = append(ep.PathParams, mp)
				}
			}

			// Extract request body (OpenAPI 3.x)
			if op.RequestBody != nil {
				ep.RequestBody = extractRequestBodySchema(op.RequestBody)
			}

			// Extract responses
			for code, resp := range op.Responses {
				if resp == nil {
					continue
				}
				statusCode := parseStatusCode(code)
				r := models.Response{
					StatusCode: statusCode,
				}
				// OpenAPI 3.x: content map
				if resp.Content != nil {
					for ct, mt := range resp.Content {
						r.ContentType = ct
						if mt.Schema != nil {
							r.Schema = convertSchema(mt.Schema)
						}
						break // Take the first content type
					}
				}
				// Swagger 2.x: direct schema
				if resp.Schema != nil && r.Schema == nil {
					r.Schema = convertSchema(resp.Schema)
					r.ContentType = "application/json"
				}
				ep.Responses = append(ep.Responses, r)
			}

			// Auth: operation-level security overrides global
			if len(op.Security) > 0 {
				ep.Auth = p.resolveSecurityAuth(&spec, op.Security)
			} else if globalAuth != nil {
				ep.Auth = globalAuth
			}

			endpoints = append(endpoints, ep)
		}
	}

	return endpoints, nil
}

// extractGlobalAuth extracts authentication info from the spec's global security.
func (p *OpenAPIParser) extractGlobalAuth(spec *genericSpec) *models.AuthInfo {
	if len(spec.Security) == 0 {
		return nil
	}
	return p.resolveSecurityAuth(spec, spec.Security)
}

// resolveSecurityAuth resolves security requirements to an AuthInfo.
func (p *OpenAPIParser) resolveSecurityAuth(spec *genericSpec, security []map[string][]string) *models.AuthInfo {
	if len(security) == 0 {
		return nil
	}

	// Get security scheme definitions
	schemes := make(map[string]securityScheme)
	if spec.SecurityDefs != nil {
		schemes = spec.SecurityDefs
	}
	if spec.Components != nil && spec.Components.SecuritySchemes != nil {
		for k, v := range spec.Components.SecuritySchemes {
			schemes[k] = v
		}
	}

	// Take the first security requirement
	for schemeName := range security[0] {
		scheme, ok := schemes[schemeName]
		if !ok {
			continue
		}
		return securitySchemeToAuth(scheme)
	}

	return nil
}

// securitySchemeToAuth converts a security scheme definition to AuthInfo.
func securitySchemeToAuth(scheme securityScheme) *models.AuthInfo {
	switch strings.ToLower(scheme.Type) {
	case "http":
		return &models.AuthInfo{
			Type:     models.AuthBearer,
			Location: "header",
			Key:      "Authorization",
		}
	case "apikey":
		return &models.AuthInfo{
			Type:     models.AuthAPIKey,
			Location: scheme.In,
			Key:      scheme.Name,
		}
	case "oauth2":
		return &models.AuthInfo{
			Type:     models.AuthOAuth2,
			Location: "header",
			Key:      "Authorization",
		}
	case "basic":
		return &models.AuthInfo{
			Type:     models.AuthBasic,
			Location: "header",
			Key:      "Authorization",
		}
	default:
		return nil
	}
}

// convertSchema converts a spec schema to a models.Schema.
func convertSchema(s *specSchema) *models.Schema {
	if s == nil {
		return nil
	}
	ms := &models.Schema{
		Type:     s.Type,
		Format:   s.Format,
		Required: s.Required,
		Enum:     s.Enum,
	}
	if len(s.Properties) > 0 {
		ms.Properties = make(map[string]*models.Schema, len(s.Properties))
		for name, prop := range s.Properties {
			ms.Properties[name] = convertSchema(prop)
		}
	}
	if s.Items != nil {
		ms.Items = convertSchema(s.Items)
	}
	return ms
}

// extractRequestBodySchema extracts the schema from a request body.
func extractRequestBodySchema(rb *specRequestBody) *models.Schema {
	if rb == nil || rb.Content == nil {
		return nil
	}
	// Prefer application/json
	if mt, ok := rb.Content["application/json"]; ok && mt.Schema != nil {
		return convertSchema(mt.Schema)
	}
	// Fallback: first content type
	for _, mt := range rb.Content {
		if mt.Schema != nil {
			return convertSchema(mt.Schema)
		}
	}
	return nil
}

// paramType resolves the type string from a spec parameter.
func paramType(p specParam) string {
	if p.Schema != nil && p.Schema.Type != "" {
		return p.Schema.Type
	}
	if p.Type != "" {
		return p.Type
	}
	return "string"
}

// isHTTPMethod returns true if the string is a valid HTTP method.
func isHTTPMethod(m string) bool {
	switch m {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "TRACE":
		return true
	}
	return false
}

// endpointID generates a deterministic ID from method and path.
func endpointID(method, path string) string {
	h := sha256.Sum256([]byte(method + ":" + path))
	return fmt.Sprintf("%x", h[:8])
}

// parseStatusCode parses a status code string (e.g., "200", "default") to int.
func parseStatusCode(code string) int {
	var sc int
	if _, err := fmt.Sscanf(code, "%d", &sc); err != nil {
		return 0 // "default" or unparseable
	}
	return sc
}
