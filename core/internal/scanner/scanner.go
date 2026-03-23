package scanner

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/gjson"

	"github.com/mkaganm/probex/internal/models"
)

// Scanner discovers API endpoints from a target URL.
type Scanner struct {
	baseURL    string
	config     models.ScanOptions
	authHeader string
	headers    map[string]string
	client     *http.Client
}

// New creates a new Scanner.
func New(baseURL string, config models.ScanOptions) *Scanner {
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &Scanner{
		baseURL: strings.TrimRight(baseURL, "/"),
		config:  config,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// SetAuth sets the authorization header for requests.
func (s *Scanner) SetAuth(header string) {
	s.authHeader = header
}

// SetHeaders sets custom headers for requests.
func (s *Scanner) SetHeaders(headers map[string]string) {
	s.headers = headers
}

// Result holds the scan results.
type Result struct {
	Endpoints []models.Endpoint `json:"endpoints"`
	BaseURL   string            `json:"base_url"`
	Duration  time.Duration     `json:"duration"`
	Auth      *models.AuthInfo  `json:"auth,omitempty"`
}

// Scan performs endpoint discovery on the target URL.
// It combines OpenAPI spec parsing, link crawling, and wordlist probing.
func (s *Scanner) Scan(ctx context.Context) (*Result, error) {
	start := time.Now()

	result := &Result{
		BaseURL: s.baseURL,
	}

	// Track all discovered endpoints by method+path to deduplicate
	seen := make(map[string]models.Endpoint)

	// Phase 1: Try to discover OpenAPI/Swagger spec
	oaParser := NewOpenAPIParser(s.baseURL)
	if s.authHeader != "" {
		oaParser.SetAuth(s.authHeader)
	}

	oaEndpoints, err := oaParser.Discover(ctx)
	if err != nil {
		log.Printf("[scanner] OpenAPI discovery: %v", err)
	}
	for _, ep := range oaEndpoints {
		key := ep.Method + ":" + ep.Path
		seen[key] = ep
	}

	// Phase 2: Crawl base URL for links
	if s.config.FollowLinks {
		maxDepth := s.config.MaxDepth
		if maxDepth == 0 {
			maxDepth = 3
		}
		crawler := NewCrawler(s.baseURL, maxDepth)
		if s.authHeader != "" {
			crawler.SetAuth(s.authHeader)
		}

		crawledURLs, err := crawler.Crawl(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			log.Printf("[scanner] crawl: %v", err)
		}

		// Probe each crawled URL with GET
		crawlEndpoints := s.probeURLs(ctx, crawledURLs, models.SourceCrawl)
		for _, ep := range crawlEndpoints {
			key := ep.Method + ":" + ep.Path
			if _, exists := seen[key]; !exists {
				seen[key] = ep
			}
		}
	}

	// Phase 3: Probe common paths from wordlist
	wordlist := DefaultWordlist
	var probePaths []string
	for _, p := range wordlist {
		// Skip template paths like "/{id}"
		if strings.Contains(p, "{") {
			continue
		}
		probePaths = append(probePaths, s.baseURL+p)
	}

	wordlistEndpoints := s.probeURLs(ctx, probePaths, models.SourceWordlist)
	for _, ep := range wordlistEndpoints {
		key := ep.Method + ":" + ep.Path
		if _, exists := seen[key]; !exists {
			seen[key] = ep
		}
	}

	// Phase 4: GraphQL discovery
	gqlScanner := NewGraphQLScanner(s.baseURL)
	if s.authHeader != "" {
		gqlScanner.SetAuth(s.authHeader)
	}
	if gqlScanner.DetectGraphQL(ctx) {
		gqlEndpoints, err := gqlScanner.Discover(ctx)
		if err == nil {
			for _, ep := range gqlEndpoints {
				key := ep.Method + ":" + ep.Path + ":" + ep.Headers["X-GraphQL-Operation"]
				if _, exists := seen[key]; !exists {
					seen[key] = ep
				}
			}
		}
	}

	// Phase 5: WebSocket discovery
	wsScanner := NewWebSocketScanner(s.baseURL)
	if s.authHeader != "" {
		wsScanner.SetAuth(s.authHeader)
	}
	wsEndpoints := wsScanner.Discover(ctx)
	for _, ep := range wsEndpoints {
		key := ep.Method + ":" + ep.Path
		if _, exists := seen[key]; !exists {
			seen[key] = ep
		}
	}

	// Phase 6: gRPC discovery
	grpcScanner := NewGRPCScanner(s.baseURL)
	if s.authHeader != "" {
		grpcScanner.SetAuth(s.authHeader)
	}
	if grpcScanner.DetectGRPC(ctx) {
		grpcEndpoints := grpcScanner.Discover(ctx)
		for _, ep := range grpcEndpoints {
			key := ep.Method + ":" + ep.Path
			if _, exists := seen[key]; !exists {
				seen[key] = ep
			}
		}
	}

	// Phase 7: Detect auth requirements from collected endpoints
	result.Auth = s.detectAuth(seen)

	// Collect all endpoints
	endpoints := make([]models.Endpoint, 0, len(seen))
	for _, ep := range seen {
		// Apply global auth if endpoint doesn't have its own
		if ep.Auth == nil && result.Auth != nil {
			ep.Auth = result.Auth
		}
		endpoints = append(endpoints, ep)
	}

	result.Endpoints = endpoints
	result.Duration = time.Since(start)
	return result, nil
}

// probeURLs concurrently probes a list of URLs with GET requests.
// Returns endpoints for URLs that respond with a non-error status.
func (s *Scanner) probeURLs(ctx context.Context, urls []string, source models.DiscoverySource) []models.Endpoint {
	concurrency := s.config.Concurrency
	if concurrency <= 0 {
		concurrency = 10
	}

	sem := make(chan struct{}, concurrency)
	var mu sync.Mutex
	var endpoints []models.Endpoint
	var wg sync.WaitGroup

	for _, u := range urls {
		select {
		case <-ctx.Done():
			break
		default:
		}

		wg.Add(1)
		go func(targetURL string) {
			defer wg.Done()

			// Use select so goroutine doesn't block on semaphore after context cancel.
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}

			ep, ok := s.probeURL(ctx, targetURL, source)
			if ok {
				mu.Lock()
				endpoints = append(endpoints, ep)
				mu.Unlock()
			}
		}(u)
	}

	wg.Wait()
	return endpoints
}

// probeURL sends a GET request to a URL and constructs an endpoint if it responds.
func (s *Scanner) probeURL(ctx context.Context, targetURL string, source models.DiscoverySource) (models.Endpoint, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return models.Endpoint{}, false
	}
	req.Header.Set("Accept", "application/json, */*")
	if s.authHeader != "" {
		req.Header.Set("Authorization", s.authHeader)
	}
	for k, v := range s.headers {
		req.Header.Set(k, v)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return models.Endpoint{}, false
	}
	defer resp.Body.Close()

	// Only consider successful or auth-required responses as valid endpoints
	if resp.StatusCode >= 500 || resp.StatusCode == 404 {
		return models.Endpoint{}, false
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1*1024*1024)) // 1MB limit
	if err != nil {
		return models.Endpoint{}, false
	}

	// Extract path from URL
	path := extractPath(targetURL, s.baseURL)

	ep := models.Endpoint{
		ID:           endpointID("GET", path),
		Method:       "GET",
		Path:         path,
		BaseURL:      s.baseURL,
		DiscoveredAt: time.Now(),
		Source:       source,
	}

	// Build response info
	respInfo := models.Response{
		StatusCode:  resp.StatusCode,
		ContentType: resp.Header.Get("Content-Type"),
		Headers:     extractResponseHeaders(resp),
	}

	// Infer schema from response body
	if strings.Contains(respInfo.ContentType, "json") && len(body) > 0 && gjson.ValidBytes(body) {
		respInfo.Schema = inferSchema(body)
		// Keep a truncated sample
		if len(body) <= 1024 {
			respInfo.SampleBody = string(body)
		} else {
			respInfo.SampleBody = string(body[:1024]) + "..."
		}
	}

	ep.Responses = []models.Response{respInfo}

	// Detect endpoint-level auth from response
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		ep.Auth = detectAuthFromResponse(resp)
	}

	return ep, true
}

