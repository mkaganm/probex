package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/probex/probex/internal/learn"
	"github.com/probex/probex/internal/models"
	"github.com/probex/probex/internal/schema"
)

// CapturedRequest holds a captured HTTP request/response pair.
type CapturedRequest struct {
	Method       string            `json:"method"`
	URL          string            `json:"url"`
	Path         string            `json:"path"`
	Headers      map[string]string `json:"headers"`
	RequestBody  string            `json:"request_body,omitempty"`
	StatusCode   int               `json:"status_code"`
	ResponseBody string            `json:"response_body,omitempty"`
	RespHeaders  map[string]string `json:"response_headers"`
	Duration     time.Duration     `json:"duration"`
	Timestamp    time.Time         `json:"timestamp"`
}

// Proxy is a reverse proxy that captures API traffic for learning.
type Proxy struct {
	listenAddr string
	targetURL  *url.URL
	proxy      *httputil.ReverseProxy
	server     *http.Server
	inferrer   *schema.Inferrer

	mu       sync.Mutex
	captures []CapturedRequest
	onEvent  func(CapturedRequest)
}

// Config holds proxy configuration.
type Config struct {
	ListenAddr string
	TargetURL  string
	OnEvent    func(CapturedRequest)
}

// New creates a new Proxy.
func New(cfg Config) (*Proxy, error) {
	target, err := url.Parse(cfg.TargetURL)
	if err != nil {
		return nil, fmt.Errorf("parse target URL: %w", err)
	}

	p := &Proxy{
		listenAddr: cfg.ListenAddr,
		targetURL:  target,
		inferrer:   schema.New(),
		onEvent:    cfg.OnEvent,
	}

	p.proxy = &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.Host = target.Host
		},
		ModifyResponse: p.captureResponse,
	}

	return p, nil
}

// Start begins the proxy server.
func (p *Proxy) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", p.handleProxy)

	p.server = &http.Server{
		Addr:    p.listenAddr,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		p.server.Shutdown(shutCtx)
	}()

	return p.server.ListenAndServe()
}

// Stop gracefully stops the proxy.
func (p *Proxy) Stop() error {
	if p.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return p.server.Shutdown(ctx)
	}
	return nil
}

// Captures returns all captured requests.
func (p *Proxy) Captures() []CapturedRequest {
	p.mu.Lock()
	defer p.mu.Unlock()
	result := make([]CapturedRequest, len(p.captures))
	copy(result, p.captures)
	return result
}

// CaptureCount returns the number of captured requests.
func (p *Proxy) CaptureCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.captures)
}

// ToHAREntries converts captures to HAR entries for the learn pipeline.
func (p *Proxy) ToHAREntries() []learn.Entry {
	p.mu.Lock()
	defer p.mu.Unlock()

	entries := make([]learn.Entry, 0, len(p.captures))
	for _, c := range p.captures {
		entry := learn.Entry{
			StartedDateTime: c.Timestamp.Format(time.RFC3339),
			Time:            float64(c.Duration.Milliseconds()),
			Request: learn.Request{
				Method: c.Method,
				URL:    c.URL,
			},
			Response: learn.Response{
				Status: c.StatusCode,
				Content: learn.Content{
					Text:     c.ResponseBody,
					MimeType: c.RespHeaders["content-type"],
				},
			},
			Timings: learn.Timings{
				Wait: float64(c.Duration.Milliseconds()),
			},
		}

		for name, value := range c.Headers {
			entry.Request.Headers = append(entry.Request.Headers, learn.Header{
				Name: name, Value: value,
			})
		}
		for name, value := range c.RespHeaders {
			entry.Response.Headers = append(entry.Response.Headers, learn.Header{
				Name: name, Value: value,
			})
		}

		if c.RequestBody != "" {
			entry.Request.PostData = &learn.PostData{
				MimeType: c.Headers["content-type"],
				Text:     c.RequestBody,
			}
		}

		entries = append(entries, entry)
	}
	return entries
}

