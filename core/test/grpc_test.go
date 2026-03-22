package test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/probex/probex/internal/generator"
	"github.com/probex/probex/internal/models"
	"github.com/probex/probex/internal/scanner"
)

func TestGRPCDetection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/grpc.health.v1.Health/Check" && r.Method == "POST" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status":"SERVING"}`))
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	gs := scanner.NewGRPCScanner(srv.URL)
	if !gs.DetectGRPC(context.Background()) {
		t.Error("expected gRPC to be detected via health check")
	}
}

func TestGRPCDetectionNegative(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer srv.Close()

	gs := scanner.NewGRPCScanner(srv.URL)
	if gs.DetectGRPC(context.Background()) {
		t.Error("expected gRPC NOT to be detected")
	}
}

func TestGRPCReflection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "ServerReflection") {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"listServicesResponse": {
					"service": [
						{"name": "UserService"},
						{"name": "OrderService"},
						{"name": "grpc.reflection.v1alpha.ServerReflection"}
					]
				}
			}`))
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	gs := scanner.NewGRPCScanner(srv.URL)
	services, err := gs.DiscoverViaReflection(context.Background())
	if err != nil {
		t.Fatalf("reflection failed: %v", err)
	}

	// Should skip internal services.
	if len(services) != 2 {
		t.Errorf("expected 2 services, got %d", len(services))
	}

	names := make([]string, len(services))
	for i, s := range services {
		names[i] = s.Name
	}
	if names[0] != "UserService" {
		t.Errorf("expected UserService, got %s", names[0])
	}
}

func TestGRPCGeneratorOutput(t *testing.T) {
	ep := models.Endpoint{
		Method:  "GRPC",
		Path:    "/UserService/GetUser",
		BaseURL: "http://localhost:50051",
		Tags:    []string{"grpc", "unary"},
		Headers: map[string]string{
			"X-GRPC-Service":    "UserService",
			"X-GRPC-Method":     "GetUser",
			"X-GRPC-StreamType": "unary",
		},
	}

	gen := generator.NewGRPC()
	tests, err := gen.Generate(ep)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if len(tests) < 5 {
		t.Errorf("expected at least 5 gRPC tests, got %d", len(tests))
	}

	// Check for reflection security test.
	found := false
	for _, tc := range tests {
		if strings.Contains(tc.Name, "reflection") {
			found = true
			if tc.Category != models.CategorySecurity {
				t.Error("reflection test should be security category")
			}
			break
		}
	}
	if !found {
		t.Error("expected reflection security test")
	}

	// All tests should have grpc tag.
	for _, tc := range tests {
		hasGRPC := false
		for _, tag := range tc.Tags {
			if tag == "grpc" {
				hasGRPC = true
				break
			}
		}
		if !hasGRPC {
			t.Errorf("test %q missing grpc tag", tc.Name)
		}
	}
}

func TestGRPCGeneratorSkipsREST(t *testing.T) {
	ep := models.Endpoint{
		Method:  "GET",
		Path:    "/api/users",
		BaseURL: "http://localhost:8080",
	}

	gen := generator.NewGRPC()
	tests, err := gen.Generate(ep)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(tests) != 0 {
		t.Errorf("expected 0 tests for REST endpoint, got %d", len(tests))
	}
}

func TestGRPCGeneratorWithStreaming(t *testing.T) {
	ep := models.Endpoint{
		Method:  "GRPC",
		Path:    "/ChatService/StreamMessages",
		BaseURL: "http://localhost:50051",
		Tags:    []string{"grpc", "bidi-stream"},
		Headers: map[string]string{
			"X-GRPC-Service":    "ChatService",
			"X-GRPC-Method":     "StreamMessages",
			"X-GRPC-StreamType": "bidi-stream",
		},
	}

	gen := generator.NewGRPC()
	tests, err := gen.Generate(ep)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	// Should have streaming-specific test.
	found := false
	for _, tc := range tests {
		if strings.Contains(tc.Name, "streaming") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected streaming test for bidi-stream endpoint")
	}
}