// detectAuth infers global auth requirements from the collected endpoints.
func (s *Scanner) detectAuth(endpoints map[string]models.Endpoint) *models.AuthInfo {
	authCount := 0
	var lastAuth *models.AuthInfo

	for _, ep := range endpoints {
		if ep.Auth != nil {
			authCount++
			lastAuth = ep.Auth
		}
		for _, resp := range ep.Responses {
			if resp.StatusCode == 401 || resp.StatusCode == 403 {
				authCount++
				if lastAuth == nil {
					lastAuth = detectAuthFromHeaders(resp.Headers)
				}
			}
		}
	}

	// If majority of endpoints require auth, treat it as global
	if authCount > 0 && lastAuth != nil {
		return lastAuth
	}
	return nil
}

// detectAuthFromResponse infers auth type from a 401/403 response.
func detectAuthFromResponse(resp *http.Response) *models.AuthInfo {
	wwwAuth := resp.Header.Get("WWW-Authenticate")
	return authFromWWWAuthenticate(wwwAuth)
}

// detectAuthFromHeaders infers auth type from response headers.
func detectAuthFromHeaders(headers map[string]string) *models.AuthInfo {
	wwwAuth, ok := headers["Www-Authenticate"]
	if !ok {
		wwwAuth = headers["WWW-Authenticate"]
	}
	return authFromWWWAuthenticate(wwwAuth)
}

