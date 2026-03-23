package test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mkaganm/probex/internal/generator"
	"github.com/mkaganm/probex/internal/models"
	"github.com/mkaganm/probex/internal/scanner"
)

func TestWebSocketDetection(t *testing.T) {
	// Mock server that responds to WebSocket upgrade.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ws" && r.Header.Get("Upgrade") == "websocket" {
			w.Header().Set("Upgrade", "websocket")
			w.Header().Set("Connection", "Upgrade")
			w.WriteHeader(101)
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	ws := scanner.NewWebSocketScanner(srv.URL)
	// Test single URL probe instead of full Discover (which probes 15 paths).
	ep, ok := ws.ProbeURL(context.Background(), "/ws")
	if !ok {
		t.Fatal("expected /ws WebSocket endpoint to be detected")
	}
	if ep.Method != "WS" {
		t.Errorf("expected WS method, got %s", ep.Method)
	}
	if ep.Path != "/ws" {
		t.Errorf("expected /ws path, got %s", ep.Path)
	}
}

func TestWebSocketAnalyze(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Upgrade") == "websocket" {
			w.Header().Set("Upgrade", "websocket")
			w.Header().Set("Sec-WebSocket-Protocol", "graphql-ws, subscriptions-transport-ws")
			w.WriteHeader(101)
			return
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()

	ws := scanner.NewWebSocketScanner(srv.URL)
	info := ws.AnalyzeWSEndpoint(context.Background(), "/ws")
	if info.Path != "/ws" {
		t.Errorf("expected /ws path, got %s", info.Path)
	}
}

func TestWebSocketGeneratorOutput(t *testing.T) {
	ep := models.Endpoint{
		Method:  "WS",
		Path:    "/ws",
		BaseURL: "http://localhost:8080",
		Tags:    []string{"websocket"},
	}

	gen := generator.NewWebSocket()
	tests, err := gen.Generate(ep)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if len(tests) < 5 {
		t.Errorf("expected at least 5 WebSocket tests, got %d", len(tests))
	}

	// Should have handshake test.
	found := false
	for _, tc := range tests {
		if strings.Contains(tc.Name, "handshake") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected handshake test")
	}

	// Should have CSWSH test.
	found = false
	for _, tc := range tests {
		if strings.Contains(tc.Name, "Origin") {
			found = true
			if tc.Category != models.CategorySecurity {
				t.Error("CSWSH test should be security category")
			}
			if tc.Severity != models.SeverityCritical {
				t.Error("CSWSH test should be critical severity")
			}
			break
		}
	}
	if !found {
		t.Error("expected Origin validation (CSWSH) test")
	}
}

func TestWebSocketGeneratorSkipsREST(t *testing.T) {
	ep := models.Endpoint{
		Method:  "GET",
		Path:    "/api/users",
		BaseURL: "http://localhost:8080",
	}

	gen := generator.NewWebSocket()
	tests, err := gen.Generate(ep)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(tests) != 0 {
		t.Errorf("expected 0 tests for REST endpoint, got %d", len(tests))
	}
}

func TestWebSocketGeneratorWithAuth(t *testing.T) {
	ep := models.Endpoint{
		Method:  "WS",
		Path:    "/ws/secure",
		BaseURL: "http://localhost:8080",
		Tags:    []string{"websocket"},
		Auth: &models.AuthInfo{
			Type:     models.AuthBearer,
			Location: "header",
			Key:      "Authorization",
		},
	}

	gen := generator.NewWebSocket()
	tests, err := gen.Generate(ep)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	// Should have auth-required test.
	found := false
	for _, tc := range tests {
		if strings.Contains(tc.Name, "authentication") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected authentication test for secured WebSocket endpoint")
	}
}
