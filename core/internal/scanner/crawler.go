package scanner

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/gjson"
)

// Crawler discovers endpoints by following links in API responses.
type Crawler struct {
	baseURL    string
	maxDepth   int
	client     *http.Client
	authHeader string

	mu      sync.Mutex
	visited map[string]bool
}

// NewCrawler creates a new Crawler.
func NewCrawler(baseURL string, maxDepth int) *Crawler {
	return &Crawler{
		baseURL:  strings.TrimRight(baseURL, "/"),
		maxDepth: maxDepth,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		visited: make(map[string]bool),
	}
}

// SetAuth sets the authorization header for requests.
func (c *Crawler) SetAuth(header string) {
	c.authHeader = header
}

// Crawl performs a BFS crawl starting from baseURL.
// It extracts links from JSON responses (pagination, HATEOAS _links, href fields, etc.)
// and respects maxDepth to avoid unbounded crawling.
func (c *Crawler) Crawl(ctx context.Context) ([]string, error) {
	baseU, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, err
	}

	type queueItem struct {
		url   string
		depth int
	}

	var discovered []string
	queue := []queueItem{{url: c.baseURL, depth: 0}}
	c.markVisited(c.baseURL)

	for len(queue) > 0 {
		select {
		case <-ctx.Done():
			return discovered, ctx.Err()
		default:
		}

		item := queue[0]
		queue = queue[1:]

		if item.depth > c.maxDepth {
			continue
		}

		links, err := c.fetchAndExtractLinks(ctx, item.url, baseU)
		if err != nil {
			// Non-fatal: continue crawling other URLs
			continue
		}

		for _, link := range links {
			if c.tryMarkVisited(link) {
				discovered = append(discovered, link)
				if item.depth+1 <= c.maxDepth {
					queue = append(queue, queueItem{url: link, depth: item.depth + 1})
				}
			}
		}
	}

	return discovered, nil
}

// fetchAndExtractLinks fetches a URL and extracts links from the JSON response.
func (c *Crawler) fetchAndExtractLinks(ctx context.Context, targetURL string, baseU *url.URL) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if c.authHeader != "" {
		req.Header.Set("Authorization", c.authHeader)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, nil
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "json") {
		return nil, nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024)) // 2MB limit
	if err != nil {
		return nil, err
	}

	if !gjson.ValidBytes(body) {
		return nil, nil
	}

	var links []string
	seen := make(map[string]bool)

	addLink := func(raw string) {
		resolved := c.resolveURL(raw, baseU)
		if resolved == "" {
			return
		}
		if !seen[resolved] {
			seen[resolved] = true
			links = append(links, resolved)
		}
	}

	result := gjson.ParseBytes(body)

	// Extract HATEOAS _links (HAL format)
	// e.g., { "_links": { "self": { "href": "/foo" }, "next": { "href": "/bar" } } }
	result.Get("_links").ForEach(func(key, value gjson.Result) bool {
		href := value.Get("href")
		if href.Exists() {
			addLink(href.String())
		}
		return true
	})

	// Extract "links" array (JSON:API style)
	// e.g., { "links": { "next": "/page/2", "prev": "/page/1" } }
	linksObj := result.Get("links")
	if linksObj.IsObject() {
		linksObj.ForEach(func(key, value gjson.Result) bool {
			if value.Type == gjson.String {
				addLink(value.String())
			}
			return true
		})
	}
	if linksObj.IsArray() {
		linksObj.ForEach(func(_, value gjson.Result) bool {
			href := value.Get("href")
			if href.Exists() {
				addLink(href.String())
			}
			return true
		})
	}

	// Extract any "href", "url", or "uri" fields recursively
	c.extractURLFields(result, addLink)

	return links, nil
}

// extractURLFields recursively searches for href/url/uri fields in JSON.
func (c *Crawler) extractURLFields(result gjson.Result, addLink func(string)) {
	if result.IsObject() {
		result.ForEach(func(key, value gjson.Result) bool {
			k := strings.ToLower(key.String())
			if (k == "href" || k == "url" || k == "uri") && value.Type == gjson.String {
				addLink(value.String())
			} else if value.IsObject() || value.IsArray() {
				c.extractURLFields(value, addLink)
			}
			return true
		})
	} else if result.IsArray() {
		result.ForEach(func(_, value gjson.Result) bool {
			if value.IsObject() || value.IsArray() {
				c.extractURLFields(value, addLink)
			}
			return true
		})
	}
}

// resolveURL resolves a potentially relative URL against the base URL.
// Returns empty string if the URL is not on the same host.
func (c *Crawler) resolveURL(raw string, baseU *url.URL) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "#" {
		return ""
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}

	resolved := baseU.ResolveReference(parsed)

	// Only follow same-host URLs
	if resolved.Host != baseU.Host {
		return ""
	}

	// Normalize: strip fragment, ensure scheme
	resolved.Fragment = ""
	result := resolved.String()

	// Strip trailing slash for consistency (unless it's the root)
	if len(result) > 1 {
		result = strings.TrimRight(result, "/")
	}

	return result
}

// markVisited marks a URL as visited (not thread-safe, use tryMarkVisited for concurrent use).
func (c *Crawler) markVisited(u string) {
	c.mu.Lock()
	c.visited[u] = true
	c.mu.Unlock()
}

// tryMarkVisited atomically checks if a URL was visited and marks it.
// Returns true if the URL was NOT previously visited (i.e., it's new).
func (c *Crawler) tryMarkVisited(u string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.visited[u] {
		return false
	}
	c.visited[u] = true
	return true
}