// ToAPIProfile generates an API profile from captured traffic.
func (p *Proxy) ToAPIProfile() *models.APIProfile {
	captures := p.Captures()
	if len(captures) == 0 {
		return nil
	}

	// Group by method + normalized path.
	type epKey struct{ method, path string }
	grouped := make(map[epKey][]CapturedRequest)
	for _, c := range captures {
		parsed, _ := url.Parse(c.URL)
		path := "/"
		if parsed != nil {
			path = parsed.Path
		}
		k := epKey{c.Method, path}
		grouped[k] = append(grouped[k], c)
	}

	profile := &models.APIProfile{
		ID:        fmt.Sprintf("proxy-%d", time.Now().Unix()),
		Name:      fmt.Sprintf("Captured from %s", p.targetURL.String()),
		BaseURL:   p.targetURL.String(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	for k, caps := range grouped {
		ep := models.Endpoint{
			ID:           fmt.Sprintf("proxy-%s-%s", k.method, k.path),
			Method:       k.method,
			Path:         k.path,
			BaseURL:      p.targetURL.String(),
			DiscoveredAt: caps[0].Timestamp,
			Source:       models.SourceTraffic,
			Headers:      make(map[string]string),
		}

		// Infer response schema from first successful response.
		for _, c := range caps {
			if c.StatusCode >= 200 && c.StatusCode < 300 && c.ResponseBody != "" {
				s, err := p.inferrer.InferFromJSON([]byte(c.ResponseBody))
				if err == nil {
					ep.Responses = append(ep.Responses, models.Response{
						StatusCode: c.StatusCode,
						Schema:     s,
						SampleBody: truncate(c.ResponseBody, 4096),
					})
					break
				}
			}
		}

		// Infer request body schema.
		if k.method == "POST" || k.method == "PUT" || k.method == "PATCH" {
			for _, c := range caps {
				if c.RequestBody != "" {
					s, err := p.inferrer.InferFromJSON([]byte(c.RequestBody))
					if err == nil {
						ep.RequestBody = s
						break
					}
				}
			}
		}

		profile.Endpoints = append(profile.Endpoints, ep)
	}

	return profile
}

// ExportHAR exports captured traffic as HAR JSON.
func (p *Proxy) ExportHAR() ([]byte, error) {
	entries := p.ToHAREntries()
	har := learn.HarFile{
		Log: learn.Log{
			Version: "1.2",
			Creator: learn.Creator{Name: "probex-proxy", Version: "1.0.0"},
			Entries: entries,
		},
	}
	return json.MarshalIndent(har, "", "  ")
}

func (p *Proxy) handleProxy(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Read request body for capture.
	var reqBody string
	if r.Body != nil {
		bodyBytes, err := io.ReadAll(r.Body)
		if err == nil {
			reqBody = string(bodyBytes)
			r.Body = io.NopCloser(&readCloserFromBytes{data: bodyBytes})
		}
	}

	// Store request info in context for the response modifier.
	ctx := context.WithValue(r.Context(), ctxKeyStart, start)
	ctx = context.WithValue(ctx, ctxKeyReqBody, reqBody)
	ctx = context.WithValue(ctx, ctxKeyReqHeaders, copyHeaders(r.Header))
	r = r.WithContext(ctx)

	p.proxy.ServeHTTP(w, r)
}

func (p *Proxy) captureResponse(resp *http.Response) error {
	start, _ := resp.Request.Context().Value(ctxKeyStart).(time.Time)
	reqBody, _ := resp.Request.Context().Value(ctxKeyReqBody).(string)
	reqHeaders, _ := resp.Request.Context().Value(ctxKeyReqHeaders).(map[string]string)

	duration := time.Since(start)

	// Read response body (and restore it for the client).
	var respBody string
	if resp.Body != nil {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err == nil {
			respBody = string(bodyBytes)
			resp.Body = io.NopCloser(&readCloserFromBytes{data: bodyBytes})
		}
	}

	capture := CapturedRequest{
		Method:       resp.Request.Method,
		URL:          resp.Request.URL.String(),
		Path:         resp.Request.URL.Path,
		Headers:      reqHeaders,
		RequestBody:  reqBody,
		StatusCode:   resp.StatusCode,
		ResponseBody: respBody,
		RespHeaders:  copyHeaders(resp.Header),
		Duration:     duration,
		Timestamp:    time.Now(),
	}

	p.mu.Lock()
	p.captures = append(p.captures, capture)
	p.mu.Unlock()

	if p.onEvent != nil {
		p.onEvent(capture)
	}

	return nil
}

type contextKey string

const (
	ctxKeyStart      contextKey = "probex_start"
	ctxKeyReqBody    contextKey = "probex_req_body"
	ctxKeyReqHeaders contextKey = "probex_req_headers"
)

func copyHeaders(h http.Header) map[string]string {
	result := make(map[string]string, len(h))
	for k, v := range h {
		if len(v) > 0 {
			result[k] = v[0]
		}
	}
	return result
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// readCloserFromBytes allows re-reading body bytes.
type readCloserFromBytes struct {
	data []byte
	pos  int
}

func (r *readCloserFromBytes) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (r *readCloserFromBytes) Close() error {
	return nil
}
