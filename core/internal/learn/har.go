package learn

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/probex/probex/internal/models"
	"github.com/probex/probex/internal/schema"
)

// HAR 1.2 structs.

// HarFile is the top-level HAR document.
type HarFile struct {
	Log Log `json:"log"`
}

// Log is the root of the HAR data.
type Log struct {
	Version string  `json:"version"`
	Creator Creator `json:"creator"`
	Entries []Entry `json:"entries"`
}

// Creator describes the tool that generated the HAR.
type Creator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Entry represents a single HTTP transaction.
type Entry struct {
	StartedDateTime string   `json:"startedDateTime"`
	Time            float64  `json:"time"` // total elapsed time in ms
	Request         Request  `json:"request"`
	Response        Response `json:"response"`
	Timings         Timings  `json:"timings"`
}

// Request is the HTTP request within an entry.
type Request struct {
	Method      string        `json:"method"`
	URL         string        `json:"url"`
	HTTPVersion string        `json:"httpVersion"`
	Headers     []Header      `json:"headers"`
	QueryString []QueryString `json:"queryString"`
	PostData    *PostData     `json:"postData,omitempty"`
	HeadersSize int           `json:"headersSize"`
	BodySize    int           `json:"bodySize"`
}

// Response is the HTTP response within an entry.
type Response struct {
	Status      int       `json:"status"`
	StatusText  string    `json:"statusText"`
	HTTPVersion string    `json:"httpVersion"`
	Headers     []Header  `json:"headers"`
	Content     Content   `json:"content"`
	HeadersSize int       `json:"headersSize"`
	BodySize    int       `json:"bodySize"`
}

// Header is a name/value pair for HTTP headers.
type Header struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// QueryString is a name/value pair for URL query parameters.
type QueryString struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// PostData describes the request body.
type PostData struct {
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
}

// Content describes the response body.
type Content struct {
	Size     int    `json:"size"`
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
}

// Timings contains timing info for the entry.
type Timings struct {
	Blocked float64 `json:"blocked"`
	DNS     float64 `json:"dns"`
	Connect float64 `json:"connect"`
	Send    float64 `json:"send"`
	Wait    float64 `json:"wait"`
	Receive float64 `json:"receive"`
	SSL     float64 `json:"ssl"`
}

// TotalMillis returns the total time for the entry timings.
// Negative values (-1) mean the phase did not apply and are ignored.
func (t Timings) TotalMillis() float64 {
	var total float64
	for _, v := range []float64{t.Blocked, t.DNS, t.Connect, t.Send, t.Wait, t.Receive, t.SSL} {
		if v > 0 {
			total += v
		}
	}
	return total
}

// pathParamRegex matches common path segments that look like IDs.
var pathParamRegex = regexp.MustCompile(`/([0-9]+|[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})(?:/|$)`)

// EndpointKey is used to group HAR entries by method + normalized path.
type EndpointKey struct {
	Method string
	Path   string
}

// String returns a human-readable key.
func (k EndpointKey) String() string {
	return k.Method + " " + k.Path
}

// ParsedHAR holds the results of parsing one or more HAR files.
type ParsedHAR struct {
	// Endpoints extracted from the HAR entries.
	Endpoints []models.Endpoint
	// Grouped maps an EndpointKey to all matching raw HAR entries.
	Grouped map[EndpointKey][]Entry
	// Ordered is the full list of entries in chronological order.
	Ordered []Entry
}

// ParseHARFile reads and parses a single HAR file from disk.
func ParseHARFile(path string) (*ParsedHAR, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read HAR file: %w", err)
	}
	return ParseHARData(data)
}

// ParseHARData parses raw HAR JSON bytes.
func ParseHARData(data []byte) (*ParsedHAR, error) {
	var har HarFile
	if err := json.Unmarshal(data, &har); err != nil {
		return nil, fmt.Errorf("parse HAR JSON: %w", err)
	}

	grouped := make(map[EndpointKey][]Entry)
	for _, entry := range har.Log.Entries {
		key := entryToKey(entry)
		grouped[key] = append(grouped[key], entry)
	}

	inferrer := schema.New()
	var endpoints []models.Endpoint

	for key, entries := range grouped {
		ep := harEntriesToEndpoint(key, entries, inferrer)
		endpoints = append(endpoints, ep)
	}

	return &ParsedHAR{
		Endpoints: endpoints,
		Grouped:   grouped,
		Ordered:   har.Log.Entries,
	}, nil
}

// entryToKey extracts a normalized EndpointKey from a HAR entry.
func entryToKey(entry Entry) EndpointKey {
	parsed, err := url.Parse(entry.Request.URL)
	if err != nil {
		return EndpointKey{Method: entry.Request.Method, Path: entry.Request.URL}
	}
	normalized := normalizePath(parsed.Path)
	return EndpointKey{Method: entry.Request.Method, Path: normalized}
}

// normalizePath replaces path segments that look like IDs with parameter placeholders.
func normalizePath(path string) string {
	// Replace UUID segments.
	result := pathParamRegex.ReplaceAllStringFunc(path, func(match string) string {
		// Determine trailing separator.
		suffix := ""
		if strings.HasSuffix(match, "/") {
			suffix = "/"
		}
		return "/{id}" + suffix
	})
	return result
}