// authFromWWWAuthenticate parses a WWW-Authenticate header value.
func authFromWWWAuthenticate(value string) *models.AuthInfo {
	if value == "" {
		return &models.AuthInfo{
			Type:     models.AuthAPIKey,
			Location: "header",
			Key:      "Authorization",
		}
	}
	lower := strings.ToLower(value)
	switch {
	case strings.HasPrefix(lower, "bearer"):
		return &models.AuthInfo{
			Type:     models.AuthBearer,
			Location: "header",
			Key:      "Authorization",
		}
	case strings.HasPrefix(lower, "basic"):
		return &models.AuthInfo{
			Type:     models.AuthBasic,
			Location: "header",
			Key:      "Authorization",
		}
	default:
		return &models.AuthInfo{
			Type:     models.AuthAPIKey,
			Location: "header",
			Key:      "Authorization",
		}
	}
}

// extractPath extracts the path portion from a full URL given the base URL.
func extractPath(fullURL, baseURL string) string {
	if strings.HasPrefix(fullURL, baseURL) {
		path := fullURL[len(baseURL):]
		if path == "" {
			return "/"
		}
		return path
	}
	// Fallback: parse the URL
	if idx := strings.Index(fullURL, "://"); idx != -1 {
		rest := fullURL[idx+3:]
		if slashIdx := strings.Index(rest, "/"); slashIdx != -1 {
			return rest[slashIdx:]
		}
	}
	return "/"
}

// extractResponseHeaders extracts relevant headers from an HTTP response.
func extractResponseHeaders(resp *http.Response) map[string]string {
	interesting := []string{
		"Content-Type", "WWW-Authenticate", "X-Request-Id",
		"X-RateLimit-Limit", "X-RateLimit-Remaining",
		"Allow", "Access-Control-Allow-Methods",
	}
	headers := make(map[string]string)
	for _, h := range interesting {
		v := resp.Header.Get(h)
		if v != "" {
			headers[h] = v
		}
	}
	return headers
}

// inferSchema infers a JSON schema from a response body.
func inferSchema(body []byte) *models.Schema {
	result := gjson.ParseBytes(body)
	return inferSchemaFromResult(result)
}

// inferSchemaFromResult recursively builds a schema from a gjson.Result.
func inferSchemaFromResult(r gjson.Result) *models.Schema {
	switch {
	case r.IsObject():
		schema := &models.Schema{
			Type:       "object",
			Properties: make(map[string]*models.Schema),
		}
		r.ForEach(func(key, value gjson.Result) bool {
			schema.Properties[key.String()] = inferSchemaFromResult(value)
			return true
		})
		return schema

	case r.IsArray():
		schema := &models.Schema{
			Type: "array",
		}
		// Infer items schema from first element
		arr := r.Array()
		if len(arr) > 0 {
			schema.Items = inferSchemaFromResult(arr[0])
		}
		return schema

	default:
		switch r.Type {
		case gjson.String:
			schema := &models.Schema{Type: "string"}
			// Detect common formats
			s := r.String()
			if looksLikeDateTime(s) {
				schema.Format = "date-time"
			} else if looksLikeEmail(s) {
				schema.Format = "email"
			} else if looksLikeUUID(s) {
				schema.Format = "uuid"
			}
			return schema
		case gjson.Number:
			if r.Float() == float64(r.Int()) && !strings.Contains(r.Raw, ".") {
				return &models.Schema{Type: "integer"}
			}
			return &models.Schema{Type: "number"}
		case gjson.True, gjson.False:
			return &models.Schema{Type: "boolean"}
		case gjson.Null:
			return &models.Schema{Type: "string"} // Nullable, default to string
		default:
			return &models.Schema{Type: "string"}
		}
	}
}

// looksLikeDateTime checks if a string looks like an ISO 8601 date-time.
func looksLikeDateTime(s string) bool {
	if len(s) < 10 {
		return false
	}
	// Simple heuristic: YYYY-MM-DD pattern
	if len(s) >= 10 && s[4] == '-' && s[7] == '-' {
		return true
	}
	return false
}

// looksLikeEmail checks if a string looks like an email.
func looksLikeEmail(s string) bool {
	return strings.Contains(s, "@") && strings.Contains(s, ".")
}

// looksLikeUUID checks if a string looks like a UUID.
func looksLikeUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	return s[8] == '-' && s[13] == '-' && s[18] == '-' && s[23] == '-'
}

// ScanResult returns a human-readable summary of the scan result.
func (r *Result) Summary() string {
	return fmt.Sprintf("Discovered %d endpoints at %s in %s",
		len(r.Endpoints), r.BaseURL, r.Duration.Round(time.Millisecond))
}
