package models

import "time"

// Endpoint represents a discovered API endpoint.
type Endpoint struct {
	ID           string            `json:"id" yaml:"id"`
	Method       string            `json:"method" yaml:"method"`
	Path         string            `json:"path" yaml:"path"`
	BaseURL      string            `json:"base_url" yaml:"base_url"`
	Headers      map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	QueryParams  []Parameter       `json:"query_params,omitempty" yaml:"query_params,omitempty"`
	PathParams   []Parameter       `json:"path_params,omitempty" yaml:"path_params,omitempty"`
	RequestBody  *Schema           `json:"request_body,omitempty" yaml:"request_body,omitempty"`
	Responses    []Response        `json:"responses,omitempty" yaml:"responses,omitempty"`
	Auth         *AuthInfo         `json:"auth,omitempty" yaml:"auth,omitempty"`
	Tags         []string          `json:"tags,omitempty" yaml:"tags,omitempty"`
	DiscoveredAt time.Time         `json:"discovered_at" yaml:"discovered_at"`
	Source       DiscoverySource   `json:"source" yaml:"source"`
}

// FullURL returns the complete URL for this endpoint.
func (e *Endpoint) FullURL() string {
	return e.BaseURL + e.Path
}

// Parameter represents a request parameter.
type Parameter struct {
	Name     string `json:"name" yaml:"name"`
	Type     string `json:"type" yaml:"type"`
	Required bool   `json:"required" yaml:"required"`
	Example  any    `json:"example,omitempty" yaml:"example,omitempty"`
}

// Response represents an observed API response.
type Response struct {
	StatusCode  int               `json:"status_code" yaml:"status_code"`
	ContentType string            `json:"content_type" yaml:"content_type"`
	Schema      *Schema           `json:"schema,omitempty" yaml:"schema,omitempty"`
	SampleBody  string            `json:"sample_body,omitempty" yaml:"sample_body,omitempty"`
	Headers     map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
}

// Schema represents an inferred JSON schema.
type Schema struct {
	Type       string             `json:"type" yaml:"type"`
	Properties map[string]*Schema `json:"properties,omitempty" yaml:"properties,omitempty"`
	Items      *Schema            `json:"items,omitempty" yaml:"items,omitempty"`
	Required   []string           `json:"required,omitempty" yaml:"required,omitempty"`
	Enum       []any              `json:"enum,omitempty" yaml:"enum,omitempty"`
	Pattern    string             `json:"pattern,omitempty" yaml:"pattern,omitempty"`
	Format     string             `json:"format,omitempty" yaml:"format,omitempty"`
	MinLength  *int               `json:"min_length,omitempty" yaml:"min_length,omitempty"`
	MaxLength  *int               `json:"max_length,omitempty" yaml:"max_length,omitempty"`
	Minimum    *float64           `json:"minimum,omitempty" yaml:"minimum,omitempty"`
	Maximum    *float64           `json:"maximum,omitempty" yaml:"maximum,omitempty"`
}

// AuthInfo describes the authentication requirements for an endpoint.
type AuthInfo struct {
	Type     AuthType `json:"type" yaml:"type"`
	Location string   `json:"location" yaml:"location"` // header, query, cookie
	Key      string   `json:"key" yaml:"key"`           // header name or query param name
}

// DiscoverySource indicates how an endpoint was discovered.
type DiscoverySource string

const (
	SourceOpenAPI    DiscoverySource = "openapi"
	SourceCrawl      DiscoverySource = "crawl"
	SourceWordlist   DiscoverySource = "wordlist"
	SourceTraffic    DiscoverySource = "traffic"
	SourceManual     DiscoverySource = "manual"
	SourceGraphQL    DiscoverySource = "graphql"
	SourceWebSocket  DiscoverySource = "websocket"
	SourceGRPC       DiscoverySource = "grpc"
	SourceIaC        DiscoverySource = "iac"
	SourceCollective DiscoverySource = "collective"
)

// AuthType represents the type of authentication.
type AuthType string

const (
	AuthNone   AuthType = "none"
	AuthBearer AuthType = "bearer"
	AuthAPIKey AuthType = "api_key"
	AuthBasic  AuthType = "basic"
	AuthOAuth2 AuthType = "oauth2"
)