// harEntriesToEndpoint converts a group of HAR entries (same method+path) into a models.Endpoint.
func harEntriesToEndpoint(key EndpointKey, entries []Entry, inferrer *schema.Inferrer) models.Endpoint {
	first := entries[0]
	parsed, _ := url.Parse(first.Request.URL)

	baseURL := ""
	if parsed != nil {
		baseURL = parsed.Scheme + "://" + parsed.Host
	}

	ep := models.Endpoint{
		ID:           endpointID(key),
		Method:       key.Method,
		Path:         key.Path,
		BaseURL:      baseURL,
		Headers:      extractCommonHeaders(entries),
		QueryParams:  extractQueryParams(entries),
		DiscoveredAt: time.Now(),
		Source:       models.SourceTraffic,
	}

	// Infer request body schema from POST/PUT/PATCH bodies.
	if key.Method == "POST" || key.Method == "PUT" || key.Method == "PATCH" {
		ep.RequestBody = inferRequestBodySchema(entries, inferrer)
	}

	// Detect auth from headers.
	ep.Auth = detectAuth(entries)

	// Build responses with inferred schemas.
	ep.Responses = buildResponses(entries, inferrer)

	return ep
}

// endpointID generates a deterministic ID from method+path.
func endpointID(key EndpointKey) string {
	h := sha256.Sum256([]byte(key.String()))
	return fmt.Sprintf("%x", h[:8])
}

// extractCommonHeaders returns headers present in all entries (excluding well-known varying ones).
func extractCommonHeaders(entries []Entry) map[string]string {
	skip := map[string]bool{
		"cookie":         true,
		"authorization":  true,
		"host":           true,
		"content-length": true,
		"user-agent":     true,
		"accept":         true,
		"connection":     true,
		"cache-control":  true,
	}

	// Count occurrences of each header name -> value.
	type hv struct{ name, value string }
	counts := make(map[hv]int)
	for _, e := range entries {
		for _, h := range e.Request.Headers {
			lower := strings.ToLower(h.Name)
			if skip[lower] {
				continue
			}
			counts[hv{lower, h.Value}]++
		}
	}

	result := make(map[string]string)
	for kv, count := range counts {
		if count == len(entries) {
			result[kv.name] = kv.value
		}
	}
	return result
}

// extractQueryParams returns query parameters observed across entries.
func extractQueryParams(entries []Entry) []models.Parameter {
	seen := make(map[string]bool)
	var params []models.Parameter
	for _, e := range entries {
		for _, qs := range e.Request.QueryString {
			if seen[qs.Name] {
				continue
			}
			seen[qs.Name] = true
			params = append(params, models.Parameter{
				Name:    qs.Name,
				Type:    "string",
				Example: qs.Value,
			})
		}
	}
	// Mark params that appear in every entry as required.
	if len(entries) > 1 {
		nameCounts := make(map[string]int)
		for _, e := range entries {
			for _, qs := range e.Request.QueryString {
				nameCounts[qs.Name]++
			}
		}
		for i := range params {
			if nameCounts[params[i].Name] == len(entries) {
				params[i].Required = true
			}
		}
	}
	return params
}

// inferRequestBodySchema infers a merged schema from request bodies.
func inferRequestBodySchema(entries []Entry, inferrer *schema.Inferrer) *models.Schema {
	var schemas []*models.Schema
	for _, e := range entries {
		if e.Request.PostData == nil || e.Request.PostData.Text == "" {
			continue
		}
		s, err := inferrer.InferFromJSON([]byte(e.Request.PostData.Text))
		if err != nil {
			continue
		}
		schemas = append(schemas, s)
	}
	return inferrer.Merge(schemas)
}

// detectAuth detects auth from request headers.
func detectAuth(entries []Entry) *models.AuthInfo {
	if len(entries) == 0 {
		return nil
	}
	for _, h := range entries[0].Request.Headers {
		lower := strings.ToLower(h.Name)
		if lower == "authorization" {
			val := h.Value
			switch {
			case strings.HasPrefix(strings.ToLower(val), "bearer "):
				return &models.AuthInfo{Type: models.AuthBearer, Location: "header", Key: "Authorization"}
			case strings.HasPrefix(strings.ToLower(val), "basic "):
				return &models.AuthInfo{Type: models.AuthBasic, Location: "header", Key: "Authorization"}
			default:
				return &models.AuthInfo{Type: models.AuthAPIKey, Location: "header", Key: "Authorization"}
			}
		}
		if lower == "x-api-key" || lower == "api-key" {
			return &models.AuthInfo{Type: models.AuthAPIKey, Location: "header", Key: h.Name}
		}
	}
	return nil
}

// buildResponses groups entries by status code and infers response schemas.
func buildResponses(entries []Entry, inferrer *schema.Inferrer) []models.Response {
	byStatus := make(map[int][]Entry)
	for _, e := range entries {
		byStatus[e.Response.Status] = append(byStatus[e.Response.Status], e)
	}

	var responses []models.Response
	for status, group := range byStatus {
		resp := models.Response{
			StatusCode:  status,
			ContentType: group[0].Response.Content.MimeType,
			Headers:     make(map[string]string),
		}

		// Copy response headers from first entry.
		for _, h := range group[0].Response.Headers {
			resp.Headers[h.Name] = h.Value
		}

		// Save sample body.
		if group[0].Response.Content.Text != "" {
			body := group[0].Response.Content.Text
			if len(body) > 4096 {
				body = body[:4096]
			}
			resp.SampleBody = body
		}

		// Infer and merge response body schemas.
		var schemas []*models.Schema
		for _, e := range group {
			if e.Response.Content.Text == "" {
				continue
			}
			s, err := inferrer.InferFromJSON([]byte(e.Response.Content.Text))
			if err != nil {
				continue
			}
			schemas = append(schemas, s)
		}
		resp.Schema = inferrer.Merge(schemas)

		responses = append(responses, resp)
	}
	return responses
}
