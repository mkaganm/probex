package test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/mkaganm/probex/internal/models"
	"github.com/mkaganm/probex/internal/scanner"
)

// ---------------------------------------------------------------------------
// 1. OpenAPI Discovery
// ---------------------------------------------------------------------------

// minimalOpenAPISpec returns a minimal OpenAPI 3.0 spec with the given paths.
func minimalOpenAPISpec() map[string]any {
	return map[string]any{
		"openapi": "3.0.0",
		"info":    map[string]string{"title": "Test API", "version": "1.0"},
		"paths": map[string]any{
			"/pets": map[string]any{
				"get": map[string]any{
					"summary": "List pets",
					"parameters": []map[string]any{
						{
							"name":     "limit",
							"in":       "query",
							"required": false,
							"schema":   map[string]string{"type": "integer"},
						},
						{
							"name":     "species",
							"in":       "query",
							"required": true,
							"schema":   map[string]string{"type": "string"},
						},
					},
					"responses": map[string]any{
						"200": map[string]any{
							"description": "OK",
							"content": map[string]any{
								"application/json": map[string]any{
									"schema": map[string]any{
										"type":  "array",
										"items": map[string]any{"type": "object"},
									},
								},
							},
						},
					},
				},
			},
			"/pets/{petId}": map[string]any{
				"get": map[string]any{
					"summary": "Get pet by ID",
					"parameters": []map[string]any{
						{
							"name":     "petId",
							"in":       "path",
							"required": true,
							"schema":   map[string]string{"type": "string"},
						},
					},
					"responses": map[string]any{
						"200": map[string]any{"description": "OK"},
					},
				},
			},
			"/pets/{petId}/toys": map[string]any{
				"post": map[string]any{
					"summary": "Add a toy",
					"parameters": []map[string]any{
						{
							"name":     "petId",
							"in":       "path",
							"required": true,
							"schema":   map[string]string{"type": "string"},
						},
					},
					"requestBody": map[string]any{
						"required": true,
						"content": map[string]any{
							"application/json": map[string]any{
								"schema": map[string]any{
									"type": "object",
									"properties": map[string]any{
										"name":  map[string]string{"type": "string"},
										"color": map[string]string{"type": "string"},
									},
									"required": []string{"name"},
								},
							},
						},
					},
					"responses": map[string]any{
						"201": map[string]any{"description": "Created"},
					},
				},
			},
		},
	}
}

