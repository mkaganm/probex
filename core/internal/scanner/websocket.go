package scanner

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/probex/probex/internal/models"
)

// WebSocketScanner discovers WebSocket endpoints.
type WebSocketScanner struct {
	baseURL    string
	client     *http.Client
	authHeader string
}

// NewWebSocketScanner creates a new WebSocket scanner.
func NewWebSocketScanner(baseURL string) *WebSocketScanner {
	return &WebSocketScanner{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

// SetAuth sets the authorization header.
func (ws *WebSocketScanner) SetAuth(header string) {
	ws.authHeader = header
}

// Common WebSocket endpoint paths.
var wsCommonPaths = []string{
	"/ws",
	"/websocket",
	"/socket",
	"/socket.io",
	"/ws/v1",
	"/ws/v2",
	"/api/ws",
	"/api/websocket",
	"/realtime",
	"/live",
	"/stream",
	"/events",
	"/notifications",
	"/chat",
	"/feed",
}

// Discover probes common paths for WebSocket upgrade support.
func (ws *WebSocketScanner) Discover(ctx context.Context) []models.Endpoint {
	var endpoints []models.Endpoint

	for _, path := range wsCommonPaths {
		if ctx.Err() != nil {
			break
		}
		if ep, ok := ws.probeWebSocket(ctx, path); ok {
			endpoints = append(endpoints, ep)
		}
	}

	return endpoints
}

// ProbeURL checks if a specific URL supports WebSocket upgrade.
func (ws *WebSocketScanner) ProbeURL(ctx context.Context, path string) (models.Endpoint, bool) {
	return ws.probeWebSocket(ctx, path)
}

func (ws *WebSocketScanner) probeWebSocket(ctx context.Context, path string) (models.Endpoint, bool) {
	url := ws.baseURL + path

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return models.Endpoint{}, false
	}

	// Set WebSocket upgrade headers.
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Sec-WebSocket-Version", "13")
	if ws.authHeader != "" {
		req.Header.Set("Authorization", ws.authHeader)
	}

	resp, err := ws.client.Do(req)
	if err != nil {
		return models.Endpoint{}, false
	}

	// For 101 responses, close immediately — the connection is upgraded and
	// reading the body would block forever.
	if resp.StatusCode == 101 {
		resp.Body.Close()
	} else {
		defer func() {
			io.ReadAll(io.LimitReader(resp.Body, 1024))
			resp.Body.Close()
		}()
	}

	// WebSocket upgrade succeeds with 101 Switching Protocols.
	// Also check for 200/400 that indicate the endpoint exists but we didn't fully upgrade.
	isWS := resp.StatusCode == 101 ||
		(resp.Header.Get("Upgrade") != "" && strings.EqualFold(resp.Header.Get("Upgrade"), "websocket")) ||
		resp.StatusCode == 426 // Upgrade Required

	if !isWS {
		// Some WS endpoints return 400 "Can only upgrade to WebSocket"
		if resp.StatusCode == 400 {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
			bodyStr := strings.ToLower(string(body))
			if strings.Contains(bodyStr, "websocket") || strings.Contains(bodyStr, "upgrade") {
				isWS = true
			}
		}
	}

	if !isWS {
		return models.Endpoint{}, false
	}

	ep := models.Endpoint{
		ID:           endpointID("WS", path),
		Method:       "WS",
		Path:         path,
		BaseURL:      ws.baseURL,
		Tags:         []string{"websocket"},
		DiscoveredAt: time.Now(),
		Source:       models.SourceWordlist,
		Responses: []models.Response{
			{
				StatusCode:  resp.StatusCode,
				ContentType: "websocket",
				Headers: map[string]string{
					"Upgrade":    resp.Header.Get("Upgrade"),
					"Connection": resp.Header.Get("Connection"),
				},
			},
		},
	}

	// Check for auth requirements.
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		ep.Auth = detectAuthFromResponse(resp)
	}

	return ep, true
}

// DetectWebSocket checks if the base URL has any WebSocket endpoints.
func (ws *WebSocketScanner) DetectWebSocket(ctx context.Context) bool {
	// Quick check on most common paths.
	quickPaths := []string{"/ws", "/websocket", "/socket.io"}
	for _, path := range quickPaths {
		if _, ok := ws.probeWebSocket(ctx, path); ok {
			return true
		}
	}
	return false
}

// WSEndpointInfo holds additional WebSocket endpoint metadata.
type WSEndpointInfo struct {
	Path         string   `json:"path"`
	Protocols    []string `json:"protocols,omitempty"`
	RequiresAuth bool     `json:"requires_auth"`
	Description  string   `json:"description,omitempty"`
}

// AnalyzeWSEndpoint gathers additional info about a WebSocket endpoint.
func (ws *WebSocketScanner) AnalyzeWSEndpoint(ctx context.Context, path string) *WSEndpointInfo {
	info := &WSEndpointInfo{Path: path}

	url := ws.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return info
	}
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Sec-WebSocket-Version", "13")

	// Try without auth.
	resp, err := ws.client.Do(req)
	if err != nil {
		return info
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		info.RequiresAuth = true
	}

	// Check for supported protocols.
	if proto := resp.Header.Get("Sec-WebSocket-Protocol"); proto != "" {
		info.Protocols = strings.Split(proto, ",")
		for i := range info.Protocols {
			info.Protocols[i] = strings.TrimSpace(info.Protocols[i])
		}
	}

	// Infer description from path.
	switch {
	case strings.Contains(path, "chat"):
		info.Description = "Chat/messaging WebSocket"
	case strings.Contains(path, "notification") || strings.Contains(path, "event"):
		info.Description = "Event/notification stream"
	case strings.Contains(path, "stream") || strings.Contains(path, "feed"):
		info.Description = fmt.Sprintf("Data stream at %s", path)
	default:
		info.Description = fmt.Sprintf("WebSocket endpoint at %s", path)
	}

	return info
}