// newOpenAPIServer returns a httptest.Server that serves the OpenAPI spec at
// /openapi.json and returns 404 for everything else.
func newOpenAPIServer(spec map[string]any) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/openapi.json":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(spec)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestScannerOpenAPIDiscovery(t *testing.T) {
	spec := minimalOpenAPISpec()
	srv := newOpenAPIServer(spec)
	defer srv.Close()

	s := scanner.New(srv.URL, models.ScanOptions{Concurrency: 5})
	result, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}

	// Build lookup by method:path
	byKey := make(map[string]models.Endpoint, len(result.Endpoints))
	for _, ep := range result.Endpoints {
		byKey[ep.Method+":"+ep.Path] = ep
	}

	// 1a. Verify that all three spec endpoints were discovered.
	for _, want := range []string{"GET:/pets", "GET:/pets/{petId}", "POST:/pets/{petId}/toys"} {
		if _, ok := byKey[want]; !ok {
			t.Errorf("expected endpoint %s not found in results", want)
		}
	}

	// 1b. Verify query parameter extraction on GET /pets.
	if ep, ok := byKey["GET:/pets"]; ok {
		if len(ep.QueryParams) != 2 {
			t.Errorf("GET /pets: expected 2 query params, got %d", len(ep.QueryParams))
		}
		foundLimit := false
		foundSpecies := false
		for _, p := range ep.QueryParams {
			if p.Name == "limit" && p.Type == "integer" && !p.Required {
				foundLimit = true
			}
			if p.Name == "species" && p.Type == "string" && p.Required {
				foundSpecies = true
			}
		}
		if !foundLimit {
			t.Error("GET /pets: missing or incorrect 'limit' query param")
		}
		if !foundSpecies {
			t.Error("GET /pets: missing or incorrect 'species' query param")
		}
	}

	// 1c. Verify path parameter extraction on GET /pets/{petId}.
	if ep, ok := byKey["GET:/pets/{petId}"]; ok {
		if len(ep.PathParams) != 1 {
			t.Errorf("GET /pets/{petId}: expected 1 path param, got %d", len(ep.PathParams))
		} else if ep.PathParams[0].Name != "petId" || !ep.PathParams[0].Required {
			t.Errorf("GET /pets/{petId}: path param mismatch: %+v", ep.PathParams[0])
		}
	}

	// 1d. Verify request body detection on POST /pets/{petId}/toys.
	if ep, ok := byKey["POST:/pets/{petId}/toys"]; ok {
		if ep.RequestBody == nil {
			t.Error("POST /pets/{petId}/toys: expected non-nil RequestBody")
		} else {
			if ep.RequestBody.Type != "object" {
				t.Errorf("POST /pets/{petId}/toys: expected RequestBody type 'object', got %q", ep.RequestBody.Type)
			}
			if _, ok := ep.RequestBody.Properties["name"]; !ok {
				t.Error("POST /pets/{petId}/toys: RequestBody missing 'name' property")
			}
		}
	}

	// 1e. All OpenAPI endpoints should have SourceOpenAPI.
	for _, key := range []string{"GET:/pets", "GET:/pets/{petId}", "POST:/pets/{petId}/toys"} {
		if ep, ok := byKey[key]; ok {
			if ep.Source != models.SourceOpenAPI {
				t.Errorf("%s: expected source %q, got %q", key, models.SourceOpenAPI, ep.Source)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// 2. Scanner.Scan() full pipeline — deduplication & context cancellation
// ---------------------------------------------------------------------------

func TestScannerDeduplication(t *testing.T) {
	// The server serves an OpenAPI spec that defines GET /users AND responds
	// 200 to a GET /users probe (wordlist will also try /users).  We expect
	// the endpoint to appear only once.
	spec := map[string]any{
		"openapi": "3.0.0",
		"info":    map[string]string{"title": "Dedup API", "version": "1.0"},
		"paths": map[string]any{
			"/users": map[string]any{
				"get": map[string]any{
					"summary": "List users",
					"responses": map[string]any{
						"200": map[string]any{"description": "OK"},
					},
				},
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/openapi.json":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(spec)
		case "/users":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"id":1,"name":"Alice"}]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	s := scanner.New(srv.URL, models.ScanOptions{Concurrency: 5})
	result, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}

	// Count how many times GET:/users appears.
	count := 0
	for _, ep := range result.Endpoints {
		if ep.Method == "GET" && ep.Path == "/users" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected GET /users exactly once, found %d times", count)
	}
}

func TestScannerContextCancellation(t *testing.T) {
	// A slow server that blocks each request for a long time.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(30 * time.Second):
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())

	s := scanner.New(srv.URL, models.ScanOptions{
		Concurrency: 2,
		FollowLinks: false, // disable crawl to keep the test focused
		Timeout:     2 * time.Second,
	})

	done := make(chan struct{})
	var scanErr error
	go func() {
		_, scanErr = s.Scan(ctx)
		close(done)
	}()

	// Cancel immediately — the scan should not take long.
	cancel()

	select {
	case <-done:
		// Scan finished promptly — success.
		_ = scanErr // we don't require a specific error
	case <-time.After(5 * time.Second):
		t.Fatal("Scan did not return within 5 seconds after context cancellation")
	}
}

// ---------------------------------------------------------------------------
// 3. Crawler — HATEOAS _links, depth limiting, same-host filtering
// ---------------------------------------------------------------------------

func TestScannerCrawlerHATEOASLinks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/":
			// Root response with HATEOAS _links.
			_, _ = w.Write([]byte(`{
				"_links": {
					"self":    {"href": "/"},
					"users":   {"href": "/api/users"},
					"orders":  {"href": "/api/orders"}
				}
			}`))
		case "/api/users":
			_, _ = w.Write([]byte(`{
				"data": [{"id":1,"name":"Alice"}],
				"_links": {
					"self": {"href": "/api/users"},
					"next": {"href": "/api/users?page=2"}
				}
			}`))
		case "/api/orders":
			_, _ = w.Write([]byte(`{"data": []}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	s := scanner.New(srv.URL, models.ScanOptions{
		Concurrency: 5,
		FollowLinks: true,
		MaxDepth:    2,
	})

	result, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}

	// The crawler should have discovered /api/users and /api/orders from HATEOAS links.
	foundUsers := false
	foundOrders := false
	for _, ep := range result.Endpoints {
		if ep.Path == "/api/users" {
			foundUsers = true
		}
		if ep.Path == "/api/orders" {
			foundOrders = true
		}
	}
	if !foundUsers {
		t.Error("expected crawled endpoint /api/users from HATEOAS _links")
	}
	if !foundOrders {
		t.Error("expected crawled endpoint /api/orders from HATEOAS _links")
	}
}

func TestScannerCrawlerDepthLimit(t *testing.T) {
	// Build a chain: / -> /a -> /b -> /c -> /d.  With MaxDepth=2 the crawler
	// should stop before reaching /d.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/":
			_, _ = w.Write([]byte(`{"_links":{"next":{"href":"/a"}}}`))
		case "/a":
			_, _ = w.Write([]byte(`{"_links":{"next":{"href":"/b"}}}`))
		case "/b":
			_, _ = w.Write([]byte(`{"_links":{"next":{"href":"/c"}}}`))
		case "/c":
			_, _ = w.Write([]byte(`{"_links":{"next":{"href":"/d"}}}`))
		case "/d":
			_, _ = w.Write([]byte(`{"value":"deep"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	s := scanner.New(srv.URL, models.ScanOptions{
		Concurrency: 2,
		FollowLinks: true,
		MaxDepth:    2,
	})

	result, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}

	// /d should NOT be discovered via crawl (depth 0->/ , 1->/a, 2->/b, 3->/c would exceed).
	// Depending on the crawl BFS, /a and /b should be found, /c might be found at depth 2,
	// but /d at depth 3 should not.
	for _, ep := range result.Endpoints {
		if ep.Path == "/d" && ep.Source == models.SourceCrawl {
			t.Error("depth limiting failed: /d was discovered via crawl beyond max depth")
		}
	}
}

func TestScannerCrawlerSameHostFilter(t *testing.T) {
	// Response contains an external link — it must NOT be followed.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/":
			_, _ = w.Write([]byte(`{
				"_links": {
					"external": {"href": "https://evil.example.com/steal"},
					"internal": {"href": "/safe"}
				}
			}`))
		case "/safe":
			_, _ = w.Write([]byte(`{"ok": true}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	s := scanner.New(srv.URL, models.ScanOptions{
		Concurrency: 5,
		FollowLinks: true,
		MaxDepth:    2,
	})

	result, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}

	for _, ep := range result.Endpoints {
		if ep.Path == "/steal" {
			t.Error("same-host filter failed: external URL was followed")
		}
	}

	// /safe should be found.
	foundSafe := false
	for _, ep := range result.Endpoints {
		if ep.Path == "/safe" {
			foundSafe = true
		}
	}
	if !foundSafe {
		t.Error("expected /safe endpoint from internal link")
	}
}

// ---------------------------------------------------------------------------
// 4. Constructor tests
// ---------------------------------------------------------------------------

func TestScannerNewDefaults(t *testing.T) {
	s := scanner.New("https://api.example.com/", models.ScanOptions{})
	if s == nil {
		t.Fatal("expected non-nil Scanner from New()")
	}

	// Verify trailing slash is trimmed by running a scan that won't panic.
	// We can't inspect private fields, but we can verify the result's BaseURL.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	s2 := scanner.New(srv.URL+"/", models.ScanOptions{Concurrency: 1})
	result, err := s2.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}
	// BaseURL should have trailing slash stripped.
	if result.BaseURL != srv.URL {
		t.Errorf("expected BaseURL %q, got %q", srv.URL, result.BaseURL)
	}
}

func TestScannerSetAuth(t *testing.T) {
	var mu sync.Mutex
	var receivedAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		receivedAuth = r.Header.Get("Authorization")
		mu.Unlock()
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	s := scanner.New(srv.URL, models.ScanOptions{Concurrency: 1})
	s.SetAuth("Bearer test-token-123")
	_, _ = s.Scan(context.Background())

	mu.Lock()
	got := receivedAuth
	mu.Unlock()
	if got != "Bearer test-token-123" {
		t.Errorf("expected auth header 'Bearer test-token-123', got %q", got)
	}
}

func TestScannerSetHeaders(t *testing.T) {
	// Serve /health so the wordlist probe hits a 200 and the custom header is
	// sent through probeURL, which is the code path that applies SetHeaders.
	var receivedTenant string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if v := r.Header.Get("X-Tenant-ID"); v != "" {
			receivedTenant = v
		}
		switch r.URL.Path {
		case "/health":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	s := scanner.New(srv.URL, models.ScanOptions{Concurrency: 1})
	s.SetHeaders(map[string]string{
		"X-Tenant-ID": "acme-corp",
	})
	_, _ = s.Scan(context.Background())

	if receivedTenant != "acme-corp" {
		t.Errorf("expected X-Tenant-ID 'acme-corp', got %q", receivedTenant)
	}
}

func TestScannerResultSummary(t *testing.T) {
	srv := newOpenAPIServer(minimalOpenAPISpec())
	defer srv.Close()

	s := scanner.New(srv.URL, models.ScanOptions{Concurrency: 5})
	result, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}

	summary := result.Summary()
	if summary == "" {
		t.Error("expected non-empty summary")
	}
	if result.Duration <= 0 {
		t.Error("expected positive Duration")
	}
}
